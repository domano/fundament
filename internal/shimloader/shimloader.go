package shimloader

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
)

//go:embed prebuilt/libFundamentShim.dylib
var shimBytes []byte

//go:embed prebuilt/manifest.json
var manifestBytes []byte

type manifest struct {
	SHA256       string `json:"sha256"`
	GeneratedAt  string `json:"generated_at"`
	SwiftVersion string `json:"swift_version"`
	SDKVersion   string `json:"sdk_version"`
}

var (
	initOnce sync.Once
	initErr  error

	dylibHandle uintptr
	dylibPath   string
	dylibHash   string
	meta        manifest
)

// Initialize extracts the embedded dylib onto disk (if necessary) and loads it.
func Initialize() error {
	initOnce.Do(func() {
		var err error
		meta, err = parseManifest(manifestBytes)
		if err != nil {
			initErr = fmt.Errorf("fundament: unable to parse shim manifest: %w", err)
			return
		}

		hash := sha256.Sum256(shimBytes)
		dylibHash = hex.EncodeToString(hash[:])
		if meta.SHA256 != "" && !equalIgnoreCase(dylibHash, meta.SHA256) {
			initErr = fmt.Errorf("fundament: embedded shim hash mismatch (have %s, want %s)", dylibHash, meta.SHA256)
			return
		}

		dylibPath, err = prepareDylib(dylibHash)
		if err != nil {
			initErr = err
			return
		}

		handle, err := purego.Dlopen(dylibPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			initErr = fmt.Errorf("fundament: failed to dlopen shim: %w", err)
			return
		}
		dylibHandle = handle
	})
	return initErr
}

// Handle returns the dylib handle. Initialize must succeed first.
func Handle() uintptr {
	return dylibHandle
}

// Path returns the path the shim was extracted to.
func Path() string {
	return dylibPath
}

// Hash returns the SHA256 for the embedded shim.
func Hash() string {
	return dylibHash
}

// Register binds the exported symbol to the provided Go function pointer.
func Register(symbol string, fptr interface{}) error {
	if err := Initialize(); err != nil {
		return err
	}
	addr, err := purego.Dlsym(dylibHandle, symbol)
	if err != nil {
		return fmt.Errorf("fundament: failed to resolve %q: %w", symbol, err)
	}
	purego.RegisterFunc(fptr, addr)
	return nil
}

func parseManifest(data []byte) (manifest, error) {
	var m manifest
	if len(data) == 0 {
		return m, errors.New("manifest is empty")
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return manifest{}, err
	}
	return m, nil
}

func prepareDylib(hash string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil || cacheDir == "" {
		cacheDir = os.TempDir()
	}

	targetDir := filepath.Join(cacheDir, "fundament-shim", hash)
	targetPath := filepath.Join(targetDir, "libFundamentShim.dylib")

	if ok := verifyExisting(targetPath, hash); ok {
		return targetPath, nil
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", fmt.Errorf("fundament: failed to make cache dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(targetDir, "libFundamentShim.*.tmp")
	if err != nil {
		return "", fmt.Errorf("fundament: failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	if _, err := tmpFile.Write(shimBytes); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("fundament: failed to write shim: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("fundament: failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("fundament: failed to atomically move shim: %w", err)
	}

	adHocCodesign(targetPath)

	return targetPath, nil
}

func verifyExisting(path, expectedHash string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	hash := sha256.Sum256(data)
	return equalIgnoreCase(expectedHash, hex.EncodeToString(hash[:]))
}

func adHocCodesign(path string) {
	if runtime.GOOS != "darwin" {
		return
	}
	if _, err := exec.LookPath("codesign"); err != nil {
		return
	}
	cmd := exec.Command("codesign", "--force", "--sign", "-", path)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("fundament: warning: ad-hoc codesign failed for %s: %v (output: %s)", path, err, string(out))
	}
}

func equalIgnoreCase(a, b string) bool {
	return len(a) == len(b) && strings.EqualFold(a, b)
}

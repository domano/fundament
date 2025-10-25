# Getting Started

- Fundament bridges the macOS 26 `SystemLanguageModel` APIs into Go. It requires:

- macOS 26 (Sequoia) or newer with Apple Intelligence entitlement.
- Xcode 16 (or the matching Command Line Tools) so the `FoundationModels` framework is available to `swift build`. Set `DEVELOPER_DIR` if you rely on a beta Xcode bundle.
- Go 1.25 or newer (cgo is optional).

## Build the Swift shim

```bash
make swift
```

The Makefile pins `MACOSX_DEPLOYMENT_TARGET=15.0`, redirects the SwiftPM module cache under `.swift-module-cache`, disables the package sandbox (required when building inside limited environments), and emits the release build into `swift/FundamentShim/.build/Release`. After the build finishes, `scripts/package_shim.sh` atomically updates `internal/shimloader/prebuilt/libFundamentShim.dylib` and its `manifest.json` (hash, Swift version, SDK metadata). Although the manifest advertises `.macOS(.v14)` for compatibility with SwiftPM 5.10, the shim itself checks at runtime and refuses to load on systems earlier than macOS 26.

## Compile the Go module

```bash
make go
```

The Go bindings embed the committed dylib and extract it into the user cache on first use, so you can `go build` or `go run` without managing `DYLD_LIBRARY_PATH` or copying artefacts.

## Running examples

Three examples showcase the core features:

- `examples/simple`: single-turn text generation.
- `examples/structured`: schema-guided JSON output using dynamic generation schemas.
- `examples/streaming`: token-level streaming over Go channels.

Build them with:

```bash
make examples
```

Locally built binaries resolve the shim automatically thanks to runtime extraction. Distribution bundles should include the Go binary only; the shim is embedded and cached at runtime under `~/Library/Caches/fundament-shim/<sha>/` (or `$XDG_CACHE_HOME`).

## Troubleshooting

- **Unavailable model**: Check `fundament.CheckAvailability()` to surface entitlement or readiness issues. The helper maps `SystemLanguageModel.Availability` reasons to Go enums and will return `AvailabilityUnavailable` with an appropriate reason when macOS 26 features are disabled.
- **Missing framework**: Ensure you can compile a Swift file that imports `FoundationModels`. Xcode beta builds might install the framework under `/Applications/Xcode-beta.app`.
- **Shim loading failures**: Clear `~/Library/Caches/fundament-shim` and rerun. If the hash mismatch persists, re-run `make swift` so the committed `internal/shimloader/prebuilt` artefacts match the embedded bytes. Hardened/notarized apps may need ad-hoc codesigning support (ensure Xcode Command Line Tools are installed).

For deeper diagnostics, re-run `swift build -Xswiftc -v` to inspect linker flags or open the package in Xcode for debugging.

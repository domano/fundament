# Toolchain & Commands

## Prerequisites

- **macOS 26** with Apple Intelligence entitlement enabled.  
- **Xcode 16 or newer** (Command Line Tools installed) to provide the `FoundationModels` framework.  
- **Go 1.25+** with cgo enabled.

## Build steps

1. **Compile the Swift shim**
   ```bash
   make swift
   ```
   - Sets `MACOSX_DEPLOYMENT_TARGET=15.0`, disables SwiftPM sandboxing, and writes build products to `swift/FundamentShim/.build/Release`.
   - Produces `libFundamentShim.dylib`, which Go links against (an `rpath` is embedded via cgo flags).

2. **Build Go packages and examples**
   ```bash
   make go
   ```
   - Runs `go build ./...` (remember to set `GOCACHE`/`GOMODCACHE` if working inside a sandbox).
   - Linking relies on the release shim artefact created in step 1.

3. **Optional: build example binaries explicitly**
   ```bash
   make examples
   ```
   This compiles `examples/simple`, `examples/structured`, and `examples/streaming`.

## Testing

- Run `make test` (or `go test ./...`) for unit coverage.  
- On macOS 26 hardware with Apple Intelligence enabled, run `make integration` (which sets the `integration` build tag) to exercise the live Swift bridge against `SystemLanguageModel`. Integration failures usually indicate entitlement, availability, or ABI drift—fix the root cause rather than skipping the tests.

## Environment tips

- When running binaries, ensure `DYLD_LIBRARY_PATH` includes `swift/FundamentShim/.build/Release` so the dynamic loader can find `libFundamentShim.dylib`.  
- Clear local caches with `rm -rf .swift-module-cache .swift-build-cache .gocache .gomodcache` after experiments to avoid stale artefacts.

Related docs: [Troubleshooting](troubleshooting.md)

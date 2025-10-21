# Getting Started

- Fundament bridges the macOS 26 `SystemLanguageModel` APIs into Go. It requires:

- macOS 26 (Sequoia) or newer with Apple Intelligence entitlement.
- Xcode 16 (or the matching Command Line Tools) so the `FoundationModels` framework is available to `swift build`. Set `DEVELOPER_DIR` if you rely on a beta Xcode bundle.
- Go 1.25 or newer with `CGO_ENABLED=1`.

## Build the Swift shim

```bash
make swift
```

The Makefile pins `MACOSX_DEPLOYMENT_TARGET=15.0`, redirects the SwiftPM module cache under `.swift-module-cache`, disables the package sandbox (required when building inside limited environments), and emits the release build into `swift/FundamentShim/.build/Release`. Although the manifest advertises `.macOS(.v14)` for compatibility with SwiftPM 5.10, the shim itself checks at runtime and will refuse to load on systems earlier than macOS 26.

## Compile the Go module

```bash
make go
```

The cgo directives include both the Release and Debug build directories, so if you iterate quickly you can run `make swift-debug` to build a debug lib and still build Go code without relinking.

## Running examples

Three examples showcase the core features:

- `examples/simple`: single-turn text generation.
- `examples/structured`: schema-guided JSON output using dynamic generation schemas.
- `examples/streaming`: token-level streaming over Go channels.

Build them with:

```bash
make examples
```

Set `DYLD_LIBRARY_PATH=swift/FundamentShim/.build/Release` (or add it to your shell profile) before running the binaries so the dynamic loader resolves the shim.

## Troubleshooting

- **Unavailable model**: Check `fundament.CheckAvailability()` to surface entitlement or readiness issues. The helper maps `SystemLanguageModel.Availability` reasons to Go enums and will return `AvailabilityUnavailable` with an appropriate reason when macOS 26 features are disabled.
- **Missing framework**: Ensure you can compile a Swift file that imports `FoundationModels`. Xcode beta builds might install the framework under `/Applications/Xcode-beta.app`.
- **Linker failures**: If Go cannot locate `libFundamentShim.dylib`, verify that `swift build` succeeded and that you are targeting the Release configuration prior to linking.

For deeper diagnostics, re-run `swift build -Xswiftc -v` to inspect linker flags or open the package in Xcode for debugging.

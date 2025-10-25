# Troubleshooting

## SwiftPM manifest or cache permission errors

- Ensure SwiftPM has write access; the Makefile already disables the sandbox and redirects caches under the repo.  
- If the build still fails, run `rm -rf .swift-module-cache .swift-build-cache` and retry `make swift`.

## Missing `FoundationModels` / availability failures

- The shim checks `SystemLanguageModel` availability; on unsupported hardware or macOS versions < 26 it returns a structured error.  
- Verify the machine is Apple Intelligence–eligible and logged into the correct Apple ID with the entitlement enabled.

## Shim extraction or loading failures

- Confirm you ran `make swift` so `internal/shimloader/prebuilt/libFundamentShim.dylib` matches the manifest hash.  
- Clear the cache directory (`rm -rf ~/Library/Caches/fundament-shim`) and rerun the binary to force re-extraction.  
- Hardened/notarized apps may need access to `codesign` for ad-hoc signing; ensure Xcode Command Line Tools are installed.

## Structured generation inaccuracies

- The JSON schema translator currently supports objects, arrays, enums (`anyOf` strings), and primitive string/int/bool types.  
- Complex schema features (references, numeric guides) are not yet implemented. Update both the Go schema builder and `buildDynamicSchema` if you need more coverage.  
- See [Decision: Schema Support](../../decisions/notes/schema_support.md) for details.

## Streaming delivers whole sentences

- The shim collects the stream and splits on spaces before invoking the callback.  
- To provide real-time token snapshots, the stream bridging code must be extended to forward partial snapshots instead of aggregating them.  
- Track this work in the interop layer; see [Architecture](../../product/design/architecture.md).

# Agent Handbook

Welcome! This project is maintained with multiple automated contributors in mind.  
Review the guidelines below before you begin making changes.

## 1. Start with the context directory

The repo includes a three-tier knowledge base under [`context/`](context/README.md):

- **Tier 1:** high-level overview (`context/README.md`).  
- **Tier 2:** category summaries (e.g. [`context/product/README.md`](context/product/README.md)).  
- **Tier 3:** deep dives (e.g. [`context/product/design/architecture.md`](context/product/design/architecture.md), [`context/operations/build/toolchain.md`](context/operations/build/toolchain.md)).

Always read the relevant Tier‑2 page and any linked Tier‑3 notes before editing code or build scripts. This ensures that new work aligns with the existing architecture and decisions.

## 2. Build & runtime expectations

- **Target platform:** macOS 26 with Apple Intelligence entitlement enabled.  
- Run `make swift` to build the Swift shim, then `make go` (or `go build ./...`) for Go components.  
- After modifying anything under `swift/FundamentShim/`, run `make swift`; it rebuilds the shim and updates `internal/shimloader/prebuilt/libFundamentShim.dylib` plus its manifest so the embedded artefact stays current.  
- Run `make test` (or `go test ./...`) to execute the Go unit tests before you ship changes.  
- On macOS 26 hardware with Apple Intelligence, run `make integration` (enables the `integration` build tag) to exercise the live Swift bridge; fix environment issues instead of skipping these checks.  
- Importing the Go module is enough—the shim is embedded and extracted on demand. Built binaries cache the dylib under the user cache directory; no rpath or environment tweaks are required.

See [`context/operations/build/toolchain.md`](context/operations/build/toolchain.md) for precise commands and troubleshooting tips.

## 3. Coding conventions

- Respect the existing layering: Go API ➝ `internal/native` ➝ Swift shim.  
- Maintain error propagation across the C boundary (non-zero error structs must be freed).  
- For structured generation, consult [`context/decisions/notes/schema_support.md`](context/decisions/notes/schema_support.md) before extending JSON schema handling.

## 4. Communicate assumptions

When making significant changes, update the appropriate Markdown in `context/` or leave an inline TODO referencing a context note. This keeps future agents aligned.

## 5. Validation checklist

Before handing off work:

1. `make swift` (or `swift build …`) succeeds without new errors, and the refreshed `internal/shimloader/prebuilt/libFundamentShim.dylib` plus `manifest.json` are committed if Swift sources changed.  
2. `go build ./...` succeeds.  
3. `go test ./...` (or `make test`) succeeds. If tests fail because of intended behaviour changes, update the tests alongside the code—never delete or comment them out just to bypass a failure.  
4. `make integration` succeeds on an entitled macOS 26 machine. The target enables the `integration` build tag—address availability or entitlement issues rather than suppressing the integration tests.  
5. Any new behaviour is documented in `context/` if it affects architecture, operations, or decisions.

Thanks for contributing responsibly!

# Implementation Plan: Go bindings for macOS Foundation LLMs

## Scope and Targets

- Deliver a pure Go package (module `github.com/domano/fundament`) that exposes ergonomic APIs for the on-device `SystemLanguageModel` introduced in macOS Sequoia (a.k.a. macOS 15 / “macOS 26”) via cgo-backed Swift interop.
- Provide runnable examples demonstrating:
  - Single-turn prompt/response.
  - Structured output generation with `@Generable` schemas.
  - Streaming partial completions.
- Ensure the library gracefully handles availability and entitlement checks so callers can detect when Apple Intelligence is unavailable (`SystemLanguageModel.availability`).  
  _Key docs_: Apple Developer Documentation for Foundation Models (`LanguageModelSession`, `SystemLanguageModel`, `LanguageModelFeedback`) retrieved via Context7; Swift interop proposals for the `@c` attribute and pointer interoperability; Swift concurrency continuations for bridging async Swift APIs to callback-based C; Go `runtime/cgo` handle utilities.

## Reference Notes from Research

- **Foundation Models API surface**: `LanguageModelSession` creation, prompt/response helpers, streaming interface, guardrails/adapters, locale support, availability checks, and feedback APIs ([Apple Developer Documentation – Generating content and performing tasks with Foundation Models](https://developer.apple.com/documentation/FoundationModels/generating-content-and-performing-tasks-with-foundation-models); `SystemLanguageModel` API overview; `LanguageModelSession` respond variants; guardrails/adapter snippets).
- **Adaptation hooks**: `SystemLanguageModel.Adapter` initialisers, download checks, compilation, and guardrails for customizing model behaviour (Apple docs on loading and using a custom adapter).
- **Safety/logging**: `LanguageModelFeedback` utilities for transcript capture and safety reporting (Apple docs on improving the safety of generative model output).
- **Swift-to-C ABI guidance**: Swift proposals for the `@c` attribute and `@convention(c)` exports, POD requirements, and pointer bridging (`AttrC.rst`, `CPointerInteropLanguageModel.rst`, `HowSwiftImportsCAPIs.md` from the Swift open-source repo).
- **Async bridging helpers**: Swift `CheckedContinuation` and `UnsafeContinuation` to funnel async `LanguageModelSession.respond` calls into callback-oriented exports.
- **Go-side interop**: `runtime/cgo` handles for passing Go references through C, `cgo` directives for linking against Swift-produced dynamic/static libs. (Go runtime docs surfaced via Context7).

## High-Level Architecture

1. **Swift shim** (`Sources/FundamentShim`):
   - Swift Package that imports `FoundationModels`.
   - Exposes a thin C-compatible surface via functions annotated `@_cdecl` / `@c`, returning opaque handles (Swift class references retained across the boundary).
   - Manages async Swift APIs by wrapping them in helper types that synchronously rendezvous with callers using `withCheckedContinuation` or dispatch queues.
   - Emits a stable `.modulemap` and headers for cgo consumption.
   - Builds as a dynamic library (`.dylib`) or static archive (`.a`) depending on `swift build` options; initial plan prefers `.dylib` to simplify ABI stability.

2. **Go binding layer** (`internal/native`):
   - Uses cgo to include the generated headers and link against the Swift artefact.
   - Wraps raw C calls with idiomatic Go types (`Session`, `GenerationOptions`, `Stream`), handling memory management with `runtime.SetFinalizer` and `runtime/cgo.NewHandle` where needed.
   - Provides error translation (convert Swift `NSError` or custom codes into Go `error` values).

3. **Public Go API** (`fundament`):
   - Surface-level functions for:
     - `NewSession(options SessionOptions)`.
     - `Respond(ctx, prompt)` returning `Response`.
     - `RespondStructured` for schema-driven generation.
     - `RespondStream` returning a Go channel / iterator for streaming tokens, backed by callbacks from Swift.
     - Availability detection (`CheckAvailability()`), locale helpers, and adapter loading.
   - Example sub-packages (`examples/simple`, `examples/structured`, `examples/streaming`) demonstrating real usage.

## Detailed Work Breakdown

### Phase 1 – Environment & Build Scaffolding

1. Add Swift Package skeleton (`swift/FundamentShim/Package.swift`) targeting macOS 15.0+.  
2. Configure Swift build to emit headers: use `swift build --show-bin-path` to discover output location, ensure `swiftc` emits `-emit-module` and `-emit-objc-header`.  
3. Add Makefile or `mage` script to orchestrate multi-language build steps (build Swift artefact, then run `go build`).  
4. Document prerequisites (Xcode 16+, command line tools, `DEVELOPER_DIR` settings).

### Phase 2 – Swift Shim Implementation

1. **Session lifecycle**:
   - Provide functions `fundament_session_create`, `fundament_session_destroy`, optionally accepting instruction strings and adapter identifiers.
   - Wrap `LanguageModelSession` initialisation, caching `SystemLanguageModel.default` and adapter management.
2. **Prompt/Response**:
   - Implement synchronous wrappers for `respond(to:)` leveraging `Task` + `CheckedContinuation` to block until completion (per Swift concurrency docs).
   - Return results as UTF-8 buffers allocated via `malloc` to transfer ownership cleanly. Provide companion free functions.
3. **Structured output**:
   - Support bridging schema definitions by passing JSON schema strings from Go, constructing `GenerationSchema` in Swift.
4. **Streaming**:
   - Expose a callback registration mechanism (`typedef void (*fundament_stream_cb)(const char *chunk, void *userdata)`), bridging Swift `ResponseStream` to repeated callback invocations on the main actor.
5. **Availability & metadata**:
   - Provide functions for availability status, locale support checks, guardrails, and adapter download/compile pathways.
6. **Error handling**:
   - Normalize Swift errors into structured C-compatible error objects (code + message). Possibly use `NSError` bridged to dictionaries or simple tagged unions.
7. **Feedback logging**:
   - Optional: expose APIs to retrieve transcripts via `LanguageModelFeedback` for telemetry.

### Phase 3 – Go Native Layer

1. Define cgo directives in `internal/native/native.go` to include generated headers and link with the Swift artefact (`#cgo LDFLAGS: -F${SRCDIR}/../swift/build -lfundament` etc.).
2. Implement handle wrappers:
   - `type sessionHandle uintptr` with constructor calling `C.fundament_session_create`.
   - Use `runtime.SetFinalizer` for automatic `destroy`.
3. Marshal data:
   - Convert Go strings to `*C.char` via `C.CString`, ensuring free.
   - Convert returned UTF-8 buffers to Go strings and free via exported Swift free function.
4. Streaming:
   - Map callback-based API to Go channels using `runtime/cgo.NewHandle` to carry channel references into C.
   - Ensure thread-safety and cgo callback best practices (callbacks invoked on threads registered with cgo).
5. Context-aware calls:
   - Provide Go methods that run Swift operations on background threads; support cancellation via contexts by introducing cooperative cancellation (e.g., cancel tokens stored in Swift).
6. Error propagation:
   - Translate Swift error structs into Go `error` with type assertions for availability vs generation errors.

### Phase 4 – Public API Design

1. Define ergonomic Go types (`Session`, `GenerationOptions`, `Adapter`, `Guardrails`, `Availability` enum).
2. Provide helper constructors mirroring Apple API concepts but idiomatic in Go (functional options pattern).
3. Encapsulate streaming as `func (s *Session) RespondStream(ctx context.Context, prompt string) (<-chan Chunk, error)` where `Chunk` carries metadata (isFinal, role).
4. Offer structured generation helper accepting Go generics with schema tags (initial approach: caller supplies target struct and we marshal/unmarshal JSON generated by Swift).
5. Add adapter management functions (download progress polling by calling Swift shim in loops).
6. Provide locale support helpers mapping to `supportsLocale`.

### Phase 5 – Examples & Tests

1. Create `examples/simple` demonstrating prompt/response and logging of availability status.
2. `examples/structured` showing `@Generable` schema usage by passing JSON schema derived from Go struct annotations.
3. `examples/streaming` printing incremental chunks.
4. Unit tests (where possible) leveraging build tags:
   - `//go:build darwin && cgo` for macOS-specific tests.
   - Mock Swift shim behind interface for non-macOS CI (use `go:build !darwin` fallback that returns helpful errors).
5. Integration smoke test script triggered manually on macOS host to ensure runtime linking works (calls each API).

### Phase 6 – Developer Experience

1. Add `docs/` with setup instructions, entitlement notes (Apple Intelligence eligibility, privacy, sandboxing).
2. Provide troubleshooting guide for common failures (model unavailable, missing adapter, entitlement issues).
3. Add automation script to verify Swift artefact built for both arm64 and x86_64 (universal binary) to support Rosetta hosts.
4. Plan for semantic versioning and release automation (GitHub Actions with macOS runners to build/test).

## Risks & Mitigations

- **Apple Intelligence availability**: many machines may lack access; provide detection APIs and skip tests gracefully.
- **Async bridging deadlocks**: rely on `CheckedContinuation` as per Swift concurrency documentation; ensure callbacks happen off the main thread to avoid UI thread issues.
- **Memory ownership**: strictly pair every exported allocation with free functions; document use for Go callers.
- **Build complexity**: centralize build steps in Makefile; cache Swift artefacts; provide preflight script verifying toolchains.
- **ABI changes**: track Foundation Models framework updates; isolate Swift shim to minimize churn.

## Next Steps

1. Scaffold Swift package and basic cgo bridge (Phase 1 & 2 foundation).
2. Implement minimal end-to-end `Respond` call to validate architecture.
3. Iterate on advanced features (structured, streaming) once plumbing is stable.
4. Flesh out documentation, examples, and release tooling.


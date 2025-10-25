# Interop Details

## Platform and availability

- The shim is compiled against **macOS 26** SDK and checks availability at runtime.  
- `fundament_session_check_availability` exposes the result to Go so callers can detect entitlement or readiness issues.  
- Agents must not attempt to run the library on earlier systems; the shim will return structured errors.

## C ABI surface

- All Swift exports use `_cdecl` with pointer-based parameters so they can be consumed from Go through `purego` symbol registration.  
- Response buffers are written into `fundament_buffer` out-parameters; Go callers must call `fundament_buffer_free` when done.  
- Streaming uses callback pointers (`fundament_stream_cb`) delivered via `purego.NewCallback`, relaying text chunks back to Go.

## Memory ownership

- Swift allocates UTF‑8 buffers using `strdup`. Ownership transfers to the Go side, which frees them via `fundament_buffer_free`.  
- Errors are represented by `fundament_error` structs and also need to be released after inspection.

## Async bridging

- Swift functions like `LanguageModelSession.respond` and `streamResponse` are awaited inside `performSync`, which marshals the result through a semaphore.  
- The closure passed to `performSync` must remain `@Sendable`; see the warning in the shim source before refactoring this logic.

Further background:

- [Architecture](architecture.md)
- [Decision: Schema Translation](../../decisions/notes/schema_support.md)

# Architecture

Fundament is composed of three layers that mirror the native Apple Foundation Models stack:

1. **Swift Shim (`swift/FundamentShim`)**  
   - Builds a dynamic library exporting a C ABI that wraps `SystemLanguageModel`, `LanguageModelSession`, and related APIs.  
   - Requires macOS 26 and Xcode 16+ because it links `FoundationModels`.  
   - Converts async Swift calls into synchronous entry points using `performSync`, ensuring Go can call the APIs serially.

2. **Native Bridge (`internal/native`)**  
   - Pure Go wrappers (via [`purego`](https://github.com/ebitengine/purego)) that marshal Go strings/options into the Swift shim and translate errors back into `error` values.  
   - Loads `libFundamentShim.dylib` at runtime, extracts it from the embedded payload, manages callbacks, and frees Swift-allocated buffers.

3. **Public Go API (`session.go`, `options.go`, `schema.go`, `availability.go`)**  
   - Provides idiomatic types such as `Session`, `GenerationOption`, and `Schema`.  
   - Supports single-turn responses, schema-guided generation, streaming via channels, and availability introspection.

```text
Go caller → fundament.Session → internal/native (purego) → Swift shim → FoundationModels (SystemLanguageModel)
```

Related docs:

- [Interop Details](interop.md)
- [Decision: Availability Enforcement](../../decisions/notes/availability.md)

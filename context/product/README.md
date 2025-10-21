# Product Overview

Fundament exposes the macOS 26 `SystemLanguageModel` Foundation API to Go applications via a Swift shim and cgo bindings.

Key resources:

- [Architecture](design/architecture.md) — system layout, layers, and module responsibilities.
- [Interop Details](design/interop.md) — Swift ↔ C ↔ Go contract, platform requirements, and memory rules.

Use these documents to understand how the public Go package, Swift shim, and Apple frameworks fit together before making behavioural changes.

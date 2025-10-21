# Availability Enforcement

## Summary

The Swift shim is compiled and executed against `SystemLanguageModel` APIs that Apple ships with macOS 26. Running on earlier versions would crash or produce linker failures because the `FoundationModels` framework is absent.

## Implications

- `fundament_session_check_availability` maps `SystemLanguageModel.Availability` into a Go-facing enum so callers can degrade gracefully.  
- All exported functions guard with `#available(macOS 26.0, *)` and emit descriptive errors (e.g. `deviceNotEligible`, `modelNotReady`).  
- Tests and examples must be skipped or stubbed on machines that do not satisfy the entitlement.

## Guidance for future changes

- Do **not** lower the deployment target until Apple backports the framework (unlikely).  
- If multi-platform support is needed, provide mock implementations under alternative build tags rather than relaxing runtime checks.

Related reading: [Interop Details](../../product/design/interop.md)

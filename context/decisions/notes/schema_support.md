# Schema Translation Strategy

## Summary

Structured generation relies on passing a JSON description of a `DynamicGenerationSchema` from Go into Swift. Because Swift cannot decode arbitrary JSON into `DynamicGenerationSchema` directly, we implemented a lightweight translator (`SchemaNode` → `DynamicGenerationSchema`) that covers common patterns.

## Current coverage

- Object schemas with named properties  
- Array schemas with `minimumElements` / `maximumElements` and item definitions  
- String enumerations via `anyOf` arrays  
- Primitive string, integer, and boolean fields

## Known limitations

- No support yet for references, nested dependencies, numeric guides, or custom `GenerationGuide` constraints.  
- The translator throws descriptive errors when it encounters unsupported shapes; Go callers should handle these errors and adjust their schema accordingly.

## Next steps (if needed)

- Extend `SchemaNode` to capture additional metadata (`guides`, range constraints, nested enums).  
- Keep the translator in sync with Go helpers (see `schema.go`) so users cannot construct schemas that Swift rejects.

Related docs: [Interop Details](../../product/design/interop.md), [Troubleshooting](../../operations/build/troubleshooting.md)

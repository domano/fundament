# Fundament

Go bindings for Apple’s on-device `SystemLanguageModel` (macOS 26). The library bridges Swift’s Foundation Models framework into an idiomatic Go API, so you can build Apple Intelligence features without leaving Go.

> ⚠️ **Requirements**: macOS 26 (Sequoia) with Apple Intelligence entitlement enabled, Xcode 16 or newer (or matching Command Line Tools), Go 1.25+ with `CGO_ENABLED=1`.

## Install & Build

Clone the repo and build the Swift shim and Go packages:

```bash
git clone https://github.com/domano/fundament.git
cd fundament
make swift   # builds libFundamentShim.dylib
make go      # builds fundament and examples
```

`make swift` writes `libFundamentShim.dylib` into `swift/FundamentShim/.build/Release` and embeds an `rpath` so Go binaries find the shim at runtime. If you run binaries outside the repo, export `DYLD_LIBRARY_PATH=swift/FundamentShim/.build/Release` first.

## Quick Start

The library centres around the `Session` type. Create a session, send prompts, and close it when you’re done:

```go
session, err := fundament.NewSession(fundament.SessionOptions{
    Instructions: "You are a concise assistant that answers in one sentence.",
})
if err != nil {
    log.Fatal(err)
}
defer session.Close()

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

resp, err := session.Respond(ctx, "Explain what Foundation Models are on macOS.")
if err != nil {
    log.Fatal(err)
}

fmt.Println(resp.Text)
```

Before you start a session, call `fundament.CheckAvailability()` to ensure the on-device model is ready. The helper returns detailed reasons (device not eligible, Apple Intelligence disabled, model still downloading, etc.) so your app can degrade gracefully.

## Examples

The `examples/` directory contains small programs you can adapt for your own projects.

### 1. Simple: single-turn response

`examples/simple/main.go` shows the minimal flow: check availability, create a session with instructions, prompt once, and print the answer.

Run it after `make swift`:

```bash
go run ./examples/simple
```

### 2. Structured: schema-guided output

`examples/structured/main.go` passes a JSON schema built with `fundament.SchemaFromRawJSON`. The schema describes the shape of the response, letting you request strongly typed content (e.g. a travel plan with specific fields). The Swift shim converts that JSON into a `DynamicGenerationSchema` before calling `LanguageModelSession.respond`.

Adapt the `schemaDefinition` map to your own structure; the current translator supports objects, arrays (with min/max counts), string enumerations, and string/int/bool primitives.

### 3. Streaming: incremental completions

`examples/streaming/main.go` demonstrates live updates via `Session.RespondStream`. The Go API returns a channel of `StreamChunk` values; each chunk contains text and a flag indicating whether it’s the final piece. This example prints a limerick word by word.

### 4. Web chat: server-rendered UI

`examples/webchat` starts a minimal HTTP server that renders a chat experience with Go’s `html/template`. Post a prompt from the browser and the handler keeps the conversation history on the server, forwarding each turn to a shared `fundament.Session`.

## Using Fundament in your project

Add the module to your `go.mod` and ensure the Swift shim is built:

```bash
go get github.com/domano/fundament
make swift  # run inside the module checkout (or vendor the shim artefact)
```

At runtime, the Go package expects `libFundamentShim.dylib` to be discoverable via `DYLD_LIBRARY_PATH` or the embedded `rpath`. For deployment, ship the dylib alongside your binary.

### Key APIs

- `fundament.NewSession(opts SessionOptions)` — creates a session bound to the default system language model.
- `(*Session).Respond(ctx, prompt, opts...)` — single prompt/response.
- `(*Session).RespondStructured(ctx, prompt, schema, opts...)` — returns structured JSON.
- `(*Session).RespondStructuredInto(ctx, prompt, schema, target, opts...)` — unmarshals directly into a Go value.
- `(*Session).RespondStream(ctx, prompt, opts...)` — returns a channel of streaming updates.
- `fundament.SchemaFromRawJSON(data)` / `SchemaFromValue(value)` — helpers for building generation schemas.
- `fundament.WithTemperature`, `WithTopP`, `WithMaxTokens`, etc. — options passed through to `GenerationOptions`.

See the source files (`session.go`, `schema.go`, `options.go`, `availability.go`) for full API signatures and comments.

## Troubleshooting

- **Unavailable model**: `fundament.CheckAvailability()` returns `AvailabilityUnavailable` with a reason (device not eligible, Apple Intelligence disabled, model not ready). Handle this before prompting.
- **Linker can’t find the shim**: double-check `make swift` succeeded and that your `DYLD_LIBRARY_PATH` includes `swift/FundamentShim/.build/Release` when executing binaries.
- **Structured schema errors**: the current translator supports objects, arrays, enums, and primitive fields. Unsupported shapes (references, numeric guides) return descriptive errors from the shim.

For deeper operational guidance, read [`docs/GettingStarted.md`](docs/GettingStarted.md) and the context notes under [`context/`](context/README.md).

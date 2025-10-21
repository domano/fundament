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

## Use in Your Project

Pull the module into your app, then build the Swift shim so the cgo bindings can link against it:

```bash
go get github.com/domano/fundament
make swift  # run inside the module checkout (or vendor the shim artefact)
```

At runtime the Go binary must be able to locate `libFundamentShim.dylib`. Running inside the repo works out of the box thanks to the embedded `rpath`. If you ship the binary elsewhere, bundle the dylib alongside it or export `DYLD_LIBRARY_PATH=swift/FundamentShim/.build/Release` before launch.

## Examples

The `examples/` directory contains small programs you can adapt for your own projects.

### 1. Simple — single turn

Minimal prompt/response with a shared `Session`:

```go
availability, err := fundament.CheckAvailability()
// ...
session, err := fundament.NewSession(fundament.SessionOptions{
	Instructions: "You are a concise assistant that answers in one sentence.",
})
defer session.Close()

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := session.Respond(ctx, "Explain what a markov chain is.")
fmt.Println("Assistant:", response.Text)
```

```bash
go run ./examples/simple
```

### 2. Structured — schema-guided output

Build a JSON schema on the fly and request strongly typed data:

```go
schemaDefinition := map[string]any{
	"name": "TravelPlan",
	"properties": []map[string]any{
		{
			"name": "destination",
			"schema": map[string]any{
				"type": "string",
			},
		},
		// ...
	},
}
schemaBytes, _ := json.Marshal(schemaDefinition)
schema, _ := fundament.SchemaFromRawJSON(schemaBytes)

res, err := session.RespondStructured(ctx, "Plan a 2-day trip to Kyoto in autumn", schema)
fmt.Println(string(res.JSON))
```

```bash
go run ./examples/structured
```

### 3. Streaming — incremental completions

Stream text chunks as they arrive:

```go
stream, err := session.RespondStream(
	ctx,
	"Write a limerick about coding Go bindings for Swift models.",
	fundament.WithTemperature(0.7),
)

for chunk := range stream {
	if chunk.Err != nil {
		log.Fatal(chunk.Err)
	}
	fmt.Print(chunk.Text, " ")
	if chunk.Final {
		fmt.Println("\n-- end --")
	}
}
```

```bash
go run ./examples/streaming
```

### 4. Web chat — server-rendered UI

Server-side conversation loop with Go templates:

```go
func (s *chatServer) handleChat(w http.ResponseWriter, r *http.Request) {
	// ...
	prompt := s.appendUserAndPrompt(strings.TrimSpace(r.FormValue("message")))

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	resp, err := s.session.Respond(ctx, prompt)
	if err != nil {
		s.appendSystemMessage(fmt.Sprintf("Response error: %v", err))
	} else {
		s.appendAssistantMessage(resp.Text)
	}
	// ...
}
```

```bash
go run ./examples/webchat
# Then open http://localhost:8080 in your browser.
```

## Key APIs

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

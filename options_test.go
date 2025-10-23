package fundament

import (
	"encoding/json"
	"testing"
)

func TestEncodeGenerationOptions(t *testing.T) {
	opts, payload, err := encodeGenerationOptions([]GenerationOption{
		WithTemperature(0.9),
		WithTopP(0.2),
		WithTopK(5),
		WithMaxTokens(256),
		WithSeed(123),
		nil,
	})
	if err != nil {
		t.Fatalf("encodeGenerationOptions error: %v", err)
	}
	if opts.Temperature == nil || *opts.Temperature != 0.9 {
		t.Fatalf("unexpected temperature %+v", opts.Temperature)
	}
	if payload == "" {
		t.Fatal("expected JSON payload")
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("invalid JSON payload: %v", err)
	}
	if decoded["temperature"] != 0.9 ||
		decoded["topP"] != 0.2 ||
		int(decoded["topK"].(float64)) != 5 ||
		int(decoded["maxTokens"].(float64)) != 256 ||
		int(decoded["seed"].(float64)) != 123 {
		t.Fatalf("unexpected decoded payload %+v", decoded)
	}

	_, payload, err = encodeGenerationOptions(nil)
	if err != nil {
		t.Fatalf("encodeGenerationOptions with nil overrides error: %v", err)
	}
	if payload != "" {
		t.Fatalf("expected empty payload, got %q", payload)
	}
}

package fundament

import (
	"encoding/json"
)

// GenerationOptions captures decoding-friendly options passed to the Swift shim.
type GenerationOptions struct {
	Temperature *float64
	TopP        *float64
	TopK        *int
	MaxTokens   *int
	Seed        *uint64
}

// GenerationOption mutates GenerationOptions before encoding them for the shim.
type GenerationOption func(*GenerationOptions)

// WithTemperature sets the sampling temperature.
func WithTemperature(v float64) GenerationOption {
	return func(opts *GenerationOptions) {
		opts.Temperature = &v
	}
}

// WithTopP configures nucleus sampling probability.
func WithTopP(v float64) GenerationOption {
	return func(opts *GenerationOptions) {
		opts.TopP = &v
	}
}

// WithTopK configures the top-k token cut-off.
func WithTopK(v int) GenerationOption {
	return func(opts *GenerationOptions) {
		opts.TopK = &v
	}
}

// WithMaxTokens caps the generated token count.
func WithMaxTokens(v int) GenerationOption {
	return func(opts *GenerationOptions) {
		opts.MaxTokens = &v
	}
}

// WithSeed enforces deterministic sampling with the specified seed.
func WithSeed(v uint64) GenerationOption {
	return func(opts *GenerationOptions) {
		opts.Seed = &v
	}
}

func encodeGenerationOptions(overrides []GenerationOption) (GenerationOptions, string, error) {
	var base GenerationOptions
	for _, opt := range overrides {
		if opt != nil {
			opt(&base)
		}
	}
	payload := map[string]any{}
	if base.Temperature != nil {
		payload["temperature"] = *base.Temperature
	}
	if base.TopP != nil {
		payload["topP"] = *base.TopP
	}
	if base.TopK != nil {
		payload["topK"] = *base.TopK
	}
	if base.MaxTokens != nil {
		payload["maxTokens"] = *base.MaxTokens
	}
	if base.Seed != nil {
		payload["seed"] = *base.Seed
	}

	if len(payload) == 0 {
		return base, "", nil
	}
	blob, err := json.Marshal(payload)
	if err != nil {
		return base, "", err
	}
	return base, string(blob), nil
}

package fundament

import (
	"encoding/json"
	"errors"
)

// Schema wraps a Foundation Models GenerationSchema serialized as JSON.
type Schema struct {
	raw json.RawMessage
}

// SchemaFromRawJSON constructs a Schema from a JSON blob that describes a DynamicGenerationSchema.
func SchemaFromRawJSON(data []byte) (Schema, error) {
	if len(data) == 0 {
		return Schema{}, errors.New("fundament: schema JSON must not be empty")
	}
	var tmp any
	if err := json.Unmarshal(data, &tmp); err != nil {
		return Schema{}, err
	}
	return Schema{raw: json.RawMessage(append([]byte(nil), data...))}, nil
}

// SchemaFromValue marshals a Go value to JSON and wraps it as a Schema.
// This helper is useful when the schema is provided as a struct literal compatible with DynamicGenerationSchema.
func SchemaFromValue(v any) (Schema, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return Schema{}, err
	}
	return SchemaFromRawJSON(data)
}

// MarshalJSON implements json.Marshaler.
func (s Schema) MarshalJSON() ([]byte, error) {
	if len(s.raw) == 0 {
		return []byte("null"), nil
	}
	return append([]byte(nil), s.raw...), nil
}

func (s Schema) String() string {
	if len(s.raw) == 0 {
		return ""
	}
	return string(s.raw)
}

// Raw returns the schema bytes.
func (s Schema) Raw() []byte {
	if len(s.raw) == 0 {
		return nil
	}
	return append([]byte(nil), s.raw...)
}

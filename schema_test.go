package fundament

import (
	"encoding/json"
	"testing"
)

func TestSchemaFromRawJSON(t *testing.T) {
	_, err := SchemaFromRawJSON(nil)
	if err == nil {
		t.Fatal("expected error for empty payload")
	}

	blob := []byte(`{"type":"object","name":"Example"}`)
	s, err := SchemaFromRawJSON(blob)
	if err != nil {
		t.Fatalf("SchemaFromRawJSON error: %v", err)
	}

	blob[2] = 'X'
	if s.String() != `{"type":"object","name":"Example"}` {
		t.Fatalf("schema should retain original bytes, got %s", s.String())
	}

	out, err := json.Marshal(struct {
		Schema Schema `json:"schema"`
	}{Schema: s})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if string(out) != `{"schema":{"type":"object","name":"Example"}}` {
		t.Fatalf("unexpected marshal result %s", string(out))
	}

	raw := s.Raw()
	if string(raw) != `{"type":"object","name":"Example"}` {
		t.Fatalf("unexpected raw %s", string(raw))
	}
	raw[2] = 'Z'
	if s.String() != `{"type":"object","name":"Example"}` {
		t.Fatal("schema raw should be immutable copy")
	}
}

func TestSchemaFromValue(t *testing.T) {
	type inner struct {
		Type string `json:"type"`
	}

	s, err := SchemaFromValue(inner{Type: "string"})
	if err != nil {
		t.Fatalf("SchemaFromValue error: %v", err)
	}
	if s.String() != `{"type":"string"}` {
		t.Fatalf("unexpected schema string %s", s.String())
	}
}

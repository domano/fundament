//go:build integration && darwin && cgo

package fundament

import (
	"context"
	"strings"
	"testing"
	"time"
)

const integrationTimeout = 45 * time.Second

func requireAvailabilityReady(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("integration tests disabled in short mode")
	}
	availability, err := CheckAvailability()
	if err != nil {
		t.Fatalf("CheckAvailability error: %v", err)
	}
	if availability.State != AvailabilityReady {
		t.Fatalf("SystemLanguageModel unavailable: %v", availability)
	}
}

func newIntegrationSession(t *testing.T) *Session {
	t.Helper()
	session, err := NewSession(SessionOptions{
		Instructions: "You are an automated test harness. Provide concise, factual answers.",
	})
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Errorf("Close error: %v", err)
		}
	})
	return session
}

func TestIntegrationRespondBasic(t *testing.T) {
	requireAvailabilityReady(t)
	session := newIntegrationSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	resp, err := session.Respond(ctx, "Acknowledge this integration test in one short sentence.")
	if err != nil {
		t.Fatalf("Respond error: %v", err)
	}
	if strings.TrimSpace(resp.Text) == "" {
		t.Fatal("expected non-empty response from model")
	}
}

func TestIntegrationRespondStructured(t *testing.T) {
	requireAvailabilityReady(t)
	session := newIntegrationSession(t)

	schemaJSON := []byte(`{
		"name": "IntegrationResult",
		"properties": [
			{
				"name": "status",
				"schema": {
					"type": "string",
					"anyOf": ["ok", "ready", "pass"]
				}
			},
			{
				"name": "notes",
				"schema": {
					"type": "string"
				}
			}
		]
	}`)
	schema, err := SchemaFromRawJSON(schemaJSON)
	if err != nil {
		t.Fatalf("SchemaFromRawJSON error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	var result struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	err = session.RespondStructuredInto(ctx, "Return JSON confirming the integration bridge is working. Use one of the allowed status values.", schema, &result)
	if err != nil {
		t.Fatalf("RespondStructuredInto error: %v", err)
	}
	status := strings.ToLower(strings.TrimSpace(result.Status))
	switch status {
	case "ok", "ready", "pass":
	default:
		t.Fatalf("unexpected status %q", result.Status)
	}
	if strings.TrimSpace(result.Notes) == "" {
		t.Fatal("expected notes to describe the result")
	}
}

func TestIntegrationRespondStream(t *testing.T) {
	requireAvailabilityReady(t)
	session := newIntegrationSession(t)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	stream, err := session.RespondStream(ctx, "Provide a short two-word confirmation for this integration test.")
	if err != nil {
		t.Fatalf("RespondStream error: %v", err)
	}

	var builder strings.Builder
	var sawFinal bool
	for chunk := range stream {
		if chunk.Err != nil {
			t.Fatalf("stream chunk error: %v", chunk.Err)
		}
		builder.WriteString(chunk.Text)
		if chunk.Final {
			sawFinal = true
		}
	}
	if !sawFinal {
		t.Fatal("expected final chunk in stream")
	}
	if strings.TrimSpace(builder.String()) == "" {
		t.Fatal("expected stream to yield text")
	}
}

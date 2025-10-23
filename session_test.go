package fundament

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
	"unsafe"

	"github.com/domano/fundament/internal/native"
)

func withSessionHooks(
	create func(string) (native.SessionRef, error),
	destroy func(native.SessionRef),
	respond func(native.SessionRef, string, string) (string, error),
	respondStructured func(native.SessionRef, string, string, string) (string, error),
	stream func(native.SessionRef, string, string, native.StreamCallback) error,
) func() {
	prevCreate := nativeSessionCreate
	prevDestroy := nativeSessionDestroy
	prevRespond := nativeSessionRespond
	prevRespondStructured := nativeSessionRespondStructured
	prevStream := nativeSessionStream

	if create != nil {
		nativeSessionCreate = create
	}
	if destroy != nil {
		nativeSessionDestroy = destroy
	}
	if respond != nil {
		nativeSessionRespond = respond
	}
	if respondStructured != nil {
		nativeSessionRespondStructured = respondStructured
	}
	if stream != nil {
		nativeSessionStream = stream
	}

	return func() {
		nativeSessionCreate = prevCreate
		nativeSessionDestroy = prevDestroy
		nativeSessionRespond = prevRespond
		nativeSessionRespondStructured = prevRespondStructured
		nativeSessionStream = prevStream
	}
}

func TestNewSessionAndRespond(t *testing.T) {
	dummyRef := native.SessionRef(unsafe.Pointer(&struct{}{}))

	restore := withSessionHooks(
		func(instr string) (native.SessionRef, error) {
			if instr != "test instructions" {
				t.Fatalf("unexpected instructions %q", instr)
			}
			return dummyRef, nil
		},
		nil,
		func(ref native.SessionRef, prompt, opts string) (string, error) {
			if ref != dummyRef {
				t.Fatalf("unexpected ref %+v", ref)
			}
			if prompt != "ping" {
				t.Fatalf("unexpected prompt %q", prompt)
			}
			// ensure generation options are encoded when provided
			if opts == "" {
				t.Fatal("expected options payload")
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(opts), &payload); err != nil {
				t.Fatalf("invalid options JSON: %v", err)
			}
			if payload["temperature"] != 0.5 {
				t.Fatalf("expected temperature 0.5, got %v", payload["temperature"])
			}
			return "pong", nil
		},
		nil,
		nil,
	)
	defer restore()

	session, err := NewSession(SessionOptions{Instructions: "test instructions"})
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}
	defer session.Close()

	resp, err := session.Respond(context.Background(), "ping", WithTemperature(0.5))
	if err != nil {
		t.Fatalf("Respond error: %v", err)
	}
	if resp.Text != "pong" {
		t.Fatalf("unexpected response %q", resp.Text)
	}
}

func TestRespondContextCancelled(t *testing.T) {
	session := &Session{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := session.Respond(ctx, "ignored")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestRespondAfterClose(t *testing.T) {
	session := &Session{
		closed: true,
	}
	_, err := session.Respond(context.Background(), "ping")
	if err == nil {
		t.Fatal("expected error when responding on closed session")
	}
}

func TestSessionCloseIdempotent(t *testing.T) {
	dummyRef := native.SessionRef(unsafe.Pointer(&struct{}{}))
	var destroys int
	restore := withSessionHooks(nil, func(ref native.SessionRef) {
		if ref != dummyRef {
			t.Fatalf("unexpected ref %v", ref)
		}
		destroys++
	}, nil, nil, nil)
	defer restore()

	session := &Session{ref: dummyRef}
	if err := session.Close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}
	if destroys != 1 {
		t.Fatalf("expected destroy once, got %d", destroys)
	}
}

func TestRespondStructuredInto(t *testing.T) {
	dummyRef := native.SessionRef(unsafe.Pointer(&struct{}{}))
	restore := withSessionHooks(
		func(string) (native.SessionRef, error) { return dummyRef, nil },
		nil,
		nil,
		func(ref native.SessionRef, prompt, schemaJSON, opts string) (string, error) {
			if ref != dummyRef {
				t.Fatalf("unexpected ref %v", ref)
			}
			if prompt != "generate" {
				t.Fatalf("unexpected prompt %q", prompt)
			}
			if schemaJSON == "" {
				t.Fatal("expected schema JSON")
			}
			return `{"message":"ok","value":42}`, nil
		},
		nil,
	)
	defer restore()

	session, err := NewSession(SessionOptions{})
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}

	schema, err := SchemaFromValue(map[string]any{
		"type": "object",
	})
	if err != nil {
		t.Fatalf("SchemaFromValue error: %v", err)
	}

	var payload struct {
		Message string
		Value   int
	}
	if err := session.RespondStructuredInto(context.Background(), "generate", schema, &payload); err != nil {
		t.Fatalf("RespondStructuredInto error: %v", err)
	}
	if payload.Message != "ok" || payload.Value != 42 {
		t.Fatalf("unexpected payload %+v", payload)
	}
}

func TestRespondStructuredRequiresSchema(t *testing.T) {
	session := &Session{}
	if _, err := session.RespondStructured(context.Background(), "prompt", Schema{}); err == nil {
		t.Fatal("expected error for empty schema")
	}
}

func TestRespondStream(t *testing.T) {
	dummyRef := native.SessionRef(unsafe.Pointer(&struct{}{}))
	var streamCalled bool
	restore := withSessionHooks(
		func(string) (native.SessionRef, error) { return dummyRef, nil },
		nil,
		nil,
		nil,
		func(ref native.SessionRef, prompt, opts string, cb native.StreamCallback) error {
			streamCalled = true
			cb("hello", false)
			cb("world", true)
			return nil
		},
	)
	defer restore()

	session, err := NewSession(SessionOptions{})
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}

	ch, err := session.RespondStream(context.Background(), "hi")
	if err != nil {
		t.Fatalf("RespondStream error: %v", err)
	}

	var got []StreamChunk
	for chunk := range ch {
		got = append(got, chunk)
	}

	if !streamCalled {
		t.Fatal("expected stream hook to be invoked")
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(got))
	}
	if got[0].Text != "hello" || got[0].Final {
		t.Fatalf("unexpected first chunk %+v", got[0])
	}
	if got[1].Text != "world" || !got[1].Final {
		t.Fatalf("unexpected second chunk %+v", got[1])
	}
}

func TestConcurrentRespondCalls(t *testing.T) {
	dummyRef := native.SessionRef(unsafe.Pointer(&struct{}{}))
	var mu sync.Mutex
	var prompts []string
	restore := withSessionHooks(
		func(string) (native.SessionRef, error) { return dummyRef, nil },
		nil,
		func(ref native.SessionRef, prompt, _ string) (string, error) {
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			prompts = append(prompts, prompt)
			mu.Unlock()
			return prompt + "-ok", nil
		},
		nil,
		nil,
	)
	defer restore()

	session, err := NewSession(SessionOptions{})
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			prompt := "p" + string(rune('A'+i))
			resp, err := session.Respond(context.Background(), prompt)
			if err != nil {
				t.Errorf("Respond error: %v", err)
				return
			}
			if resp.Text != prompt+"-ok" {
				t.Errorf("unexpected response %q", resp.Text)
			}
		}(i)
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if len(prompts) != 3 {
		t.Fatalf("expected 3 prompts recorded, got %d", len(prompts))
	}
}

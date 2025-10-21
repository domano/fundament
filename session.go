package fundament

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/domano/fundament/internal/native"
)

// SessionOptions configure how a Session is created.
type SessionOptions struct {
	Instructions string
}

// Session wraps a native session handle.
type Session struct {
	mu      sync.RWMutex
	ref     native.SessionRef
	closed  bool
	instr   string
	created time.Time
}

// NewSession creates a new LanguageModelSession bound to the default SystemLanguageModel.
func NewSession(opts SessionOptions) (*Session, error) {
	ref, err := native.SessionCreate(opts.Instructions)
	if err != nil {
		return nil, err
	}
	return &Session{
		ref:     ref,
		instr:   opts.Instructions,
		created: time.Now(),
	}, nil
}

// Close releases native resources.
func (s *Session) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	ref := s.ref
	s.ref = nil
	s.closed = true
	s.mu.Unlock()
	if ref != nil {
		native.SessionDestroy(ref)
	}
	return nil
}

// Response captures the result of a Respond call.
type Response struct {
	Text string
}

// StructuredResponse captures a structured result in JSON form.
type StructuredResponse struct {
	JSON json.RawMessage
}

// Respond performs a single-shot generation call.
func (s *Session) Respond(ctx context.Context, prompt string, opts ...GenerationOption) (Response, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return Response{}, err
		}
	}
	_, blob, err := encodeGenerationOptions(opts)
	if err != nil {
		return Response{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.ref == nil {
		return Response{}, errors.New("fundament: session has been closed")
	}
	text, err := native.SessionRespond(s.ref, prompt, blob)
	if err != nil {
		return Response{}, err
	}
	return Response{Text: text}, nil
}

// RespondStructured generates content guided by a schema, returning raw JSON.
func (s *Session) RespondStructured(ctx context.Context, prompt string, schema Schema, opts ...GenerationOption) (StructuredResponse, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return StructuredResponse{}, err
		}
	}
	if len(schema.raw) == 0 {
		return StructuredResponse{}, errors.New("fundament: schema must not be empty")
	}
	_, blob, err := encodeGenerationOptions(opts)
	if err != nil {
		return StructuredResponse{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.ref == nil {
		return StructuredResponse{}, errors.New("fundament: session has been closed")
	}
	text, err := native.SessionRespondStructured(s.ref, prompt, string(schema.raw), blob)
	if err != nil {
		return StructuredResponse{}, err
	}
	return StructuredResponse{JSON: json.RawMessage(text)}, nil
}

// RespondStructuredInto populates target with the structured response.
func (s *Session) RespondStructuredInto(ctx context.Context, prompt string, schema Schema, target any, opts ...GenerationOption) error {
	res, err := s.RespondStructured(ctx, prompt, schema, opts...)
	if err != nil {
		return err
	}
	if target == nil {
		return errors.New("fundament: target must not be nil")
	}
	return json.Unmarshal(res.JSON, target)
}

// StreamChunk represents an incremental update during streaming.
type StreamChunk struct {
	Text  string
	Final bool
	Err   error
}

// RespondStream streams a response into a channel. The returned channel is closed when streaming completes or on error.
func (s *Session) RespondStream(ctx context.Context, prompt string, opts ...GenerationOption) (<-chan StreamChunk, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	_, blob, err := encodeGenerationOptions(opts)
	if err != nil {
		return nil, err
	}

	out := make(chan StreamChunk, 8)
	go func() {
		defer close(out)
		s.mu.RLock()
		if s.closed || s.ref == nil {
			s.mu.RUnlock()
			return
		}
		ref := s.ref
		s.mu.RUnlock()
		err := native.SessionStream(ref, prompt, blob, func(chunk string, final bool) {
			select {
			case <-ctx.Done():
				return
			case out <- StreamChunk{Text: chunk, Final: final}:
			}
		})
		if err != nil && errors.Is(ctx.Err(), context.Canceled) {
			return
		}
		if err != nil {
			select {
			case <-ctx.Done():
			case out <- StreamChunk{Err: err, Final: true}:
			}
			return
		}
	}()
	return out, nil
}

// Instructions returns the initial instructions configured for the session.
func (s *Session) Instructions() string {
	return s.instr
}

func (s *Session) String() string {
	return fmt.Sprintf("Session(created=%s)", s.created.Format(time.RFC3339))
}

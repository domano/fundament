package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/domano/fundament"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

var pageTemplate = template.Must(template.ParseFS(templateFS, "templates/index.gohtml"))

type message struct {
	Role    string
	Content string
}

func initialMessages() []message {
	return []message{
		{Role: "assistant", Content: "Hi there! Ask me about anything."},
	}
}

type chatServer struct {
	session *fundament.Session
	mu      sync.Mutex
	history []message
}

func (s *chatServer) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "could not read form", http.StatusBadRequest)
			return
		}

		userMessage := strings.TrimSpace(r.FormValue("message"))
		if userMessage == "" {
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
			return
		}

		prompt := s.appendUserAndPrompt(userMessage)

		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()

		resp, err := s.session.Respond(ctx, prompt)
		if err != nil {
			log.Printf("respond: %v", err)
			s.appendSystemMessage(fmt.Sprintf("Response error: %v", err))
		} else {
			s.appendAssistantMessage(resp.Text)
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet && r.URL.Query().Get("reset") == "1" {
		s.resetConversation()
		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
		return
	}

	if err := pageTemplate.Execute(w, struct {
		History []message
	}{
		History: s.historySnapshot(),
	}); err != nil {
		log.Printf("template execute: %v", err)
	}
}

func (s *chatServer) appendUserAndPrompt(content string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, message{Role: "user", Content: content})
	return buildPrompt(s.history)
}

func (s *chatServer) appendAssistantMessage(content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, message{Role: "assistant", Content: content})
}

func (s *chatServer) appendSystemMessage(content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = append(s.history, message{Role: "system", Content: content})
}

func (s *chatServer) resetConversation() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = initialMessages()
}

func (s *chatServer) historySnapshot() []message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]message, len(s.history))
	copy(out, s.history)
	return out
}

func buildPrompt(history []message) string {
	var b strings.Builder
	b.WriteString("Continue the conversation between the user and the assistant.\n\n")
	for _, msg := range history {
		switch msg.Role {
		case "assistant":
			b.WriteString("Assistant: ")
		case "system":
			b.WriteString("System: ")
		default:
			b.WriteString("User: ")
		}
		b.WriteString(msg.Content)
		b.WriteByte('\n')
	}
	b.WriteString("Assistant:")
	return b.String()
}

func main() {
	availability, err := fundament.CheckAvailability()
	if err != nil {
		log.Fatalf("check availability: %v", err)
	}
	if availability.State != fundament.AvailabilityReady {
		log.Fatalf("system language model unavailable: %s", availability)
	}

	session, err := fundament.NewSession(fundament.SessionOptions{
		Instructions: "You are a friendly assistant answering questions on a website chat widget. Keep responses short and helpful.",
	})
	if err != nil {
		log.Fatalf("new session: %v", err)
	}
	defer session.Close()

	server := &chatServer{
		session: session,
		history: initialMessages(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleChat)

	addr := ":8080"
	log.Printf("web chat example listening on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen and serve: %v", err)
	}
}

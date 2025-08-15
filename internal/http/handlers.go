package http

import (
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"waitroom-chatbot/internal/core"
	"waitroom-chatbot/internal/db"
	"waitroom-chatbot/pkg"
)

// Server bundles together dependencies required by HTTP handlers.
type Server struct {
	Repo       *db.Repository
	Chat       *core.ChatService
	Templates  *template.Template
	MessageCap int
}

// NewServer constructs a Server. Templates are loaded from internal/http/templates.
func NewServer(repo *db.Repository, chat *core.ChatService, messageCap int) (*Server, error) {
	tmplPath := filepath.Join("internal", "http", "templates", "*.html")
	tmpl, err := template.ParseGlob(tmplPath)
	if err != nil {
		return nil, err
	}
	return &Server{Repo: repo, Chat: chat, Templates: tmpl, MessageCap: messageCap}, nil
}

// ServeHTTP performs very small routing based on path.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/":
		s.handleStartPage(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/start":
		s.handleStart(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/chat/"):
		nationalID := strings.TrimPrefix(r.URL.Path, "/chat/")
		s.handleChatPage(w, r, nationalID)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/users/") && strings.HasSuffix(r.URL.Path, "/messages"):
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 4 {
			nationalID := parts[3]
			s.handlePostMessage(w, r, nationalID)
			return
		}
		http.NotFound(w, r)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/sessions/") && strings.HasSuffix(r.URL.Path, "/messages"):
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) >= 4 {
			nationalID := parts[3]
			s.handlePostMessage(w, r, nationalID)
			return
		}
		http.NotFound(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleStartPage renders the initial form for collecting user details.
func (s *Server) handleStartPage(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("national_id"); err == nil && c.Value != "" {
		http.Redirect(w, r, "/chat/"+c.Value, http.StatusSeeOther)
		return
	}
	if err := s.Templates.ExecuteTemplate(w, "start", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleStart processes the start form, stores user info and redirects to chat page.
func (s *Server) handleStart(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	u := &pkg.User{
		NationalID: r.FormValue("national_id"),
		Phone:      r.FormValue("phone"),
		Name:       r.FormValue("name"),
	}
	if u.NationalID == "" || u.Phone == "" || u.Name == "" {
		http.Error(w, "missing fields", http.StatusBadRequest)
		return
	}
	if err := s.Repo.UpsertUser(r.Context(), u); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "national_id",
		Value:  u.NationalID,
		Path:   "/",
		MaxAge: int((365 * 24 * time.Hour).Seconds()),
	})
	http.Redirect(w, r, "/chat/"+u.NationalID, http.StatusSeeOther)
}

// GetTranscriptSince returns the transcript for a nationalID but only messages
// with created_at >= since. It reuses GetTranscript and filters in-memory to
// avoid coupling to any specific SQL shape used by GetTranscript.
// Moved to db/repository.go

// handleChatPage renders the chat interface for a user.
func (s *Server) handleChatPage(w http.ResponseWriter, r *http.Request, nationalID string) {
	transcript, err := s.Repo.GetTranscript(r.Context(), nationalID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		SessionID  string // template expects .SessionID
		NationalID string // keep for any other template usage
		Transcript []pkg.Message
	}{
		SessionID:  nationalID,
		NationalID: nationalID,
		Transcript: transcript,
	}
	if err := s.Templates.ExecuteTemplate(w, "patient", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handlePostMessage accepts a patient message, checks weekly cap and responds with bot reply.
func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request, nationalID string) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	content := r.FormValue("content")
	if strings.TrimSpace(content) == "" {
		http.Error(w, "empty message", http.StatusBadRequest)
		return
	}
	count, err := s.Repo.CountUserMessagesThisWeek(r.Context(), nationalID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count >= s.MessageCap {
		// send cap message only
		botMsg, _ := s.Repo.CreateMessage(r.Context(), nationalID, pkg.RoleBot, core.CapMessage)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<div class="msg bot">` + template.HTMLEscapeString(botMsg.Content) + `</div>`))
		return
	}
	// store patient message
	if _, err := s.Repo.CreateMessage(r.Context(), nationalID, pkg.RolePatient, content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Build LLM reply using last week's transcript for context
	since := time.Now().AddDate(0, 0, -7)
	ctxTranscript, err := s.Repo.GetTranscriptSince(r.Context(), nationalID, since)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reply, err := s.Chat.ReplyWithContext(r.Context(), nationalID, content, ctxTranscript)
	if err != nil {
		// Trigger HTMX error bubble; patient bubble already appended client-side
		http.Error(w, "llm error", http.StatusBadGateway)
		return
	}
	if _, err := s.Repo.CreateMessage(r.Context(), nationalID, pkg.RoleBot, reply); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	escReply := template.HTMLEscapeString(reply)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<div class="msg bot">` + escReply + `</div>`))
}

package http

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"waitroom-chatbot/internal/core"
	"waitroom-chatbot/internal/db"
	"waitroom-chatbot/pkg"
)

// Server bundles together the dependencies required by HTTP handlers.  It
// implements http.Handler so it can be passed to http.ListenAndServe.
type Server struct {
	Repo       *db.Repository
	Chat       *core.ChatService
	Summarizer *core.Summarizer
	Notifier   *db.Notifier
	Templates  *template.Template
	MessageCap int
}

// NewServer constructs a Server.  It loads HTML templates from the
// internal/http/templates directory relative to the current working directory.
func NewServer(repo *db.Repository, chat *core.ChatService, summarizer *core.Summarizer, notifier *db.Notifier, messageCap int) (*Server, error) {
	tmplPath := filepath.Join("internal", "http", "templates", "*.html")
	tmpl, err := template.ParseGlob(tmplPath)
	if err != nil {
		return nil, err
	}
	return &Server{
		Repo:       repo,
		Chat:       chat,
		Summarizer: summarizer,
		Notifier:   notifier,
		Templates:  tmpl,
		MessageCap: messageCap,
	}, nil
}

// ServeHTTP dispatches incoming requests based on the URL path.  Minimal
// routing logic is implemented here to keep dependencies light.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := r.URL.Path
	switch {
	// Create a new session: POST /api/sessions
	case path == "/api/sessions" && r.Method == http.MethodPost:
		s.handleCreateSession(w, r.WithContext(ctx))
		return
	// Post a message: POST /api/sessions/{id}/messages
	case strings.HasPrefix(path, "/api/sessions/") && strings.HasSuffix(path, "/messages") && r.Method == http.MethodPost:
		// Extract session ID between /api/sessions/ and /messages
		parts := strings.Split(path, "/")
		if len(parts) < 4 {
			http.NotFound(w, r)
			return
		}
		sessionID := parts[3]
		s.handlePostMessage(w, r.WithContext(ctx), sessionID)
		return
	// Doctor API: list sessions as JSON
	case path == "/api/doctor/sessions" && r.Method == http.MethodGet:
		s.handleDoctorSessionsAPI(w, r.WithContext(ctx))
		return
	// Doctor API: get session detail JSON
	case strings.HasPrefix(path, "/api/doctor/sessions/") && r.Method == http.MethodGet:
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			sessionID := parts[4-1]
			// If the next segment is "stream" then this is SSE
			if len(parts) >= 5 && parts[4] == "stream" {
				s.handleDoctorSSE(w, r.WithContext(ctx), sessionID)
				return
			}
			s.handleDoctorSessionAPI(w, r.WithContext(ctx), sessionID)
			return
		}
	// Doctor HTML page
	case path == "/doctor" && r.Method == http.MethodGet:
		s.handleDoctorPage(w, r.WithContext(ctx))
		return
	// Doctor HTML detail page
	case strings.HasPrefix(path, "/doctor/sessions/") && r.Method == http.MethodGet:
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			sessionID := parts[3]
			s.handleDoctorSessionPage(w, r.WithContext(ctx), sessionID)
			return
		}
	// Patient HTML page: GET /patient/sessions/{id}
	case strings.HasPrefix(path, "/patient/sessions/") && r.Method == http.MethodGet:
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			sessionID := parts[3]
			s.handlePatientPage(w, r.WithContext(ctx), sessionID)
			return
		}
	default:
		http.NotFound(w, r)
	}
}

// handleCreateSession creates a new anonymous session.  It reads optional
// administrative fields from the request body (JSON) and returns a JSON
// response with the session ID.
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// For simplicity, ignore input and create a new session with default cap.
	sess, err := s.Repo.CreateSession(ctx, s.MessageCap, nil, nil, nil, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Start URL for patient UI
	startURL := "/patient/sessions/" + sess.ID
	resp := map[string]interface{}{
		"session_id": sess.ID,
		"start_url":  startURL,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handlePatientPage renders the chat interface for a patient session.
func (s *Server) handlePatientPage(w http.ResponseWriter, r *http.Request, sessionID string) {
	ctx := r.Context()
	// Load transcript to display existing messages
	transcript, err := s.Repo.GetTranscript(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		SessionID  string
		Transcript []pkg.Message
	}{
		SessionID:  sessionID,
		Transcript: transcript,
	}
	if err := s.Templates.ExecuteTemplate(w, "patient.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handlePostMessage processes a patient message, generates a bot reply, and
// returns a small HTML snippet representing the new messages to append to
// the transcript.  This endpoint is triggered via HTMX from the patient UI.
func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request, sessionID string) {
	ctx := r.Context()
	// Parse form to get content
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	content := r.FormValue("content")
	if strings.TrimSpace(content) == "" {
		http.Error(w, "empty message", http.StatusBadRequest)
		return
	}
	// Enforce message cap
	count, err := s.Repo.CountPatientMessages(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count >= s.MessageCap {
		// Reply with cap message only
		botMessage, _ := s.Repo.CreateMessage(ctx, sessionID, pkg.RoleBot, core.CapMessage)
		// Return HTML snippet for the cap message
		io.WriteString(w, `<div class="message bot">`+botMessage.Content+`</div>`)
		return
	}
	// Persist patient message
	_, err = s.Repo.CreateMessage(ctx, sessionID, pkg.RolePatient, content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Generate bot reply via LLM
	reply, _ := s.Chat.Reply(ctx, sessionID, content)
	// Persist bot message
	_, err = s.Repo.CreateMessage(ctx, sessionID, pkg.RoleBot, reply)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Trigger summarisation asynchronously (fire and forget)
	go func() {
		transcript, err := s.Repo.GetTranscript(context.Background(), sessionID)
		if err != nil {
			log.Println("failed to load transcript:", err)
			return
		}
		existing, _ := s.Repo.GetSummary(context.Background(), sessionID)
		summary, err := s.Summarizer.Summarize(context.Background(), sessionID, transcript, existing)
		if err != nil {
			log.Println("failed to summarise:", err)
		}
		if summary != nil {
			if err := s.Repo.UpsertSummary(context.Background(), summary); err != nil {
				log.Println("failed to upsert summary:", err)
			}
			// Notify doctors via Postgres channel (no-op in stub)
			_ = s.Notifier.Notify(context.Background(), sessionID)
		}
	}()
	// Return HTML snippet containing bot reply
	io.WriteString(w, `<div class="message bot">`+reply+`</div>`)
}

// handleDoctorPage renders the HTML dashboard for doctors.
func (s *Server) handleDoctorPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessions, err := s.Repo.ListActiveSessions(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Sessions []pkg.DoctorSessionPreview
	}{sessions}
	if err := s.Templates.ExecuteTemplate(w, "doctor.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleDoctorSessionPage renders the detail view for a single session.
func (s *Server) handleDoctorSessionPage(w http.ResponseWriter, r *http.Request, sessionID string) {
	ctx := r.Context()
	// Load session
	sess, err := s.Repo.GetSession(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Load summary
	summary, err := s.Repo.GetSummary(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Load transcript
	transcript, err := s.Repo.GetTranscript(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := struct {
		Session    *pkg.Session
		Summary    *pkg.Summary
		Transcript []pkg.Message
	}{sess, summary, transcript}
	// doctor_session.html is used as a partial fragment inserted into the
	// details area via HTMX
	if err := s.Templates.ExecuteTemplate(w, "doctor_session.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleDoctorSessionsAPI returns a JSON list of active sessions.  Doctors can
// consume this endpoint to build custom dashboards.
func (s *Server) handleDoctorSessionsAPI(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sessions, err := s.Repo.ListActiveSessions(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// handleDoctorSessionAPI returns a JSON representation of a session summary and transcript.
func (s *Server) handleDoctorSessionAPI(w http.ResponseWriter, r *http.Request, sessionID string) {
	ctx := r.Context()
	summary, err := s.Repo.GetSummary(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	transcript, err := s.Repo.GetTranscript(ctx, sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"summary":    summary,
		"transcript": transcript,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleDoctorSSE streams summary updates for a session using SSE.  The
// server sends a single event with the current summary and then exits.  In
// future this should subscribe to the Postgres NOTIFY channel and emit
// updates as they arrive.
func (s *Server) handleDoctorSSE(w http.ResponseWriter, r *http.Request, sessionID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	// Send initial event
	if err := s.sendSummaryEvent(w, sessionID); err != nil {
		log.Println("failed to send summary event:", err)
		return
	}
	flusher.Flush()
	// In stub mode, we do not keep the connection open.  A full implementation
	// would listen on the Notifier and write events until ctx.Done().
}

// sendSummaryEvent writes a summary_update event to the SSE response for the
// given session.  It serialises the summary and writes it as JSON after the
// "data:" prefix.
func (s *Server) sendSummaryEvent(w http.ResponseWriter, sessionID string) error {
	summary, err := s.Repo.GetSummary(context.Background(), sessionID)
	if err != nil {
		return err
	}
	if summary == nil {
		// no summary yet
		return nil
	}
	payload := map[string]interface{}{
		"type":       "summary_update",
		"session_id": sessionID,
		"key_points": summary.KeyPoints,
		"free_text":  summary.FreeText,
		"updated_at": summary.UpdatedAt,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "data: "+string(data)+"\n\n")
	return err
}

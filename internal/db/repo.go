package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"waitroom-chatbot/pkg"

	"github.com/google/uuid"
)

// Repository encapsulates all database operations.  Storing this behind an
// interface makes it easy to swap out the persistence layer for tests or
// future migrations (e.g. to a different database).
type Repository struct {
	DB *sql.DB
}

// NewRepository constructs a new Repository from an existing sql.DB.  The
// caller is responsible for opening and closing the database connection.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// CreateSession inserts a new session with a generated UUID and returns the
// created Session.  The message cap is read from the provided value or
// defaults to 50.
func (r *Repository) CreateSession(ctx context.Context, messageCap int, phone, nationalID, clientIP, userAgent *string) (*pkg.Session, error) {
	id := uuid.New().String()
	var sess pkg.Session
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO sessions (id, message_cap, patient_phone, patient_national_id, client_ip, user_agent)
         VALUES ($1, $2, $3, $4, $5, $6)
         RETURNING id, created_at, message_cap, patient_phone, patient_national_id, client_ip, user_agent`,
		id, messageCap, phone, nationalID, clientIP, userAgent,
	).Scan(&sess.ID, &sess.CreatedAt, &sess.MessageCap, &sess.PatientPhone, &sess.PatientID, &sess.ClientIP, &sess.UserAgent)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// GetSession retrieves a session by ID.  Returns sql.ErrNoRows if not found.
func (r *Repository) GetSession(ctx context.Context, id string) (*pkg.Session, error) {
	var sess pkg.Session
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, created_at, closed_at, message_cap, patient_phone, patient_national_id, client_ip, user_agent
         FROM sessions WHERE id = $1`, id,
	).Scan(&sess.ID, &sess.CreatedAt, &sess.ClosedAt, &sess.MessageCap, &sess.PatientPhone, &sess.PatientID, &sess.ClientIP, &sess.UserAgent)
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// CreateMessage inserts a new message for a session.
func (r *Repository) CreateMessage(ctx context.Context, sessionID string, role pkg.MessageRole, content string) (*pkg.Message, error) {
	var msg pkg.Message
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO messages (session_id, role, content)
         VALUES ($1, $2, $3)
         RETURNING id, session_id, role, content, created_at`,
		sessionID, role, content,
	).Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// CountPatientMessages returns the number of messages sent by the patient in a session.
func (r *Repository) CountPatientMessages(ctx context.Context, sessionID string) (int, error) {
	var count int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM messages WHERE session_id = $1 AND role = 'patient'`, sessionID,
	).Scan(&count)
	return count, err
}

// GetTranscript returns all messages for a session ordered by creation time.
func (r *Repository) GetTranscript(ctx context.Context, sessionID string) ([]pkg.Message, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, session_id, role, content, created_at
         FROM messages
         WHERE session_id = $1
         ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var transcript []pkg.Message
	for rows.Next() {
		var m pkg.Message
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		transcript = append(transcript, m)
	}
	return transcript, rows.Err()
}

// GetSummary retrieves the summary for a session.  Returns nil if no summary exists.
func (r *Repository) GetSummary(ctx context.Context, sessionID string) (*pkg.Summary, error) {
	var s pkg.Summary
	row := r.DB.QueryRowContext(ctx,
		`SELECT id, session_id, key_points, structured, free_text, updated_at
         FROM summaries WHERE session_id = $1`, sessionID)
	var keyPointsData []byte
	var structuredData []byte
	err := row.Scan(&s.ID, &s.SessionID, &keyPointsData, &structuredData, &s.FreeText, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Decode JSON fields
	if err := json.Unmarshal(keyPointsData, &s.KeyPoints); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(structuredData, &s.Structured); err != nil {
		return nil, err
	}
	return &s, nil
}

// UpsertSummary inserts or updates a summary for a session.
func (r *Repository) UpsertSummary(ctx context.Context, summary *pkg.Summary) error {
	keyPointsData, err := json.Marshal(summary.KeyPoints)
	if err != nil {
		return err
	}
	structuredData, err := json.Marshal(summary.Structured)
	if err != nil {
		return err
	}
	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO summaries (session_id, key_points, structured, free_text, updated_at)
         VALUES ($1, $2, $3, $4, $5)
         ON CONFLICT (session_id) DO UPDATE
           SET key_points = EXCLUDED.key_points,
               structured = EXCLUDED.structured,
               free_text = EXCLUDED.free_text,
               updated_at = EXCLUDED.updated_at`,
		summary.SessionID, keyPointsData, structuredData, summary.FreeText, summary.UpdatedAt)
	return err
}

// ListActiveSessions returns a list of sessions that are still open (closed_at is NULL).
// It also returns a preview of the summary and last message time for each session.
func (r *Repository) ListActiveSessions(ctx context.Context) ([]pkg.DoctorSessionPreview, error) {
	rows, err := r.DB.QueryContext(ctx, `
        SELECT s.id, COALESCE(su.key_points, '[]'::jsonb) AS key_points,
               su.updated_at, COALESCE(MAX(m.created_at), s.created_at) AS last_message
        FROM sessions s
        LEFT JOIN summaries su ON su.session_id = s.id
        LEFT JOIN messages m ON m.session_id = s.id
        WHERE s.closed_at IS NULL
        GROUP BY s.id, su.key_points, su.updated_at
        ORDER BY last_message DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []pkg.DoctorSessionPreview
	for rows.Next() {
		var preview pkg.DoctorSessionPreview
		var keyPointsData []byte
		if err := rows.Scan(&preview.SessionID, &keyPointsData, &preview.UpdatedAt, &preview.LastMessage); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(keyPointsData, &preview.KeyPoints); err != nil {
			return nil, err
		}
		list = append(list, preview)
	}
	return list, rows.Err()
}

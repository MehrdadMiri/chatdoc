package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"waitroom-chatbot/pkg"

	"github.com/google/uuid"
)

// Repository wraps database operations for users and messages.
// A single postgres database is used in this stub implementation.
type Repository struct {
	DB *sql.DB
}

// NewRepository constructs a new Repository from an existing sql.DB.
// The caller is responsible for managing the DB connection lifecycle.
func NewRepository(db *sql.DB) *Repository { return &Repository{DB: db} }

// UpsertUser creates or updates a session for the user identified by national ID.
func (r *Repository) UpsertUser(ctx context.Context, u *pkg.User) error {
	// Try to update the latest session with this national ID
	res, err := r.DB.ExecContext(ctx,
		`UPDATE sessions
         SET patient_phone = $1, patient_name = $2
         WHERE patient_national_id = $3`,
		u.Phone, u.Name, u.NationalID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		// Insert new session
		newID := uuid.New()
		_, err := r.DB.ExecContext(ctx,
			`INSERT INTO sessions (id, patient_national_id, patient_phone, patient_name)
             VALUES ($1, $2, $3, $4)`,
			newID, u.NationalID, u.Phone, u.Name,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetUser retrieves the most recent session for a user by national ID.
func (r *Repository) GetUser(ctx context.Context, nationalID string) (*pkg.User, error) {
	var u pkg.User
	err := r.DB.QueryRowContext(ctx,
		`SELECT patient_national_id, patient_phone, patient_name, created_at
         FROM sessions
         WHERE patient_national_id = $1
         ORDER BY created_at DESC
         LIMIT 1`,
		nationalID,
	).Scan(&u.NationalID, &u.Phone, &u.Name, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateMessage stores a new message for the given national ID.
func (r *Repository) CreateMessage(ctx context.Context, nationalID string, role pkg.MessageRole, content string) (*pkg.Message, error) {
	// Find the latest session ID for this nationalID
	var sessionID uuid.UUID
	err := r.DB.QueryRowContext(ctx,
		`SELECT id FROM sessions
         WHERE patient_national_id = $1
         ORDER BY created_at DESC
         LIMIT 1`, nationalID).Scan(&sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no session found for national ID %s", nationalID)
		}
		return nil, err
	}
	var m pkg.Message
	err = r.DB.QueryRowContext(ctx,
		`INSERT INTO messages (session_id, role, content)
         VALUES ($1, $2, $3)
         RETURNING id, role, content, created_at`,
		sessionID, role, content,
	).Scan(&m.ID, &m.Role, &m.Content, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	m.NationalID = nationalID
	return &m, nil
}

// GetTranscript returns messages from the last week for a user ordered by creation time.
func (r *Repository) GetTranscript(ctx context.Context, nationalID string) ([]pkg.Message, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT m.id, s.patient_national_id, m.role, m.content, m.created_at
         FROM messages m
         JOIN sessions s ON m.session_id = s.id
         WHERE s.patient_national_id = $1
           AND m.created_at >= NOW() - INTERVAL '7 days'
         ORDER BY m.created_at ASC`, nationalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var transcript []pkg.Message
	for rows.Next() {
		var m pkg.Message
		if err := rows.Scan(&m.ID, &m.NationalID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		transcript = append(transcript, m)
	}
	return transcript, rows.Err()
}

// CountUserMessagesThisWeek counts patient messages from the start of the
// current week (ISO week starting Monday) for usage‑cap enforcement.
func (r *Repository) CountUserMessagesThisWeek(ctx context.Context, nationalID string) (int, error) {
	var count int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*)
         FROM messages m
         JOIN sessions s ON m.session_id = s.id
         WHERE s.patient_national_id = $1
           AND m.role = 'patient'
           AND m.created_at >= date_trunc('week', NOW())`,
		nationalID,
	).Scan(&count)
	return count, err
}

// GetTranscriptSince returns the transcript for a nationalID but only messages
// with created_at >= since. It reuses GetTranscript and filters in-memory to
// avoid coupling to any specific SQL shape used by GetTranscript.
func (r *Repository) GetTranscriptSince(ctx context.Context, nationalID string, since time.Time) ([]pkg.Message, error) {
	all, err := r.GetTranscript(ctx, nationalID)
	if err != nil {
		return nil, err
	}
	out := make([]pkg.Message, 0, len(all))
	for _, m := range all {
		if m.CreatedAt.After(since) || m.CreatedAt.Equal(since) {
			out = append(out, m)
		}
	}
	return out, nil
}

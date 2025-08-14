package db

import (
	"context"
	"database/sql"

	"waitroom-chatbot/pkg"
)

// Repository wraps database operations for users and messages.
// A single postgres database is used in this stub implementation.
type Repository struct {
	DB *sql.DB
}

// NewRepository constructs a new Repository from an existing sql.DB.
// The caller is responsible for managing the DB connection lifecycle.
func NewRepository(db *sql.DB) *Repository { return &Repository{DB: db} }

// UpsertUser creates or updates a user identified by national ID.
func (r *Repository) UpsertUser(ctx context.Context, u *pkg.User) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO users (national_id, phone, name)
         VALUES ($1, $2, $3)
         ON CONFLICT (national_id) DO UPDATE
           SET phone = EXCLUDED.phone,
               name  = EXCLUDED.name`,
		u.NationalID, u.Phone, u.Name,
	)
	return err
}

// GetUser retrieves a user by national ID.
func (r *Repository) GetUser(ctx context.Context, nationalID string) (*pkg.User, error) {
	var u pkg.User
	err := r.DB.QueryRowContext(ctx,
		`SELECT national_id, phone, name, created_at FROM users WHERE national_id = $1`,
		nationalID,
	).Scan(&u.NationalID, &u.Phone, &u.Name, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateMessage stores a new message for the given national ID.
func (r *Repository) CreateMessage(ctx context.Context, nationalID string, role pkg.MessageRole, content string) (*pkg.Message, error) {
	var m pkg.Message
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO messages (national_id, role, content)
         VALUES ($1, $2, $3)
         RETURNING id, national_id, role, content, created_at`,
		nationalID, role, content,
	).Scan(&m.ID, &m.NationalID, &m.Role, &m.Content, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// GetTranscript returns messages from the last week for a user ordered by creation time.
func (r *Repository) GetTranscript(ctx context.Context, nationalID string) ([]pkg.Message, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, national_id, role, content, created_at
         FROM messages
         WHERE national_id = $1
           AND created_at >= NOW() - INTERVAL '7 days'
         ORDER BY created_at ASC`, nationalID)
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
		`SELECT COUNT(*) FROM messages
         WHERE national_id = $1 AND role = 'patient'
           AND created_at >= date_trunc('week', NOW())`,
		nationalID,
	).Scan(&count)
	return count, err
}

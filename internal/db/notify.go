package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
)

// Notifier wraps the LISTEN/NOTIFY mechanism in PostgreSQL.  It can send
// notifications when summaries are updated and listen for them on the
// doctor dashboard.  In this skeleton the functionality is simplified.
type Notifier struct {
	DB      *sql.DB
	Channel string
}

// NewNotifier constructs a new Notifier.  The channel should match the
// POSTGRES_NOTIFY_CHANNEL environment variable.
func NewNotifier(db *sql.DB, channel string) *Notifier {
	return &Notifier{DB: db, Channel: channel}
}

// Notify sends a notification to the specified channel with the session ID.
func (n *Notifier) Notify(ctx context.Context, sessionID string) error {
	channel := pq.QuoteIdentifier(n.Channel)
	_, err := n.DB.ExecContext(ctx, fmt.Sprintf("NOTIFY %s, $1", channel), sessionID)
	return err
}

// Listen blocks and yields session IDs as they are received on the channel.
// It returns a channel of strings.  In a real implementation you would
// terminate the goroutine when the context is cancelled.
func (n *Notifier) Listen(ctx context.Context) (<-chan string, error) {
	// Establish a separate connection to avoid interfering with other queries.
	conn, err := n.DB.Conn(ctx)
	if err != nil {
		return nil, err
	}
	// Issue a LISTEN command for the channel.
	channel := pq.QuoteIdentifier(n.Channel)
	if _, err := conn.ExecContext(ctx, fmt.Sprintf("LISTEN %s", channel)); err != nil {
		return nil, err
	}
	// Create a channel to deliver notifications.
	ch := make(chan string)
	go func() {
		defer func() {
			_ = conn.Close()
			close(ch)
		}()
		for {
			// Wait for a notification.  The underlying driver blocks until
			// a notification is available or the context is cancelled.
			// pq allows us to use WaitForNotification via a raw connection.
			// For simplicity we use QueryRow to check for notifications.
			// In production code, use pgx or LISTEN/NOTIFY support in pq.
			select {
			case <-ctx.Done():
				return
			default:
				var sessionID string
				// Using `SELECT 1` as a dummy to keep the connection alive.
				if err := conn.QueryRowContext(ctx, "SELECT 1").Scan(new(int)); err != nil {
					log.Println("notifier poll error:", err)
				}
				// Poll for notifications via pq listener (not implemented in stub).
				_ = sessionID
				// In this skeleton we do not deliver notifications.
			}
		}
	}()
	return ch, nil
}

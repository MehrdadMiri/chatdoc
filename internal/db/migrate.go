package db

import (
	"context"
	"database/sql"

	_ "embed"
)

//go:embed schema.sql
var schemaSQL string

// Migrate applies the database schema to the given database. It executes the
// statements in schema.sql which create tables and types if they do not
// already exist.
func Migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, schemaSQL)
	return err
}

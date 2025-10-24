package migration

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migration.sql
var migrationSQL string

// Migrate runs database migrations
func Migrate(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// check if tables exist
	var tableExists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'logs'
		)
	`).Scan(&tableExists)

	if err != nil {
		return fmt.Errorf("failed to check if tables exist: %w", err)
	}

	if tableExists {
		// tables already exist, skip migration
		return nil
	}

	_, err = db.Exec(ctx, migrationSQL)
	if err != nil {
		return fmt.Errorf("failed to run migration: %w", err)
	}

	return nil
}

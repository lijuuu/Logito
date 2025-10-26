package migration

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lijuuu/Logito/log-ingestor/internal/logger"
)

//go:embed migration.sql
var migrationSQL string

func Migrate(db *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var tableExists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'logs'
		)
	`).Scan(&tableExists)

	if err != nil {
		logger.Error("Failed to check if tables exist: %v", err)
		return fmt.Errorf("failed to check if tables exist: %w", err)
	}

	if tableExists {
		logger.Database("Tables already exist, skipping migration")
		return nil
	}

	logger.Database("Running database migration...")
	_, err = db.Exec(ctx, migrationSQL)
	if err != nil {
		logger.Error("Failed to run migration: %v", err)
		return fmt.Errorf("failed to run migration: %w", err)
	}

	logger.Database("Database migration completed successfully")
	return nil
}

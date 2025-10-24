package postgres

import (
	"time"
)

// Log represents the database model for logs table
type Log struct {
	ID           int64                  `db:"id"`
	Level        string                 `db:"level"`
	Message      string                 `db:"message"`
	ResourceID   string                 `db:"resourceId"`
	Timestamp    time.Time              `db:"timestamp"`
	TraceID      *string                `db:"traceId"`
	SpanID       *string                `db:"spanId"`
	Commit       *string                `db:"commit"`
	Metadata     map[string]interface{} `db:"metadata"`
	Indexed      bool                   `db:"indexed"`
	ProcessingAt *time.Time             `db:"processingAt"`
}

// Config represents database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

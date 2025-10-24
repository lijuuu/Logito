package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Client represents the postgres client with connection pool
type Client struct {
	pool   *pgxpool.Pool
	config Config
}

// NewClient creates a new postgres client with connection pool
func NewClient(config Config) (*Client, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.User, config.Password, config.Host, config.Port, config.DBName)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dsn: %w", err)
	}

	// configure connection pool
	poolConfig.MaxConns = int32(config.MaxOpenConns)
	poolConfig.MinConns = int32(config.MaxIdleConns)
	poolConfig.MaxConnLifetime = config.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{
		pool:   pool,
		config: config,
	}, nil
}

// Close closes the connection pool
func (c *Client) Close() {
	c.pool.Close()
}

// GetPool returns the underlying connection pool
func (c *Client) GetPool() *pgxpool.Pool {
	return c.pool
}

// InsertBatch inserts a batch of log entries using postgres copy command for maximum performance
func (c *Client) InsertBatch(ctx context.Context, entries []*logentry.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// use copy command for maximum performance
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// start copy transaction
	rowsAffected, err := conn.CopyFrom(ctx, pgx.Identifier{"logs"}, []string{
		"level", "message", "resource_id", "timestamp", "trace_id", "span_id",
		"commit", "metadata", "indexed", "processing_at",
	}, pgx.CopyFromSlice(len(entries), func(i int) ([]interface{}, error) {
		entry := entries[i]
		return []interface{}{
			entry.Level,
			entry.Message,
			entry.ResourceID,
			entry.Timestamp,
			entry.TraceID,
			entry.SpanID,
			entry.Commit,
			entry.Metadata,
			entry.Indexed,
			entry.ProcessingAt,
		}, nil
	}))

	if err != nil {
		return fmt.Errorf("failed to copy batch: %w", err)
	}

	if rowsAffected != int64(len(entries)) {
		return fmt.Errorf("expected %d rows affected, got %d", len(entries), rowsAffected)
	}

	return nil
}

// HealthCheck checks if the database connection is healthy
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.pool.Ping(ctx)
}

package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lijuuu/Logito/log-ingestor/internal/logger"
	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Client struct {
	pool   *pgxpool.Pool
	config Config
}

func NewClient(config Config) (*Client, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.User, config.Password, config.Host, config.Port, config.DBName)

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dsn: %w", err)
	}

	// Apply connection pool configuration from YAML
	poolConfig.MaxConns = int32(config.MaxOpenConns)
	poolConfig.MinConns = int32(config.MaxIdleConns)
	poolConfig.MaxConnLifetime = config.ConnMaxLifetime
	if config.ConnMaxIdleTime > 0 {
		poolConfig.MaxConnIdleTime = config.ConnMaxIdleTime
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Database("PostgreSQL connection pool initialized - MaxConns: %d, MinConns: %d, MaxLifetime: %v",
		config.MaxOpenConns, config.MaxIdleConns, config.ConnMaxLifetime)

	return &Client{
		pool:   pool,
		config: config,
	}, nil
}

func (c *Client) Close() {
	c.pool.Close()
}

func (c *Client) GetPool() *pgxpool.Pool {
	return c.pool
}

func (c *Client) InsertBatch(ctx context.Context, entries []*logentry.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		logger.Error("Failed to acquire database connection: %v", err)
		return fmt.Errorf("failed to acquire connection: %w", err)
	}
	defer conn.Release()

	// Use PostgreSQL COPY for bulk insert - much faster than individual INSERTs
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
		logger.Error("Failed to insert batch of %d entries: %v", len(entries), err)
		return fmt.Errorf("failed to copy batch: %w", err)
	}

	if rowsAffected != int64(len(entries)) {
		logger.Error("Batch insert mismatch - Expected: %d, Got: %d", len(entries), rowsAffected)
		return fmt.Errorf("expected %d rows affected, got %d", len(entries), rowsAffected)
	}

	logger.Database("Successfully inserted batch of %d entries", len(entries))
	return nil
}

func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.pool.Ping(ctx)
}

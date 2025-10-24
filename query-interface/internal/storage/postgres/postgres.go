package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lijuuu/Logito/query-interface/pkg/logentry"

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

	//configure connection pool
	poolConfig.MaxConns = int32(config.MaxOpenConns)
	poolConfig.MinConns = int32(config.MaxIdleConns)

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	//test connection
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

// GetUnindexedLogs fetches unindexed logs without marking them as processing
func (c *Client) GetUnindexedLogs(ctx context.Context, limit int) ([]*logentry.LogEntry, error) {
	// First, fetch unindexed logs that are not currently being processed
	// or have been stuck processing for more than 5 minutes
	query := `
		SELECT id, level, message, resource_id, timestamp, trace_id, span_id, commit, metadata, indexed, processing_at
		FROM logs 
		WHERE indexed = false 
		AND (processing_at IS NULL OR processing_at < NOW() - INTERVAL '5 minutes')
		ORDER BY timestamp ASC 
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`
	
	rows, err := c.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unindexed logs: %w", err)
	}
	defer rows.Close()

	var entries []*logentry.LogEntry
	for rows.Next() {
		entry := &logentry.LogEntry{}
		var id int64

		err := rows.Scan(
			&id,
			&entry.Level,
			&entry.Message,
			&entry.ResourceID,
			&entry.Timestamp,
			&entry.TraceID,
			&entry.SpanID,
			&entry.Commit,
			&entry.Metadata,
			&entry.Indexed,
			&entry.ProcessingAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}

		entry.ID = id
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return entries, nil
}

// MarkAsIndexed marks logs as indexed after successful processing
func (c *Client) MarkAsIndexed(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	query := `
		UPDATE logs
		SET indexed = true, processing_at = NULL
		WHERE id = ANY($1)
	`

	_, err := c.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("failed to mark logs as indexed: %w", err)
	}

	return nil
}

// MarkAsProcessing marks logs as being processed
func (c *Client) MarkAsProcessing(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	query := `
		UPDATE logs
		SET processing_at = NOW()
		WHERE id = ANY($1)
	`

	_, err := c.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("failed to mark logs as processing: %w", err)
	}

	return nil
}

// MarkAsFailed marks logs as failed to index
func (c *Client) MarkAsFailed(ctx context.Context, ids []int64, reason string) error {
	if len(ids) == 0 {
		return nil
	}

	//reset processing_at to allow retry
	query := `
		UPDATE logs
		SET processing_at = NULL
		WHERE id = ANY($1)
	`

	_, err := c.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("failed to mark logs as failed: %w", err)
	}

	return nil
}

// GetTotalCount returns the total number of logs in postgres
func (c *Client) GetTotalCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM logs`

	err := c.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total count: %w", err)
	}

	return count, nil
}

// GetIndexedCount returns the number of indexed logs in postgres
func (c *Client) GetIndexedCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM logs WHERE indexed = true`

	err := c.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get indexed count: %w", err)
	}

	return count, nil
}

// GetUnindexedCount returns the number of unindexed logs in postgres
func (c *Client) GetUnindexedCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM logs WHERE indexed = false`

	err := c.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unindexed count: %w", err)
	}

	return count, nil
}

// ClearAllLogs removes all logs from postgres (for force refresh)
func (c *Client) ClearAllLogs(ctx context.Context) error {
	query := `DELETE FROM logs`

	_, err := c.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear all logs: %w", err)
	}

	return nil
}

// ResetAllToUnindexed resets all logs to unindexed status (for force refresh)
func (c *Client) ResetAllToUnindexed(ctx context.Context) error {
	query := `UPDATE logs SET indexed = false, processing_at = NULL`

	_, err := c.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to reset all logs to unindexed: %w", err)
	}

	return nil
}

// HealthCheck checks if the database connection is healthy
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.pool.Ping(ctx)
}

package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/logger"
	"github.com/lijuuu/Logito/query-interface/pkg/logentry"

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Database("PostgreSQL connection pool initialized - MaxConns: %d, MinConns: %d", config.MaxOpenConns, config.MaxIdleConns)

	return &Client{
		pool:   pool,
		config: config,
	}, nil
}

func (c *Client) Close() {
	c.pool.Close()
}

// GetPool returns the underlying database pool for direct queries
func (c *Client) GetPool() *pgxpool.Pool {
	return c.pool
}

func (c *Client) GetUnindexedLogs(ctx context.Context, limit int) ([]*logentry.LogEntry, error) {

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

	if len(entries) > 0 {
		logger.Database("Fetched batch of unindexed logs - Size: %d", len(entries))
	}

	return entries, nil
}

func (c *Client) MarkAsIndexed(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	// Check if logs table exists first
	var tableExists bool
	checkQuery := `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'logs')`
	err := c.pool.QueryRow(ctx, checkQuery).Scan(&tableExists)
	if err != nil || !tableExists {
		return nil // Table doesn't exist yet, no need to update
	}

	query := `
		UPDATE logs
		SET indexed = true, processing_at = NULL
		WHERE id = ANY($1)
	`

	_, err = c.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("failed to mark logs as indexed: %w", err)
	}

	return nil
}

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

func (c *Client) MarkAsFailed(ctx context.Context, ids []int64, reason string) error {
	if len(ids) == 0 {
		return nil
	}

	query := `
		UPDATE logs
		SET indexed = false, processing_at = NULL
		WHERE id = ANY($1)
	`

	_, err := c.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("failed to mark logs as failed: %w", err)
	}

	return nil
}

func (c *Client) GetTotalCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM logs`

	err := c.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get total count: %w", err)
	}

	return count, nil
}

func (c *Client) GetIndexedCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM logs WHERE indexed = true`

	err := c.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get indexed count: %w", err)
	}

	return count, nil
}

func (c *Client) GetUnindexedCount(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM logs WHERE indexed = false`

	err := c.pool.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unindexed count: %w", err)
	}

	return count, nil
}

func (c *Client) ClearAllLogs(ctx context.Context) error {
	query := `DELETE FROM logs`

	_, err := c.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to clear all logs: %w", err)
	}

	return nil
}

func (c *Client) ResetAllToUnindexed(ctx context.Context) error {
	query := `UPDATE logs SET indexed = false, processing_at = NULL`

	_, err := c.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to reset all logs to unindexed: %w", err)
	}

	return nil
}

func (c *Client) ResetIndexedToUnindexed(ctx context.Context) error {
	query := `UPDATE logs SET indexed = false, processing_at = NULL WHERE indexed = true`

	_, err := c.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to reset indexed logs to unindexed: %w", err)
	}

	return nil
}

func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return c.pool.Ping(ctx)
}

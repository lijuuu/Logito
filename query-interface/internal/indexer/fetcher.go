package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/storage/postgres"
	"github.com/lijuuu/Logito/query-interface/pkg/logentry"
)

// Fetcher handles fetching unindexed logs from postgres
type Fetcher struct {
	dbClient *postgres.Client
}

// NewFetcher creates a new fetcher instance
func NewFetcher(dbClient *postgres.Client) *Fetcher {
	return &Fetcher{
		dbClient: dbClient,
	}
}

// FetchUnindexedLogs fetches logs that haven't been indexed yet
func (f *Fetcher) FetchUnindexedLogs(ctx context.Context, limit int) ([]*logentry.LogEntry, error) {

	entries, err := f.dbClient.GetUnindexedLogs(ctx, limit)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// MarkAsIndexed marks logs as indexed after successful es indexing
func (f *Fetcher) MarkAsIndexed(ctx context.Context, ids []int64) error {
	return f.dbClient.MarkAsIndexed(ctx, ids)
}

// MarkAsFailed marks logs as failed to index (for retry logic)
func (f *Fetcher) MarkAsFailed(ctx context.Context, ids []int64, reason string) error {
	return f.dbClient.MarkAsFailed(ctx, ids, reason)
}

// ResetAllToUnindexed resets all logs to unindexed status
func (f *Fetcher) ResetAllToUnindexed(ctx context.Context) error {
	return f.dbClient.ResetAllToUnindexed(ctx)
}

// GetTotalCount returns the estimated total number of logs in postgres
func (f *Fetcher) GetTotalCount(ctx context.Context) (int64, error) {
	return f.dbClient.GetTotalCount(ctx)
}

// GetCounts returns count statistics for postgres and elasticsearch
func (f *Fetcher) GetCounts(ctx context.Context, esClient *ESClient) (map[string]interface{}, error) {
	// get postgres counts
	totalInPostgres, err := f.dbClient.GetTotalCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total postgres count: %w", err)
	}

	indexedInPostgres, err := f.dbClient.GetIndexedCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get indexed postgres count: %w", err)
	}

	unindexedInPostgres, err := f.dbClient.GetUnindexedCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get unindexed postgres count: %w", err)
	}

	// get elasticsearch count
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size":             0,
		"track_total_hits": true,
	}

	esResponse, err := esClient.Search(ctx, esQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get elasticsearch count: %w", err)
	}

	totalInES := esResponse.Hits.Total.Value

	// remaining rows to index is simply the unindexed count in postgres
	remainingToIndex := unindexedInPostgres

	return map[string]interface{}{
		"totalInPostgres":     totalInPostgres,
		"indexedInPostgres":   indexedInPostgres,
		"unindexedInPostgres": unindexedInPostgres,
		"totalInES":           totalInES,
		"remainingToIndex":    remainingToIndex,
		"timestamp":           time.Now().UTC(),
	}, nil
}

// GetStats returns fetcher statistics
func (f *Fetcher) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// this would query the database for statistics
	// for now, return empty stats
	return map[string]interface{}{
		"lastFetch": time.Now(),
		"status":    "active",
	}, nil
}

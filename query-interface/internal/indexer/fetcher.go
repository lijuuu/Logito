package indexer

import (
	"context"
	"fmt"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/storage/postgres"
	"github.com/lijuuu/Logito/query-interface/pkg/logentry"
)

type Fetcher struct {
	dbClient *postgres.Client
}

func NewFetcher(dbClient *postgres.Client) *Fetcher {
	return &Fetcher{
		dbClient: dbClient,
	}
}

func (f *Fetcher) FetchUnindexedLogs(ctx context.Context, limit int) ([]*logentry.LogEntry, error) {
	entries, err := f.dbClient.GetUnindexedLogs(ctx, limit)
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (f *Fetcher) MarkAsIndexed(ctx context.Context, ids []int64) error {
	return f.dbClient.MarkAsIndexed(ctx, ids)
}

func (f *Fetcher) MarkAsFailed(ctx context.Context, ids []int64, reason string) error {
	return f.dbClient.MarkAsFailed(ctx, ids, reason)
}

func (f *Fetcher) ResetAllToUnindexed(ctx context.Context) error {
	return f.dbClient.ResetAllToUnindexed(ctx)
}

func (f *Fetcher) ResetIndexedToUnindexed(ctx context.Context) error {
	return f.dbClient.ResetIndexedToUnindexed(ctx)
}

func (f *Fetcher) GetTotalCount(ctx context.Context) (int64, error) {
	return f.dbClient.GetTotalCount(ctx)
}

func (f *Fetcher) GetCounts(ctx context.Context, esClient *ESClient) (map[string]interface{}, error) {
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

func (f *Fetcher) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return map[string]interface{}{
		"lastFetch": time.Now(),
		"status":    "active",
	}, nil
}

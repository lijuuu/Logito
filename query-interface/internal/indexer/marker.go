package indexer

import (
	"context"

	"github.com/lijuuu/Logito/query-interface/internal/storage/postgres"
)

// Marker handles marking logs as being processed
type Marker struct {
	dbClient *postgres.Client
}

// NewMarker creates a new marker instance
func NewMarker(dbClient *postgres.Client) *Marker {
	return &Marker{
		dbClient: dbClient,
	}
}

// MarkAsProcessing marks logs as being processed to prevent duplicate processing
func (m *Marker) MarkAsProcessing(ctx context.Context, ids []int64) error {
	//this would update the processingAt timestamp to currentTime
	return m.dbClient.MarkAsProcessing(ctx, ids)
}

// MarkAsIndexed marks logs as successfully indexed
func (m *Marker) MarkAsIndexed(ctx context.Context, ids []int64) error {
	return m.dbClient.MarkAsIndexed(ctx, ids)
}

func (m *Marker) MarkAsFailed(ctx context.Context, ids []int64, reason string) error {
	//this would update the processingAt = NULL
	return m.dbClient.MarkAsFailed(ctx, ids, reason)
}

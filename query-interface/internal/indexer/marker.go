package indexer

import (
	"context"

	"github.com/lijuuu/Logito/query-interface/internal/storage/postgres"
)

type Marker struct {
	dbClient *postgres.Client
}

func NewMarker(dbClient *postgres.Client) *Marker {
	return &Marker{
		dbClient: dbClient,
	}
}

func (m *Marker) MarkAsProcessing(ctx context.Context, ids []int64) error {
	return m.dbClient.MarkAsProcessing(ctx, ids)
}

func (m *Marker) MarkAsIndexed(ctx context.Context, ids []int64) error {
	return m.dbClient.MarkAsIndexed(ctx, ids)
}

func (m *Marker) MarkAsFailed(ctx context.Context, ids []int64, reason string) error {
	return m.dbClient.MarkAsFailed(ctx, ids, reason)
}

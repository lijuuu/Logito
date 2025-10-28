package dlq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lijuuu/Logito/query-interface/internal/config"
	"github.com/lijuuu/Logito/query-interface/internal/logger"
	"github.com/lijuuu/Logito/query-interface/internal/storage/postgres"
	"github.com/lijuuu/Logito/query-interface/pkg/logentry"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DLQClient struct {
	collection *mongo.Collection
	dbClient   *postgres.Client
}

type FailedMessage struct {
	ID        string    `json:"id"`
	Payload   string    `json:"payload"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

type DLQStats struct {
	TotalCount int `json:"total_count"`
}

func NewDLQClient(config *config.Config, dbClient *postgres.Client) (*DLQClient, error) {
	if !config.DLQ.Enabled {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.DLQ.Connection.URI))
	if err != nil {
		return nil, err
	}

	coll := client.Database(config.DLQ.Connection.Database).Collection(config.DLQ.Connection.Collection)
	logger.Database("Successfully connected to MongoDB DLQ - DB: %s, Collection: %s", config.DLQ.Connection.Database, config.DLQ.Connection.Collection)
	return &DLQClient{collection: coll, dbClient: dbClient}, nil
}

func (d *DLQClient) GetCount(ctx context.Context) (int64, error) {
	if d == nil {
		return 0, nil
	}

	count, err := d.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (d *DLQClient) GetMessages(ctx context.Context, limit int) ([]FailedMessage, error) {
	if d == nil {
		return []FailedMessage{}, nil
	}

	cur, err := d.collection.Find(ctx, bson.M{}, options.Find().SetLimit(int64(limit)).SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var msgs []FailedMessage
	for cur.Next(ctx) {
		var doc struct {
			ID        string    `bson:"_id"`
			Payload   string    `bson:"payload"`
			Reason    string    `bson:"reason"`
			CreatedAt time.Time `bson:"created_at"`
		}
		if err := cur.Decode(&doc); err != nil {
			continue
		}

		msgs = append(msgs, FailedMessage{
			ID:        doc.ID,
			Payload:   doc.Payload,
			Reason:    doc.Reason,
			CreatedAt: doc.CreatedAt,
		})
	}
	return msgs, nil
}

func (d *DLQClient) ForceAddAllMessages(ctx context.Context) error {
	if d == nil || d.dbClient == nil {
		return nil
	}

	cur, err := d.collection.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	var logEntries []logentry.LogEntry
	for cur.Next(ctx) {
		var doc struct {
			ID        string    `bson:"_id"`
			Payload   string    `bson:"payload"`
			Reason    string    `bson:"reason"`
			CreatedAt time.Time `bson:"created_at"`
		}
		if err := cur.Decode(&doc); err != nil {
			continue
		}

		var logEntry logentry.LogEntry
		if err := json.Unmarshal([]byte(doc.Payload), &logEntry); err != nil {
			continue
		}

		logEntries = append(logEntries, logEntry)
	}

	if len(logEntries) == 0 {
		_, err := d.collection.DeleteMany(ctx, bson.M{})
		return err
	}

	conn, err := d.dbClient.GetPool().Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"logs"}, []string{"level", "message", "resource_id", "timestamp", "trace_id", "span_id", "commit", "metadata", "indexed", "processing_at"}, pgx.CopyFromSlice(len(logEntries), func(i int) ([]interface{}, error) {
		entry := logEntries[i]
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
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	_, err = d.collection.DeleteMany(ctx, bson.M{})
	return err
}

func (d *DLQClient) AckMessage(ctx context.Context, id string) error {
	if d == nil {
		return nil
	}

	_, err := d.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

func (d *DLQClient) ClearAll(ctx context.Context) error {
	if d == nil {
		return nil
	}

	_, err := d.collection.DeleteMany(ctx, bson.M{})
	return err
}

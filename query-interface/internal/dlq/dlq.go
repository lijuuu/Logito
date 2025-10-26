package dlq

import (
	"context"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/config"
	"github.com/lijuuu/Logito/query-interface/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DLQClient struct {
	collection *mongo.Collection
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

func NewDLQClient(config *config.Config) (*DLQClient, error) {
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
	return &DLQClient{collection: coll}, nil
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
	if d == nil {
		return nil
	}

	// TODO: Implement force add all logic - add all messages directly to PostgreSQL
	// For now, just clear all messages
	_, err := d.collection.DeleteMany(ctx, bson.M{})
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

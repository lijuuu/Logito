package dlq

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lijuuu/Logito/log-ingestor/internal/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDLQ struct {
	collection *mongo.Collection
}

func NewMongoDLQ(connURI, dbName, collectionName string) (*MongoDLQ, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connURI))
	if err != nil {
		return nil, err
	}

	coll := client.Database(dbName).Collection(collectionName)
	logger.Database("Successfully connected to MongoDB DLQ - DB: %s, Collection: %s", dbName, collectionName)
	return &MongoDLQ{collection: coll}, nil
}

func (m *MongoDLQ) Send(payload any, reason string, failureType string) error {
	var payloadStr string
	switch v := payload.(type) {
	case string:
		payloadStr = v
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			logger.Error("DLQ: Failed to marshal JSON payload: %v", err)
			return err
		}
		payloadStr = string(jsonBytes)
	}
	doc := bson.M{
		"payload":      payloadStr,
		"reason":       reason,
		"failure_type": failureType,
		"created_at":   time.Now(),
	}
	result, err := m.collection.InsertOne(context.Background(), doc)
	if err != nil {
		logger.Error("DLQ: Failed to send message to DLQ: %v", err)
		return err
	}
	logger.Info("DLQ: Sent to DLQ - ID: %v, Type: %s", result.InsertedID, failureType)
	return nil
}

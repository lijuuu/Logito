package dlq

import (
	"time"

	"github.com/lijuuu/Logito/log-ingestor/internal/config"
)

type DLQ interface {
	Send(payload any, reason string, failureType string) error
}

type FailedMessage struct {
	ID          string
	Payload     []byte
	Reason      string
	FailureType string
	CreatedAt   time.Time
}

// Factory function that creates DLQ based on configuration type
func NewDLQ(config *config.Config) DLQ {
	if !config.DLQ.Enabled {
		return nil
	}

	switch config.DLQ.Type {
	// case "sql": //TODO
	// 	sqlDLQ, err := NewSQLDLQ(config.DLQ.Connection.URI)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	return sqlDLQ
	case "mongodb":
		mongoDLQ, err := NewMongoDLQ(config.DLQ.Connection.URI, config.DLQ.Connection.Database, config.DLQ.Connection.Collection)
		if err != nil {
			panic(err)
		}
		return mongoDLQ
	default:
		panic("unsupported dlq type")
	}
}

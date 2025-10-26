package dlq

import (
	"github.com/lijuuu/Logito/log-ingestor/internal/config"
)

func ShouldSendToDLQ(cfg *config.Config, failureType string) bool {
	if !cfg.DLQ.Enabled {
		return false
	}

	switch failureType {
	case FailureTypeParseError:
		return cfg.LogIngestor.DLQ.FailureTypes.ParseError
	case FailureTypeDBFailure:
		return cfg.LogIngestor.DLQ.FailureTypes.DBFailure
	case FailureTypeValidationError:
		return cfg.LogIngestor.DLQ.FailureTypes.ValidationError
	case FailureTypeTimeoutError:
		return cfg.LogIngestor.DLQ.FailureTypes.TimeoutError
	default:
		return false
	}
}

func SendToDLQIfEnabled(dlq DLQ, cfg *config.Config, payload any, reason string, failureType string) error {
	if !ShouldSendToDLQ(cfg, failureType) {
		return nil
	}
	return dlq.Send(payload, reason, failureType)
}

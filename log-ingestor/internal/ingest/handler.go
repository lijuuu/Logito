package ingest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lijuuu/Logito/log-ingestor/internal/config"
	"github.com/lijuuu/Logito/log-ingestor/internal/dlq"
	"github.com/lijuuu/Logito/log-ingestor/internal/logger"
	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

// Handler handles log ingestion requests
type Handler struct {
	batcher *Batcher
	pool    *ObjectPool
	dlq     dlq.DLQ
	config  *config.Config
}

// NewHandler creates a new log ingestion handler
func NewHandler(batcher *Batcher, pool *ObjectPool, dlq dlq.DLQ, cfg *config.Config) *Handler {
	return &Handler{
		batcher: batcher,
		pool:    pool,
		dlq:     dlq,
		config:  cfg,
	}
}

func (h *Handler) IngestLogs(c *gin.Context) {
	rawBody, _ := c.GetRawData()

	var requestBody interface{}

	if err := json.Unmarshal(rawBody, &requestBody); err != nil {
		logger.Error("Invalid JSON format in request: %v", err)

		if h.dlq != nil {
			dlq.SendToDLQIfEnabled(h.dlq, h.config, string(rawBody), "JSON parsing failed: "+err.Error(), dlq.FailureTypeParseError)
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json format"})
		return
	}

	var logs []map[string]interface{}

	switch v := requestBody.(type) {
	case map[string]interface{}:
		logs = []map[string]interface{}{v}
	case []interface{}:
		for _, item := range v {
			if logMap, ok := item.(map[string]interface{}); ok {
				logs = append(logs, logMap)
			} else {
				logger.Error("Invalid log entry format in array")

				// Send invalid log entry to DLQ
				if h.dlq != nil {
					dlq.SendToDLQIfEnabled(h.dlq, h.config, item, "Invalid log entry format in array", dlq.FailureTypeParseError)
				}

				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid log entry format in array"})
				return
			}
		}
	default:
		logger.Error("Invalid request body type")

		// Send invalid request body to DLQ
		if h.dlq != nil {
			dlq.SendToDLQIfEnabled(h.dlq, h.config, requestBody, "Invalid request body type", dlq.FailureTypeParseError)
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "request body must be a log object or array of logs"})
		return
	}

	const maxBatchSize = 5000
	if len(logs) > maxBatchSize {
		logger.Error("Batch size too large: %d (max: %d)", len(logs), maxBatchSize)

		// Send oversized batch to DLQ
		if h.dlq != nil {
			dlq.SendToDLQIfEnabled(h.dlq, h.config, logs, "Batch size too large", dlq.FailureTypeValidationError)
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "batch size too large, maximum 5000 logs per request"})
		return
	}

	validLogs := h.pool.GetLogEntrySlice()
	defer h.pool.PutLogEntrySlice(validLogs)

	invalidLogs := make([]map[string]interface{}, 0)

	for _, logData := range logs {
		entry, err := h.parseLogEntry(logData)
		if err != nil {
			invalidLogs = append(invalidLogs, logData)

			// Send invalid log to DLQ
			if h.dlq != nil {
				dlq.SendToDLQIfEnabled(h.dlq, h.config, logData, "Log parsing validation failed: "+err.Error(), dlq.FailureTypeValidationError)
			}
			continue
		}
		validLogs = append(validLogs, entry)
	}

	if len(validLogs) > 0 {
		h.batcher.AddLogs(validLogs)
	}

	response := gin.H{
		"message":   "logs processed",
		"total":     len(logs),
		"valid":     len(validLogs),
		"invalid":   len(invalidLogs),
		"timestamp": time.Now().UTC(),
	}

	if len(invalidLogs) > 0 {
		response["invalidLogs"] = invalidLogs
	}

	c.JSON(http.StatusOK, response)
}

// Direct field mapping to avoid JSON marshaling/unmarshaling overhead
func (h *Handler) parseLogEntry(logData map[string]interface{}) (*logentry.LogEntry, error) {
	entry := h.pool.GetLogEntry()

	if level, ok := logData["level"].(string); ok {
		entry.Level = level
	}
	if message, ok := logData["message"].(string); ok {
		entry.Message = message
	}
	if resourceID, ok := logData["resourceId"].(string); ok {
		entry.ResourceID = resourceID
	}
	if timestamp, ok := logData["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			entry.Timestamp = t
		}
	}
	if traceID, ok := logData["traceId"].(string); ok {
		entry.TraceID = &traceID
	}
	if spanID, ok := logData["spanId"].(string); ok {
		entry.SpanID = &spanID
	}
	if commit, ok := logData["commit"].(string); ok {
		entry.Commit = &commit
	}
	if metadata, ok := logData["metadata"].(map[string]interface{}); ok {
		entry.Metadata = metadata
	}

	if err := entry.Validate(); err != nil {
		h.pool.PutLogEntry(entry)
		return nil, err
	}

	entry.ProcessingAt = nil
	entry.Indexed = false

	return entry, nil
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "log-ingestor",
	})
}

func (h *Handler) GetQuickStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "running",
		"timestamp": time.Now().UTC(),
		"service":   "log-ingestor",
		"note":      "use /stats for detailed metrics",
	})
}

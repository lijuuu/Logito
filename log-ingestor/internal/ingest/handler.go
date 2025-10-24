package ingest

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

// Handler handles log ingestion requests
type Handler struct {
	batcher *Batcher
	pool    *ObjectPool
}

// NewHandler creates a new log ingestion handler
func NewHandler(batcher *Batcher, pool *ObjectPool) *Handler {
	return &Handler{
		batcher: batcher,
		pool:    pool,
	}
}

// IngestLogs handles POST /logs endpoint for single or bulk log ingestion
func (h *Handler) IngestLogs(c *gin.Context) {
	var requestBody interface{}

	// parse request body
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid json format",
		})
		return
	}

	// determine if it's a single log or array of logs
	var logs []map[string]interface{}

	switch v := requestBody.(type) {
	case map[string]interface{}:
		// single log entry
		logs = []map[string]interface{}{v}
	case []interface{}:
		// array of log entries
		for _, item := range v {
			if logMap, ok := item.(map[string]interface{}); ok {
				logs = append(logs, logMap)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "invalid log entry format in array",
				})
				return
			}
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request body must be a log object or array of logs",
		})
		return
	}

	// limit batch size to prevent memory issues
	const maxBatchSize = 1000
	if len(logs) > maxBatchSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "batch size too large, maximum 1000 logs per request",
		})
		return
	}

	// get slice from pool for better memory management
	validLogs := h.pool.GetLogEntrySlice()
	defer h.pool.PutLogEntrySlice(validLogs)

	invalidLogs := make([]map[string]interface{}, 0)

	for _, logData := range logs {
		entry, err := h.parseLogEntry(logData)
		if err != nil {
			//pass the invalid logs to DLQ
			invalidLogs = append(invalidLogs, logData)
			continue
		}

		validLogs = append(validLogs, entry)
	}

	// add valid logs to batcher
	if len(validLogs) > 0 {
		h.batcher.AddLogs(validLogs)
	}

	// return response
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

// parseLogEntry parses a log entry from map and validates it
func (h *Handler) parseLogEntry(logData map[string]interface{}) (*logentry.LogEntry, error) {
	// get entry from pool for reuse
	entry := h.pool.GetLogEntry()

	// use json-iterator for high-performance marshaling/unmarshaling
	jsonBytes, err := jsoniter.Marshal(logData)
	if err != nil {
		h.pool.PutLogEntry(entry) // return to pool on error
		return nil, err
	}

	// use json-iterator for high-performance unmarshaling
	if err := jsoniter.Unmarshal(jsonBytes, entry); err != nil {
		h.pool.PutLogEntry(entry) // return to pool on error
		return nil, err
	}

	// validate required fields
	if err := entry.Validate(); err != nil {
		h.pool.PutLogEntry(entry) // return to pool on error
		return nil, err
	}

	// set initial processing state
	entry.ProcessingAt = nil // start as unprocessed, so that indexer can pick it
	entry.Indexed = false

	return entry, nil
}

// HealthCheck handles health check endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "log-ingestor",
	})
}

// GetQuickStats returns basic stats without heavy locking
func (h *Handler) GetQuickStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "running",
		"timestamp": time.Now().UTC(),
		"service":   "log-ingestor",
		"note":      "use /stats for detailed metrics",
	})
}

// GetStats returns current batcher statistics
func (h *Handler) GetStats(c *gin.Context) {
	stats := h.batcher.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"batcher":   stats,
		"timestamp": time.Now().UTC(),
	})
}

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	LogIngestorURL = "http://localhost:3000"
	QueryAPIURL    = "http://localhost:4000"
	TestTimeout    = 30 * time.Second
)

// TestLogEntry represents a log entry for testing
type TestLogEntry struct {
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	ResourceID string                 `json:"resourceId"`
	Timestamp  string                 `json:"timestamp"`
	TraceID    *string                `json:"traceId,omitempty"`
	SpanID     *string                `json:"spanId,omitempty"`
	Commit     *string                `json:"commit,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// TestIngestResponse represents the response from log ingestion
type TestIngestResponse struct {
	Message     string                   `json:"message"`
	Total       int                      `json:"total"`
	Valid       int                      `json:"valid"`
	Invalid     int                      `json:"invalid"`
	Timestamp   string                   `json:"timestamp"`
	InvalidLogs []map[string]interface{} `json:"invalidLogs,omitempty"`
}

// TestSearchResponse represents the response from search API
type TestSearchResponse struct {
	Results    []map[string]interface{} `json:"results"`
	Total      int                      `json:"total"`
	Took       int                      `json:"took"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	HasNext    bool                     `json:"hasNext"`
	HasPrev    bool                     `json:"hasPrev"`
	TotalPages int                      `json:"totalPages"`
}

// TestHealthResponse represents health check response
type TestHealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
}

func TestEndToEndFlow_LogIngestionAndSearch(t *testing.T) {
	// Skip if services are not running
	if !isServiceRunning(LogIngestorURL) || !isServiceRunning(QueryAPIURL) {
		t.Skip("Services are not running. Start with: docker-compose up")
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	// Step 1: Test log ingestor health
	t.Run("LogIngestorHealth", func(t *testing.T) {
		resp, err := http.Get(LogIngestorURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health TestHealthResponse
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "healthy", health.Status)
		assert.Equal(t, "log-ingestor", health.Service)
	})

	// Step 2: Test query interface health
	t.Run("QueryInterfaceHealth", func(t *testing.T) {
		resp, err := http.Get(QueryAPIURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var health TestHealthResponse
		err = json.NewDecoder(resp.Body).Decode(&health)
		require.NoError(t, err)

		assert.Equal(t, "healthy", health.Status)
		assert.Equal(t, "query-interface", health.Service)
	})

	// Step 3: Ingest test logs
	t.Run("IngestLogs", func(t *testing.T) {
		testLogs := []TestLogEntry{
			{
				Level:      "info",
				Message:    "Application started successfully",
				ResourceID: "server-001",
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				TraceID:    stringPtr("trace-123"),
				SpanID:     stringPtr("span-456"),
				Commit:     stringPtr("abc123"),
				Metadata: map[string]interface{}{
					"parentResourceId": "parent-789",
					"version":          "1.0.0",
				},
			},
			{
				Level:      "error",
				Message:    "Database connection failed",
				ResourceID: "server-002",
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				TraceID:    stringPtr("trace-456"),
				SpanID:     stringPtr("span-789"),
				Commit:     stringPtr("def456"),
				Metadata: map[string]interface{}{
					"parentResourceId": "parent-012",
					"errorCode":        "DB_CONN_001",
				},
			},
			{
				Level:      "warn",
				Message:    "High memory usage detected",
				ResourceID: "server-001",
				Timestamp:  time.Now().UTC().Format(time.RFC3339),
				TraceID:    stringPtr("trace-789"),
				SpanID:     stringPtr("span-012"),
				Commit:     stringPtr("ghi789"),
			},
		}

		jsonData, err := json.Marshal(testLogs)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(ctx, "POST", LogIngestorURL+"/logs", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var ingestResp TestIngestResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestResp)
		require.NoError(t, err)

		assert.Equal(t, "logs processed", ingestResp.Message)
		assert.Equal(t, 3, ingestResp.Total)
		assert.Equal(t, 3, ingestResp.Valid)
		assert.Equal(t, 0, ingestResp.Invalid)
	})

	// Step 4: Wait for indexing (give some time for async processing)
	t.Run("WaitForIndexing", func(t *testing.T) {
		time.Sleep(5 * time.Second) // Wait for logs to be processed and indexed
	})

	// Step 5: Search for logs
	t.Run("SearchLogs", func(t *testing.T) {
		// Search for all logs
		resp, err := http.Get(QueryAPIURL + "/search?limit=10&page=1")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp TestSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&searchResp)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, searchResp.Total, 3) // Should have at least our test logs
		assert.GreaterOrEqual(t, len(searchResp.Results), 3)
		assert.Equal(t, 1, searchResp.Page)
		assert.Equal(t, 10, searchResp.Limit)
	})

	// Step 6: Search with filters
	t.Run("SearchWithFilters", func(t *testing.T) {
		// Search for error logs only
		resp, err := http.Get(QueryAPIURL + "/search?level=error&limit=10&page=1")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp TestSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&searchResp)
		require.NoError(t, err)

		// Should have at least one error log
		assert.GreaterOrEqual(t, searchResp.Total, 1)

		// All returned logs should be error level (case insensitive)
		for _, log := range searchResp.Results {
			if source, ok := log["_source"].(map[string]interface{}); ok {
				level := source["level"].(string)
				assert.Equal(t, "error", strings.ToLower(level))
			}
		}
	})

	// Step 7: Search by resource ID
	t.Run("SearchByResourceID", func(t *testing.T) {
		resp, err := http.Get(QueryAPIURL + "/search?resourceId=server-001&limit=10&page=1")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp TestSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&searchResp)
		require.NoError(t, err)

		// Should have at least 2 logs from server-001
		assert.GreaterOrEqual(t, searchResp.Total, 2)

		// All returned logs should be from server-001
		for _, log := range searchResp.Results {
			if source, ok := log["_source"].(map[string]interface{}); ok {
				assert.Equal(t, "server-001", source["resourceId"])
			}
		}
	})

	// Step 8: Search by message content
	t.Run("SearchByMessage", func(t *testing.T) {
		resp, err := http.Get(QueryAPIURL + "/search?message=database&limit=10&page=1")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var searchResp TestSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&searchResp)
		require.NoError(t, err)

		// Should find logs with "database" in message (either our test logs or existing ones)
		assert.GreaterOrEqual(t, searchResp.Total, 1)

		// Check that at least one message contains "database" (case insensitive)
		found := false
		for _, log := range searchResp.Results {
			if source, ok := log["_source"].(map[string]interface{}); ok {
				if message, ok := source["message"].(string); ok {
					// Check case insensitive
					if contains(strings.ToLower(message), "database") {
						found = true
						break
					}
				}
			}
		}
		assert.True(t, found, "Should find log with 'database' in message")
	})

	// Step 9: Test pagination
	t.Run("TestPagination", func(t *testing.T) {
		// Get first page
		resp1, err := http.Get(QueryAPIURL + "/search?limit=2&page=1")
		require.NoError(t, err)
		defer resp1.Body.Close()

		assert.Equal(t, http.StatusOK, resp1.StatusCode)

		var searchResp1 TestSearchResponse
		err = json.NewDecoder(resp1.Body).Decode(&searchResp1)
		require.NoError(t, err)

		// Get second page
		resp2, err := http.Get(QueryAPIURL + "/search?limit=2&page=2")
		require.NoError(t, err)
		defer resp2.Body.Close()

		assert.Equal(t, http.StatusOK, resp2.StatusCode)

		var searchResp2 TestSearchResponse
		err = json.NewDecoder(resp2.Body).Decode(&searchResp2)
		require.NoError(t, err)

		// Pages should have different results
		if len(searchResp1.Results) > 0 && len(searchResp2.Results) > 0 {
			id1 := searchResp1.Results[0]["_id"]
			id2 := searchResp2.Results[0]["_id"]
			assert.NotEqual(t, id1, id2)
		}
	})

	// Step 10: Test invalid search parameters
	t.Run("TestInvalidSearchParams", func(t *testing.T) {
		// Test invalid page
		resp, err := http.Get(QueryAPIURL + "/search?page=invalid")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// Test invalid limit
		resp, err = http.Get(QueryAPIURL + "/search?limit=2000")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestEndToEndFlow_InvalidLogHandling(t *testing.T) {
	// Skip if services are not running
	if !isServiceRunning(LogIngestorURL) {
		t.Skip("Log ingestor service is not running")
	}

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	// Test with invalid logs
	t.Run("IngestInvalidLogs", func(t *testing.T) {
		invalidLogs := []map[string]interface{}{
			{
				"level":      "info",
				"message":    "Valid log",
				"resourceId": "server-001",
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			},
			{
				"level":     "error",
				"message":   "Invalid log - missing resourceId",
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
			{
				"level":      "warn",
				"message":    "Invalid log - missing timestamp",
				"resourceId": "server-002",
			},
			{
				"message":    "Invalid log - missing level",
				"resourceId": "server-003",
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			},
		}

		jsonData, err := json.Marshal(invalidLogs)
		require.NoError(t, err)

		req, err := http.NewRequestWithContext(ctx, "POST", LogIngestorURL+"/logs", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var ingestResp TestIngestResponse
		err = json.NewDecoder(resp.Body).Decode(&ingestResp)
		require.NoError(t, err)

		assert.Equal(t, "logs processed", ingestResp.Message)
		assert.Equal(t, 4, ingestResp.Total)
		assert.Equal(t, 1, ingestResp.Valid)   // Only one valid log
		assert.Equal(t, 3, ingestResp.Invalid) // Three invalid logs
		assert.NotNil(t, ingestResp.InvalidLogs)
		assert.Len(t, ingestResp.InvalidLogs, 3)
	})
}

func TestEndToEndFlow_ServiceEndpoints(t *testing.T) {
	// Skip if services are not running
	if !isServiceRunning(LogIngestorURL) || !isServiceRunning(QueryAPIURL) {
		t.Skip("Services are not running")
	}

	// Test log ingestor stats endpoint
	t.Run("LogIngestorStats", func(t *testing.T) {
		resp, err := http.Get(LogIngestorURL + "/stats")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var stats map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&stats)
		require.NoError(t, err)

		assert.Contains(t, stats, "batcher")
	})

	// Test query interface sync status
	t.Run("QueryInterfaceSyncStatus", func(t *testing.T) {
		resp, err := http.Get(QueryAPIURL + "/sync-status")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var syncStatus map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&syncStatus)
		require.NoError(t, err)

		assert.Contains(t, syncStatus, "status")
	})

	// Test query interface metadata
	t.Run("QueryInterfaceMetadata", func(t *testing.T) {
		resp, err := http.Get(QueryAPIURL + "/metadata")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var metadata map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&metadata)
		require.NoError(t, err)

		assert.Contains(t, metadata, "metadata")
		if metadataData, ok := metadata["metadata"].(map[string]interface{}); ok {
			assert.Contains(t, metadataData, "levels")
			assert.Contains(t, metadataData, "resourceIds")
		}
	})

	// Test query interface counts
	t.Run("QueryInterfaceCounts", func(t *testing.T) {
		resp, err := http.Get(QueryAPIURL + "/counts")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var counts map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&counts)
		require.NoError(t, err)

		assert.Contains(t, counts, "counts")
		if countsData, ok := counts["counts"].(map[string]interface{}); ok {
			assert.Contains(t, countsData, "total")
			assert.Contains(t, countsData, "byLevel")
			assert.Contains(t, countsData, "byResource")
		}
	})
}

// Helper functions

func isServiceRunning(url string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func stringPtr(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}

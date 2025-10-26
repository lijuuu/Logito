package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/auth"
	"github.com/lijuuu/Logito/query-interface/internal/config"
	"github.com/lijuuu/Logito/query-interface/internal/dlq"
	"github.com/lijuuu/Logito/query-interface/internal/indexer"
	"github.com/lijuuu/Logito/query-interface/internal/logger"
	"github.com/lijuuu/Logito/query-interface/internal/worker"

	"github.com/gin-gonic/gin"
)

func validatePagination(page, limit int) error {
	maxOffset := 10000000
	offset := (page - 1) * limit
	if offset >= maxOffset {
		return fmt.Errorf("page %d with limit %d exceeds maximum offset of %d. Please use a smaller page number or larger limit", page, limit, maxOffset)
	}
	return nil
}

type Handler struct {
	esClient    *indexer.ESClient
	indexWorker *worker.IndexWorker
	fetcher     *indexer.Fetcher
	dlqClient   *dlq.DLQClient
	authService *auth.AuthService
	config      *config.Config
}

func NewHandler(esClient *indexer.ESClient, indexWorker *worker.IndexWorker, fetcher *indexer.Fetcher, dlqClient *dlq.DLQClient, authService *auth.AuthService, cfg *config.Config) *Handler {
	return &Handler{
		esClient:    esClient,
		indexWorker: indexWorker,
		fetcher:     fetcher,
		dlqClient:   dlqClient,
		authService: authService,
		config:      cfg,
	}
}

func (h *Handler) Search(c *gin.Context) {
	startTime := time.Now()

	message := c.Query("message")
	regex := c.Query("regex")
	level := c.Query("level")
	resourceId := c.Query("resourceId")
	traceId := c.Query("traceId")
	spanId := c.Query("spanId")
	commit := c.Query("commit")
	parentResourceId := c.Query("parentResourceId")

	page := 1
	limit := 10
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "page must be a positive integer",
			})
			return
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "limit must be a positive integer between 1 and 1000",
			})
			return
		}
	}

	if err := validatePagination(page, limit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	startTimeParam := c.Query("startTime")
	endTimeParam := c.Query("endTime")

	if regex != "" {
		if err := h.validateRegex(regex); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid regex pattern: %v", err),
			})
			return
		}
	}

	esQuery := h.buildSearchQuery(message, regex, level, resourceId, traceId, spanId, commit, parentResourceId, startTimeParam, endTimeParam, page, limit)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response, err := h.esClient.Search(ctx, esQuery)
	if err != nil {
		logger.Error("Search failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "search failed",
		})
		return
	}

	logger.Query("Search completed - Total: %d, Results: %d, Took: %dms, RequestDuration: %v",
		response.Hits.Total.Value, len(response.Hits.Hits), response.Took, time.Since(startTime))

	total := response.Hits.Total.Value
	totalPages := (total + int64(limit) - 1) / int64(limit)
	hasNext := int64(page) < totalPages
	hasPrev := page > 1

	c.JSON(http.StatusOK, gin.H{
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
		"hasNext":    hasNext,
		"hasPrev":    hasPrev,
		"results":    response.Hits.Hits,
		"took":       response.Took,
	})
}

func (h *Handler) buildSearchQuery(message, regex, level, resourceId, traceId, spanId, commit, parentResourceId, startTime, endTime string, page, limit int) map[string]interface{} {
	esQuery := map[string]interface{}{
		"from":             (page - 1) * limit,
		"size":             limit,
		"track_total_hits": true,
		"sort": []map[string]interface{}{
			{"timestamp": map[string]interface{}{"order": "desc"}},
		},
	}

	boolQuery := map[string]interface{}{
		"must": []map[string]interface{}{},
	}

	if message != "" {
		messageQuery := h.buildMessageQuery(message)
		if messageQuery != nil {
			boolQuery["must"] = append(boolQuery["must"].([]map[string]interface{}), messageQuery)
		}
	}

	if regex != "" {
		regexQuery := h.buildRegexQuery(regex)
		if regexQuery != nil {
			boolQuery["must"] = append(boolQuery["must"].([]map[string]interface{}), regexQuery)
		}
	}

	filters := []map[string]interface{}{}

	if level != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"level": level,
			},
		})
	}

	if resourceId != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"resourceId": resourceId,
			},
		})
	}

	if traceId != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"traceId": traceId,
			},
		})
	}

	if spanId != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"spanId": spanId,
			},
		})
	}

	if commit != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"commit": commit,
			},
		})
	}

	if parentResourceId != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"metadata.parentResourceId": parentResourceId,
			},
		})
	}

	if startTime != "" || endTime != "" {
		dateRange := map[string]interface{}{}
		if startTime != "" {
			dateRange["gte"] = startTime
		}
		if endTime != "" {
			dateRange["lte"] = endTime
		}

		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"timestamp": dateRange,
			},
		})
	}

	if len(filters) > 0 {
		boolQuery["filter"] = filters
	}

	esQuery["query"] = map[string]interface{}{
		"bool": boolQuery,
	}

	return esQuery
}

func (h *Handler) buildMetadataQuery(level, resourceId, startTime, endTime string) map[string]interface{} {
	esQuery := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"levels": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "level",
					"size":  100,
				},
			},
			"resourceIds": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "resourceId",
					"size":  100,
				},
			},
			"traceIds": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "traceId",
					"size":  100,
				},
			},
			"spanIds": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "spanId",
					"size":  100,
				},
			},
			"commits": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "commit",
					"size":  100,
				},
			},
			"parentResourceIds": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "metadata.parentResourceId.keyword",
					"size":  100,
				},
			},
			"timestamp_range": map[string]interface{}{
				"stats": map[string]interface{}{
					"field": "timestamp",
				},
			},
		},
	}

	// add filters if provided
	boolQuery := map[string]interface{}{
		"must": []map[string]interface{}{},
	}

	filters := []map[string]interface{}{}

	if level != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"level": level,
			},
		})
	}

	if resourceId != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"resourceId": resourceId,
			},
		})
	}

	if startTime != "" || endTime != "" {
		dateRange := map[string]interface{}{}
		if startTime != "" {
			dateRange["gte"] = startTime
		}
		if endTime != "" {
			dateRange["lte"] = endTime
		}

		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"timestamp": dateRange,
			},
		})
	}

	if len(filters) > 0 {
		boolQuery["filter"] = filters
	}

	esQuery["query"] = map[string]interface{}{
		"bool": boolQuery,
	}

	return esQuery
}

func (h *Handler) buildCountQuery(level, resourceId, startTime, endTime string) map[string]interface{} {
	esQuery := map[string]interface{}{
		"size": 0,
		"aggs": map[string]interface{}{
			"total_count": map[string]interface{}{
				"value_count": map[string]interface{}{
					"field": "id",
				},
			},
			"level_counts": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "level",
					"size":  100,
				},
			},
			"resource_counts": map[string]interface{}{
				"terms": map[string]interface{}{
					"field": "resourceId",
					"size":  100,
				},
			},
			"hourly_counts": map[string]interface{}{
				"date_histogram": map[string]interface{}{
					"field":             "timestamp",
					"calendar_interval": "hour",
					"min_doc_count":     0,
				},
			},
			"daily_counts": map[string]interface{}{
				"date_histogram": map[string]interface{}{
					"field":             "timestamp",
					"calendar_interval": "day",
					"min_doc_count":     0,
				},
			},
		},
	}

	// add filters if provided
	boolQuery := map[string]interface{}{
		"must": []map[string]interface{}{},
	}

	filters := []map[string]interface{}{}

	if level != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"level": level,
			},
		})
	}

	if resourceId != "" {
		filters = append(filters, map[string]interface{}{
			"term": map[string]interface{}{
				"resourceId": resourceId,
			},
		})
	}

	if startTime != "" || endTime != "" {
		dateRange := map[string]interface{}{}
		if startTime != "" {
			dateRange["gte"] = startTime
		}
		if endTime != "" {
			dateRange["lte"] = endTime
		}

		filters = append(filters, map[string]interface{}{
			"range": map[string]interface{}{
				"timestamp": dateRange,
			},
		})
	}

	if len(filters) > 0 {
		boolQuery["filter"] = filters
	}

	esQuery["query"] = map[string]interface{}{
		"bool": boolQuery,
	}

	return esQuery
}

func (h *Handler) extractMetadataFromResponse(response *indexer.SearchResponse) map[string]interface{} {
	metadata := make(map[string]interface{})

	if response.Aggregations != nil {
		aggregations := response.Aggregations
		if levels, ok := aggregations["levels"].(map[string]interface{}); ok {
			if buckets, ok := levels["buckets"].([]interface{}); ok {
				metadata["levels"] = buckets
			} else {
				metadata["levels"] = []interface{}{}
			}
		} else {
			metadata["levels"] = []interface{}{}
		}

		if resourceIds, ok := aggregations["resourceIds"].(map[string]interface{}); ok {
			if buckets, ok := resourceIds["buckets"].([]interface{}); ok {
				metadata["resourceIds"] = buckets
			} else {
				metadata["resourceIds"] = []interface{}{}
			}
		} else {
			metadata["resourceIds"] = []interface{}{}
		}

		if traceIds, ok := aggregations["traceIds"].(map[string]interface{}); ok {
			if buckets, ok := traceIds["buckets"].([]interface{}); ok {
				metadata["traceIds"] = buckets
			} else {
				metadata["traceIds"] = []interface{}{}
			}
		} else {
			metadata["traceIds"] = []interface{}{}
		}

		if spanIds, ok := aggregations["spanIds"].(map[string]interface{}); ok {
			if buckets, ok := spanIds["buckets"].([]interface{}); ok {
				metadata["spanIds"] = buckets
			} else {
				metadata["spanIds"] = []interface{}{}
			}
		} else {
			metadata["spanIds"] = []interface{}{}
		}

		if commits, ok := aggregations["commits"].(map[string]interface{}); ok {
			if buckets, ok := commits["buckets"].([]interface{}); ok {
				metadata["commits"] = buckets
			} else {
				metadata["commits"] = []interface{}{}
			}
		} else {
			metadata["commits"] = []interface{}{}
		}

		if parentResourceIds, ok := aggregations["parentResourceIds"].(map[string]interface{}); ok {
			if buckets, ok := parentResourceIds["buckets"].([]interface{}); ok {
				metadata["parentResourceIds"] = buckets
			} else {
				metadata["parentResourceIds"] = []interface{}{}
			}
		} else {
			metadata["parentResourceIds"] = []interface{}{}
		}

		if timestampRange, ok := aggregations["timestamp_range"].(map[string]interface{}); ok {
			metadata["timestampRange"] = timestampRange
		} else {
			metadata["timestampRange"] = map[string]interface{}{}
		}
	} else {
		metadata["levels"] = []interface{}{}
		metadata["resourceIds"] = []interface{}{}
		metadata["traceIds"] = []interface{}{}
		metadata["spanIds"] = []interface{}{}
		metadata["commits"] = []interface{}{}
		metadata["parentResourceIds"] = []interface{}{}
		metadata["timestampRange"] = map[string]interface{}{}
	}

	return metadata
}

func (h *Handler) extractCountsFromResponse(response *indexer.SearchResponse) map[string]interface{} {
	counts := make(map[string]interface{})

	if response.Aggregations != nil {
		aggregations := response.Aggregations
		if totalCount, ok := aggregations["total_count"].(map[string]interface{}); ok {
			if value, ok := totalCount["value"].(float64); ok {
				counts["total"] = int64(value)
			} else {
				counts["total"] = int64(0)
			}
		} else {
			counts["total"] = int64(0)
		}

		if levelCounts, ok := aggregations["level_counts"].(map[string]interface{}); ok {
			if buckets, ok := levelCounts["buckets"].([]interface{}); ok {
				counts["byLevel"] = buckets
			} else {
				counts["byLevel"] = []interface{}{}
			}
		} else {
			counts["byLevel"] = []interface{}{}
		}

		if resourceCounts, ok := aggregations["resource_counts"].(map[string]interface{}); ok {
			if buckets, ok := resourceCounts["buckets"].([]interface{}); ok {
				counts["byResource"] = buckets
			} else {
				counts["byResource"] = []interface{}{}
			}
		} else {
			counts["byResource"] = []interface{}{}
		}

		if hourlyCounts, ok := aggregations["hourly_counts"].(map[string]interface{}); ok {
			if buckets, ok := hourlyCounts["buckets"].([]interface{}); ok {
				counts["hourly"] = buckets
			} else {
				counts["hourly"] = []interface{}{}
			}
		} else {
			counts["hourly"] = []interface{}{}
		}

		if dailyCounts, ok := aggregations["daily_counts"].(map[string]interface{}); ok {
			if buckets, ok := dailyCounts["buckets"].([]interface{}); ok {
				counts["daily"] = buckets
			} else {
				counts["daily"] = []interface{}{}
			}
		} else {
			counts["daily"] = []interface{}{}
		}
	} else {
		counts["total"] = int64(0)
		counts["byLevel"] = []interface{}{}
		counts["byResource"] = []interface{}{}
		counts["hourly"] = []interface{}{}
		counts["daily"] = []interface{}{}
	}

	return counts
}

func (h *Handler) GetMetadata(c *gin.Context) {
	startTime := time.Now()

	level := c.Query("level")
	resourceId := c.Query("resourceId")
	startTimeParam := c.Query("startTime")
	endTimeParam := c.Query("endTime")

	esQuery := h.buildMetadataQuery(level, resourceId, startTimeParam, endTimeParam)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response, err := h.esClient.Search(ctx, esQuery)
	if err != nil {
		logger.Error("Metadata aggregation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "metadata aggregation failed",
			"details": err.Error(),
		})
		return
	}

	metadata := h.extractMetadataFromResponse(response)

	logger.Query("Metadata completed - Took: %dms, RequestDuration: %v, TotalHits: %d",
		response.Took, time.Since(startTime), response.Hits.Total.Value)

	if response.Hits.Total.Value == 0 {
		metadata = map[string]interface{}{
			"levels":            []interface{}{},
			"resourceIds":       []interface{}{},
			"traceIds":          []interface{}{},
			"spanIds":           []interface{}{},
			"commits":           []interface{}{},
			"parentResourceIds": []interface{}{},
			"timestampRange":    map[string]interface{}{},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"metadata": metadata,
		"took":     response.Took,
	})
}

func (h *Handler) GetCounts(c *gin.Context) {
	startTime := time.Now()

	level := c.Query("level")
	resourceId := c.Query("resourceId")
	startTimeParam := c.Query("startTime")
	endTimeParam := c.Query("endTime")

	esQuery := h.buildCountQuery(level, resourceId, startTimeParam, endTimeParam)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response, err := h.esClient.Search(ctx, esQuery)
	if err != nil {
		logger.Error("Count aggregation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "count aggregation failed",
		})
		return
	}

	counts := h.extractCountsFromResponse(response)

	logger.Query("Counts completed - Took: %dms, RequestDuration: %v",
		response.Took, time.Since(startTime))

	c.JSON(http.StatusOK, gin.H{
		"counts": counts,
		"took":   response.Took,
	})
}

func (h *Handler) GetLogEntry(c *gin.Context) {
	startTime := time.Now()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id parameter is required",
		})
		return
	}

	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"id": id,
			},
		},
		"size": 1,
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response, err := h.esClient.Search(ctx, esQuery)
	if err != nil {
		logger.Error("GetLogEntry failed for ID %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve log entry",
		})
		return
	}

	if len(response.Hits.Hits) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "log entry not found",
		})
		return
	}

	logger.Query("GetLogEntry completed - ID: %s, Took: %dms, RequestDuration: %v",
		id, response.Took, time.Since(startTime))

	c.JSON(http.StatusOK, gin.H{
		"logEntry": response.Hits.Hits[0].Source,
		"took":     response.Took,
	})
}

func (h *Handler) HealthCheck(c *gin.Context) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := h.esClient.HealthCheck(ctx)
	if err != nil {
		logger.Error("Health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	logger.Info("Health check completed - Status: healthy, RequestDuration: %v", time.Since(startTime))

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "query-interface",
	})
}

// ESHealthCheck provides detailed Elasticsearch health information
func (h *Handler) ESHealthCheck(c *gin.Context) {
	startTime := time.Now()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	healthStatus, err := h.esClient.GetHealthStatus(ctx)
	if err != nil {
		logger.Error("ES health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "unhealthy",
			"error":     err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
			"duration":  time.Since(startTime).String(),
		})
		return
	}

	// Get connection pool stats
	poolStats := h.esClient.GetPoolStats()

	logger.Info("ES health check completed - Status: %s, HealthScore: %.1f%%, Duration: %v",
		healthStatus.Status, healthStatus.HealthScore, time.Since(startTime))

	c.JSON(http.StatusOK, gin.H{
		"status":      healthStatus.Status,
		"healthScore": healthStatus.HealthScore,
		"healthy":     healthStatus.Healthy,
		"timestamp":   time.Now().Format(time.RFC3339),
		"duration":    time.Since(startTime).String(),
		"poolStats":   poolStats,
	})
}

func (h *Handler) GetSyncStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	counts, err := h.fetcher.GetCounts(ctx, h.esClient)
	if err != nil {
		logger.Error("Failed to get counts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get count statistics",
		})
		return
	}

	workerHealthy := h.indexWorker.HealthCheck(ctx) == nil

	status := "healthy"
	if counts["remainingToIndex"].(int64) > 0 {
		status = "syncing"
	}
	if !workerHealthy {
		status = "unhealthy"
	}

	counts["status"] = status
	counts["workerHealthy"] = workerHealthy

	c.JSON(http.StatusOK, counts)
}

func (h *Handler) ResetIndexingStatus(c *gin.Context) {
	userRole, exists := c.Get("user_role")
	if !exists || userRole != auth.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	logger.Info("Clearing Elasticsearch index...")
	err := h.esClient.DeleteIndex(ctx)
	if err != nil {
		logger.Error("Failed to clear Elasticsearch index: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to clear Elasticsearch index",
		})
		return
	}

	// Recreate the Elasticsearch index
	logger.Info("Recreating Elasticsearch index...")
	err = h.esClient.CreateIndex(ctx)
	if err != nil {
		logger.Error("Failed to recreate Elasticsearch index: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to recreate Elasticsearch index",
		})
		return
	}

	// Reset all indexed entries to unindexed in PostgreSQL
	logger.Info("Resetting PostgreSQL indexing status...")
	err = h.fetcher.ResetIndexedToUnindexed(ctx)
	if err != nil {
		logger.Error("Failed to reset indexing status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to reset indexing status",
		})
		return
	}

	logger.Info("Successfully reset indexing status - cleared ES index and reset PostgreSQL")
	c.JSON(http.StatusOK, gin.H{
		"message":   "Indexing status reset successfully - Elasticsearch cleared and PostgreSQL reset",
		"timestamp": time.Now().UTC(),
	})
}

func (h *Handler) buildMessageQuery(message string) map[string]interface{} {
	if message == "" {
		return nil
	}

	// For sentence search (with spaces), use match_phrase for better phrase matching
	if strings.Contains(message, " ") {
		return map[string]interface{}{
			"match_phrase": map[string]interface{}{
				"message": message,
			},
		}
	}

	// For single word search, use match
	return map[string]interface{}{
		"match": map[string]interface{}{
			"message": message,
		},
	}
}

func (h *Handler) validateRegex(pattern string) error {
	_, err := regexp.Compile(pattern)
	return err
}

func (h *Handler) buildRegexQuery(pattern string) map[string]interface{} {
	if pattern == "" {
		return nil
	}

	// Use bool query with should clauses to search across all text fields
	shouldClauses := []map[string]interface{}{
		{
			"regexp": map[string]interface{}{
				"message": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"level": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"resourceId": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"traceId": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"spanId": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"commit": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"metadata.parentResourceId": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"metadata.environment": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"metadata.service": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"metadata.version": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"metadata.userId": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"metadata.requestId": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
		{
			"regexp": map[string]interface{}{
				"metadata.errorCode": map[string]interface{}{
					"value": pattern,
					"flags": "ALL",
				},
			},
		},
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{
			"should": shouldClauses,
		},
	}
}

// DLQ Management Endpoints

func (h *Handler) GetDLQCount(c *gin.Context) {
	if h.dlqClient == nil {
		c.JSON(http.StatusOK, gin.H{
			"count":   0,
			"message": "DLQ is disabled",
		})
		return
	}

	count, err := h.dlqClient.GetCount(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get DLQ count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get DLQ count",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": count,
	})
}

func (h *Handler) GetDLQMessages(c *gin.Context) {
	if h.dlqClient == nil {
		c.JSON(http.StatusOK, gin.H{
			"messages": []dlq.FailedMessage{},
			"message":  "DLQ is disabled",
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 10
	}

	messages, err := h.dlqClient.GetMessages(c.Request.Context(), limit)
	if err != nil {
		logger.Error("Failed to get DLQ messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get DLQ messages",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messages,
		"count":    len(messages),
	})
}

func (h *Handler) ForceAddAllDLQMessages(c *gin.Context) {
	userRole, exists := c.Get("user_role")
	if !exists || userRole != auth.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	if h.dlqClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "DLQ is disabled",
		})
		return
	}

	err := h.dlqClient.ForceAddAllMessages(c.Request.Context())
	if err != nil {
		logger.Error("Failed to force add all DLQ messages: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to force add all messages",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All messages force added successfully",
	})
}

func (h *Handler) ClearDLQ(c *gin.Context) {
	userRole, exists := c.Get("user_role")
	if !exists || userRole != auth.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	if h.dlqClient == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "DLQ is disabled",
		})
		return
	}

	err := h.dlqClient.ClearAll(c.Request.Context())
	if err != nil {
		logger.Error("Failed to clear DLQ: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to clear DLQ",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "DLQ cleared successfully",
	})
}

// Authentication Endpoints

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Role  string `json:"role"`
	Email string `json:"email"`
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	role, err := h.authService.AuthenticateUser(req.Email, req.Password, h.config)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
		})
		return
	}

	token, err := h.authService.GenerateToken(req.Email, role)
	if err != nil {
		logger.Error("Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		Role:  string(role),
		Email: req.Email,
	})
}

func (h *Handler) GetProfile(c *gin.Context) {
	email, _ := c.Get("user_email")
	role, _ := c.Get("user_role")

	c.JSON(http.StatusOK, gin.H{
		"email": email,
		"role":  role,
	})
}

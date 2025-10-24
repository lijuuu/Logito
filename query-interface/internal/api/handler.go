package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lijuuu/Logito/query-interface/internal/indexer"
	"github.com/lijuuu/Logito/query-interface/internal/worker"

	"github.com/gin-gonic/gin"
)

// validatePagination validates pagination parameters and returns error if invalid
// supports high page numbers up to max_result_window limit (1,000,000)
func validatePagination(page, limit int) error {
	maxOffset := 10000000 // max_result_window setting in elasticsearch mapping
	offset := (page - 1) * limit
	if offset >= maxOffset {
		return fmt.Errorf("page %d with limit %d exceeds maximum offset of %d. Please use a smaller page number or larger limit", page, limit, maxOffset)
	}
	return nil
}

// handles query interface requests
type Handler struct {
	esClient    *indexer.ESClient
	indexWorker *worker.IndexWorker
	fetcher     *indexer.Fetcher
}

// creates a new query handler
func NewHandler(esClient *indexer.ESClient, indexWorker *worker.IndexWorker, fetcher *indexer.Fetcher) *Handler {
	return &Handler{
		esClient:    esClient,
		indexWorker: indexWorker,
		fetcher:     fetcher,
	}
}

// handles search requests
func (h *Handler) Search(c *gin.Context) {
	startTime := time.Now()

	// parse query parameters
	message := c.Query("message")
	regex := c.Query("regex")
	level := c.Query("level")
	resourceId := c.Query("resourceId")
	traceId := c.Query("traceId")
	spanId := c.Query("spanId")
	commit := c.Query("commit")
	parentResourceId := c.Query("parentResourceId")

	log.Printf("Search request - Message: '%s', Regex: '%s', Level: '%s', ResourceID: '%s', TraceID: '%s', SpanID: '%s', Commit: '%s', ParentResourceID: '%s'",
		message, regex, level, resourceId, traceId, spanId, commit, parentResourceId)

	// parse pagination
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

	// validate pagination limits
	if err := validatePagination(page, limit); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// parse date range
	startTimeParam := c.Query("startTime")
	endTimeParam := c.Query("endTime")

	log.Printf("Search pagination - Page: %d, Limit: %d, StartTime: %s, EndTime: %s",
		page, limit, startTimeParam, endTimeParam)

	// validate regex if provided
	if regex != "" {
		if err := h.validateRegex(regex); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid regex pattern: %v", err),
			})
			return
		}
	}

	// build elasticsearch query
	esQuery := h.buildSearchQuery(message, regex, level, resourceId, traceId, spanId, commit, parentResourceId, startTimeParam, endTimeParam, page, limit)

	// debug: log the elasticsearch query
	queryJSON, _ := json.MarshalIndent(esQuery, "", "  ")
	log.Printf("Elasticsearch Query: %s", string(queryJSON))

	// execute search
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response, err := h.esClient.Search(ctx, esQuery)
	if err != nil {
		log.Printf("Search failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "search failed",
		})
		return
	}

	// log search results
	log.Printf("Search completed - Total: %d, Results: %d, Took: %dms, RequestDuration: %v",
		response.Hits.Total.Value, len(response.Hits.Hits), response.Took, time.Since(startTime))

	// calculate pagination info
	total := response.Hits.Total.Value
	totalPages := (total + int64(limit) - 1) / int64(limit) // ceiling division
	hasNext := int64(page) < totalPages
	hasPrev := page > 1

	// format response
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

// buildSearchQuery builds the elasticsearch query from parameters
func (h *Handler) buildSearchQuery(message, regex, level, resourceId, traceId, spanId, commit, parentResourceId, startTime, endTime string, page, limit int) map[string]interface{} {
	// base query structure
	esQuery := map[string]interface{}{
		"from":             (page - 1) * limit,
		"size":             limit,
		"track_total_hits": true,
		"sort": []map[string]interface{}{
			{"timestamp": map[string]interface{}{"order": "desc"}},
		},
	}

	// build bool query
	boolQuery := map[string]interface{}{
		"must": []map[string]interface{}{},
	}

	// add message search to must clause
	if message != "" {
		messageQuery := h.buildMessageQuery(message)
		if messageQuery != nil {
			boolQuery["must"] = append(boolQuery["must"].([]map[string]interface{}), messageQuery)
		}
	}

	// add regex search to must clause
	if regex != "" {
		regexQuery := h.buildRegexQuery(regex)
		if regexQuery != nil {
			boolQuery["must"] = append(boolQuery["must"].([]map[string]interface{}), regexQuery)
		}
	}

	// add field filters
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

	// add date range filter
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

	// add filters to bool query
	if len(filters) > 0 {
		boolQuery["filter"] = filters
	}

	// add bool query to main query
	esQuery["query"] = map[string]interface{}{
		"bool": boolQuery,
	}

	return esQuery
}

// buildMetadataQuery builds an aggregation query for metadata
func (h *Handler) buildMetadataQuery(level, resourceId, startTime, endTime string) map[string]interface{} {
	esQuery := map[string]interface{}{
		"size": 0, // We only want aggregations, not documents
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
					"field": "metadata.parentResourceId",
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

	//Add date range filter
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

	//Add filters to bool query
	if len(filters) > 0 {
		boolQuery["filter"] = filters
	}

	//Add bool query to main query
	esQuery["query"] = map[string]interface{}{
		"bool": boolQuery,
	}

	return esQuery
}

// buildCountQuery builds an aggregation query for counts
func (h *Handler) buildCountQuery(level, resourceId, startTime, endTime string) map[string]interface{} {
	esQuery := map[string]interface{}{
		"size": 0, // We only want aggregations, not documents
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

	//Add date range filter
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

	//Add filters to bool query
	if len(filters) > 0 {
		boolQuery["filter"] = filters
	}

	//Add bool query to main query
	esQuery["query"] = map[string]interface{}{
		"bool": boolQuery,
	}

	return esQuery
}

// extractMetadataFromResponse extracts metadata from elasticsearch response
func (h *Handler) extractMetadataFromResponse(response *indexer.SearchResponse) map[string]interface{} {
	metadata := make(map[string]interface{})

	// extract aggregations from the actual es response
	if response.Aggregations != nil {
		aggregations := response.Aggregations
		// extract levels aggregation
		if levels, ok := aggregations["levels"].(map[string]interface{}); ok {
			if buckets, ok := levels["buckets"].([]interface{}); ok {
				metadata["levels"] = buckets
			} else {
				metadata["levels"] = []interface{}{}
			}
		} else {
			metadata["levels"] = []interface{}{}
		}

		// extract resourceids aggregation
		if resourceIds, ok := aggregations["resourceIds"].(map[string]interface{}); ok {
			if buckets, ok := resourceIds["buckets"].([]interface{}); ok {
				metadata["resourceIds"] = buckets
			} else {
				metadata["resourceIds"] = []interface{}{}
			}
		} else {
			metadata["resourceIds"] = []interface{}{}
		}

		// extract traceids aggregation
		if traceIds, ok := aggregations["traceIds"].(map[string]interface{}); ok {
			if buckets, ok := traceIds["buckets"].([]interface{}); ok {
				metadata["traceIds"] = buckets
			} else {
				metadata["traceIds"] = []interface{}{}
			}
		} else {
			metadata["traceIds"] = []interface{}{}
		}

		// extract spanids aggregation
		if spanIds, ok := aggregations["spanIds"].(map[string]interface{}); ok {
			if buckets, ok := spanIds["buckets"].([]interface{}); ok {
				metadata["spanIds"] = buckets
			} else {
				metadata["spanIds"] = []interface{}{}
			}
		} else {
			metadata["spanIds"] = []interface{}{}
		}

		// extract commits aggregation
		if commits, ok := aggregations["commits"].(map[string]interface{}); ok {
			if buckets, ok := commits["buckets"].([]interface{}); ok {
				metadata["commits"] = buckets
			} else {
				metadata["commits"] = []interface{}{}
			}
		} else {
			metadata["commits"] = []interface{}{}
		}

		// extract parentresourceids aggregation
		if parentResourceIds, ok := aggregations["parentResourceIds"].(map[string]interface{}); ok {
			if buckets, ok := parentResourceIds["buckets"].([]interface{}); ok {
				metadata["parentResourceIds"] = buckets
			} else {
				metadata["parentResourceIds"] = []interface{}{}
			}
		} else {
			metadata["parentResourceIds"] = []interface{}{}
		}

		// extract timestamp range stats
		if timestampRange, ok := aggregations["timestamp_range"].(map[string]interface{}); ok {
			metadata["timestampRange"] = timestampRange
		} else {
			metadata["timestampRange"] = map[string]interface{}{}
		}
	} else {
		// fallback to empty structure if no aggregations
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

// extractCountsFromResponse extracts counts from elasticsearch response
func (h *Handler) extractCountsFromResponse(response *indexer.SearchResponse) map[string]interface{} {
	counts := make(map[string]interface{})

	// extract aggregations from the actual es response
	if response.Aggregations != nil {
		aggregations := response.Aggregations
		// extract total count
		if totalCount, ok := aggregations["total_count"].(map[string]interface{}); ok {
			if value, ok := totalCount["value"].(float64); ok {
				counts["total"] = int64(value)
			} else {
				counts["total"] = int64(0)
			}
		} else {
			counts["total"] = int64(0)
		}

		// extract level counts
		if levelCounts, ok := aggregations["level_counts"].(map[string]interface{}); ok {
			if buckets, ok := levelCounts["buckets"].([]interface{}); ok {
				counts["byLevel"] = buckets
			} else {
				counts["byLevel"] = []interface{}{}
			}
		} else {
			counts["byLevel"] = []interface{}{}
		}

		// extract resource counts
		if resourceCounts, ok := aggregations["resource_counts"].(map[string]interface{}); ok {
			if buckets, ok := resourceCounts["buckets"].([]interface{}); ok {
				counts["byResource"] = buckets
			} else {
				counts["byResource"] = []interface{}{}
			}
		} else {
			counts["byResource"] = []interface{}{}
		}

		// extract hourly counts
		if hourlyCounts, ok := aggregations["hourly_counts"].(map[string]interface{}); ok {
			if buckets, ok := hourlyCounts["buckets"].([]interface{}); ok {
				counts["hourly"] = buckets
			} else {
				counts["hourly"] = []interface{}{}
			}
		} else {
			counts["hourly"] = []interface{}{}
		}

		// extract daily counts
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
		// fallback to empty structure if no aggregations
		counts["total"] = int64(0)
		counts["byLevel"] = []interface{}{}
		counts["byResource"] = []interface{}{}
		counts["hourly"] = []interface{}{}
		counts["daily"] = []interface{}{}
	}

	return counts
}

// GetMetadata handles metadata aggregation requests
func (h *Handler) GetMetadata(c *gin.Context) {
	startTime := time.Now()

	// parse query parameters for filtering
	level := c.Query("level")
	resourceId := c.Query("resourceId")
	startTimeParam := c.Query("startTime")
	endTimeParam := c.Query("endTime")

	log.Printf("Metadata request - Level: %s, ResourceID: %s, StartTime: %s, EndTime: %s",
		level, resourceId, startTimeParam, endTimeParam)

	// build aggregation query
	esQuery := h.buildMetadataQuery(level, resourceId, startTimeParam, endTimeParam)

	//Execute search
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response, err := h.esClient.Search(ctx, esQuery)
	if err != nil {
		log.Printf("Metadata aggregation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "metadata aggregation failed",
			"details": err.Error(),
		})
		return
	}

	// extract aggregations from response
	metadata := h.extractMetadataFromResponse(response)

	log.Printf("Metadata completed - Took: %dms, RequestDuration: %v, TotalHits: %d",
		response.Took, time.Since(startTime), response.Hits.Total.Value)

	// if no data in elasticsearch, return empty metadata structure
	if response.Hits.Total.Value == 0 {
		log.Printf("No data found in Elasticsearch, returning empty metadata")
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

// GetCounts handles count aggregation requests
func (h *Handler) GetCounts(c *gin.Context) {
	startTime := time.Now()

	// parse query parameters for filtering
	level := c.Query("level")
	resourceId := c.Query("resourceId")
	startTimeParam := c.Query("startTime")
	endTimeParam := c.Query("endTime")

	log.Printf("Counts request - Level: %s, ResourceID: %s, StartTime: %s, EndTime: %s",
		level, resourceId, startTimeParam, endTimeParam)

	// build count aggregation query
	esQuery := h.buildCountQuery(level, resourceId, startTimeParam, endTimeParam)

	//Execute search
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	response, err := h.esClient.Search(ctx, esQuery)
	if err != nil {
		log.Printf("Count aggregation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "count aggregation failed",
		})
		return
	}

	// extract counts from response
	counts := h.extractCountsFromResponse(response)

	log.Printf("Counts completed - Took: %dms, RequestDuration: %v",
		response.Took, time.Since(startTime))

	c.JSON(http.StatusOK, gin.H{
		"counts": counts,
		"took":   response.Took,
	})
}

// GetLogEntry handles single log entry retrieval
func (h *Handler) GetLogEntry(c *gin.Context) {
	startTime := time.Now()

	id := c.Param("id")
	if id == "" {
		log.Printf("GetLogEntry failed - missing id parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id parameter is required",
		})
		return
	}

	log.Printf("GetLogEntry request - ID: %s", id)

	// build query to get specific log entry
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"id": id,
			},
		},
		"size": 1,
	}

	//Execute search
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response, err := h.esClient.Search(ctx, esQuery)
	if err != nil {
		log.Printf("GetLogEntry failed for ID %s: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve log entry",
		})
		return
	}

	if len(response.Hits.Hits) == 0 {
		log.Printf("GetLogEntry - log entry not found for ID: %s", id)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "log entry not found",
		})
		return
	}

	log.Printf("GetLogEntry completed - ID: %s, Took: %dms, RequestDuration: %v",
		id, response.Took, time.Since(startTime))

	c.JSON(http.StatusOK, gin.H{
		"logEntry": response.Hits.Hits[0].Source,
		"took":     response.Took,
	})
}

// HealthCheck handles health check endpoint
func (h *Handler) HealthCheck(c *gin.Context) {
	startTime := time.Now()

	log.Printf("Health check request")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	err := h.esClient.HealthCheck(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	log.Printf("Health check completed - Status: healthy, RequestDuration: %v", time.Since(startTime))

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "query-interface",
	})
}

// GetSyncStatus returns the current index sync status
func (h *Handler) GetSyncStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// get count statistics
	counts, err := h.fetcher.GetCounts(ctx, h.esClient)
	if err != nil {
		log.Printf("Failed to get counts: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get count statistics",
		})
		return
	}

	// check if worker is healthy
	workerHealthy := h.indexWorker.HealthCheck(ctx) == nil

	status := "healthy"
	if counts["remainingToIndex"].(int64) > 0 {
		status = "syncing"
	}
	if !workerHealthy {
		status = "unhealthy"
	}

	// add status and worker health to response
	counts["status"] = status
	counts["workerHealthy"] = workerHealthy

	c.JSON(http.StatusOK, counts)
}

// buildMessageQuery builds a message query with support for proximity search
func (h *Handler) buildMessageQuery(message string) map[string]interface{} {
	if message == "" {
		return nil
	}

	// Check if the message contains proximity operators (+)
	if strings.Contains(message, "+") {
		// Split by + to get individual terms
		terms := strings.Split(message, "+")
		// Trim whitespace from each term
		for i, term := range terms {
			terms[i] = strings.TrimSpace(term)
		}

		// Filter out empty terms
		validTerms := []string{}
		for _, term := range terms {
			if term != "" {
				validTerms = append(validTerms, term)
			}
		}

		if len(validTerms) == 0 {
			return nil
		}

		if len(validTerms) == 1 {
			// Single term, use regular match
			return map[string]interface{}{
				"match": map[string]interface{}{
					"message": validTerms[0],
				},
			}
		}

		// Multiple terms, use span_near for proximity search
		// This ensures terms are close to each other (within 5 words by default)
		spanQueries := []map[string]interface{}{}
		for _, term := range validTerms {
			spanQueries = append(spanQueries, map[string]interface{}{
				"span_term": map[string]interface{}{
					"message": term,
				},
			})
		}

		return map[string]interface{}{
			"span_near": map[string]interface{}{
				"clauses":  spanQueries,
				"slop":     0,    // Terms must be adjacent (next to each other)
				"in_order": true, // Terms must be in the specified order
			},
		}
	}

	// Regular message search without proximity operators
	return map[string]interface{}{
		"match": map[string]interface{}{
			"message": message,
		},
	}
}

// validateRegex validates a regex pattern
func (h *Handler) validateRegex(pattern string) error {
	_, err := regexp.Compile(pattern)
	return err
}

// buildRegexQuery builds a regex query for elasticsearch
func (h *Handler) buildRegexQuery(pattern string) map[string]interface{} {
	if pattern == "" {
		return nil
	}

	// Use regexp query for pattern matching on the message field
	return map[string]interface{}{
		"regexp": map[string]interface{}{
			"message": map[string]interface{}{
				"value": pattern,
				"flags": "ALL",
			},
		},
	}
}

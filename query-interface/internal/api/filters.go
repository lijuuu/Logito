package api

import (
	"strings"
	"time"
)

// FilterConfig represents filter configuration
type FilterConfig struct {
	AllowedFields []string
}

// NewFilterConfig creates a new filter configuration
func NewFilterConfig(allowedFields []string) *FilterConfig {
	return &FilterConfig{
		AllowedFields: allowedFields,
	}
}

// ValidateField checks if a field is allowed for filtering
func (f *FilterConfig) ValidateField(field string) bool {
	for _, allowed := range f.AllowedFields {
		if field == allowed {
			return true
		}
	}
	return false
}

// ParseTimestampRange parses timestamp range from query parameters
func ParseTimestampRange(startTime, endTime string) (time.Time, time.Time, error) {
	var start, end time.Time
	var err error

	if startTime != "" {
		start, err = time.Parse(time.RFC3339, startTime)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if endTime != "" {
		end, err = time.Parse(time.RFC3339, endTime)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	return start, end, nil
}

// BuildFieldFilter builds a field filter for elasticsearch
func BuildFieldFilter(field, value string) map[string]interface{} {
	// handle nested fields (e.g., metadata.parentresourceid)
	if strings.Contains(field, ".") {
		return map[string]interface{}{
			"term": map[string]interface{}{
				field: value,
			},
		}
	}

	// handle exact match for keyword fields
	keywordFields := []string{"level", "resourceId", "traceId", "spanId", "commit"}
	for _, kf := range keywordFields {
		if field == kf {
			return map[string]interface{}{
				"term": map[string]interface{}{
					field: value,
				},
			}
		}
	}

	// handle text search for message field
	if field == "message" {
		return map[string]interface{}{
			"match": map[string]interface{}{
				"message": value,
			},
		}
	}

	// default to term match
	return map[string]interface{}{
		"term": map[string]interface{}{
			field: value,
		},
	}
}

// BuildDateRangeFilter builds a date range filter
func BuildDateRangeFilter(startTime, endTime time.Time) map[string]interface{} {
	dateRange := map[string]interface{}{}

	if !startTime.IsZero() {
		dateRange["gte"] = startTime.Format(time.RFC3339)
	}

	if !endTime.IsZero() {
		dateRange["lte"] = endTime.Format(time.RFC3339)
	}

	return map[string]interface{}{
		"range": map[string]interface{}{
			"timestamp": dateRange,
		},
	}
}

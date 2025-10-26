package api

import (
	"strings"
	"time"
)

type FilterConfig struct {
	AllowedFields []string
}

func NewFilterConfig(allowedFields []string) *FilterConfig {
	return &FilterConfig{
		AllowedFields: allowedFields,
	}
}

func (f *FilterConfig) ValidateField(field string) bool {
	for _, allowed := range f.AllowedFields {
		if field == allowed {
			return true
		}
	}
	return false
}

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

func BuildFieldFilter(field, value string) map[string]interface{} {
	if strings.Contains(field, ".") {
		return map[string]interface{}{
			"term": map[string]interface{}{
				field: value,
			},
		}
	}

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

	if field == "message" {
		return map[string]interface{}{
			"match": map[string]interface{}{
				"message": value,
			},
		}
	}

	return map[string]interface{}{
		"term": map[string]interface{}{
			field: value,
		},
	}
}

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

package logentry

import (
	"time"
)

type LogEntry struct {
	ID           int64                  `json:"id" db:"id"`
	Level        string                 `json:"level" db:"level"`
	Message      string                 `json:"message" db:"message"`
	ResourceID   string                 `json:"resourceId" db:"resource_id"`
	Timestamp    time.Time              `json:"timestamp" db:"timestamp"`
	TraceID      *string                `json:"traceId,omitempty" db:"trace_id"`
	SpanID       *string                `json:"spanId,omitempty" db:"span_id"`
	Commit       *string                `json:"commit,omitempty" db:"commit"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	Indexed      bool                   `json:"-" db:"indexed"`
	ProcessingAt *time.Time             `json:"-" db:"processing_at"`
}

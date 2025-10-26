package logentry

import (
	"encoding/json"
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

func (le *LogEntry) Validate() error {
	if le.Level == "" {
		return &ValidationError{Field: "level", Message: "level is required"}
	}
	if le.Message == "" {
		return &ValidationError{Field: "message", Message: "message is required"}
	}
	if le.ResourceID == "" {
		return &ValidationError{Field: "resourceId", Message: "resourceId is required"}
	}
	if le.Timestamp.IsZero() {
		return &ValidationError{Field: "timestamp", Message: "timestamp is required"}
	}
	return nil
}

func (le *LogEntry) ToJSON() ([]byte, error) {
	return json.Marshal(le)
}

type ValidationError struct {
	Field   string
	Message string
}

func (ve *ValidationError) Error() string {
	return ve.Message
}

type Batch struct {
	Entries []*LogEntry
	Size    int
}

func NewBatch(capacity int) *Batch {
	return &Batch{
		Entries: make([]*LogEntry, 0, capacity),
		Size:    0,
	}
}

func (b *Batch) Add(entry *LogEntry) {
	b.Entries = append(b.Entries, entry)
	b.Size++
}

func (b *Batch) IsFull(capacity int) bool {
	return b.Size >= capacity
}

func (b *Batch) IsEmpty() bool {
	return b.Size == 0
}

func (b *Batch) Clear() {
	b.Entries = b.Entries[:0]
	b.Size = 0
}

func (b *Batch) Reset() {
	b.Clear()
}

package logentry

import (
	"encoding/json"
	"time"
)

// LogEntry represents a single log entry with all required and optional fields
type LogEntry struct {
	ID         int64                  `json:"id" db:"id"`
	Level      string                 `json:"level" db:"level"`
	Message    string                 `json:"message" db:"message"`
	ResourceID string                 `json:"resourceId" db:"resource_id"`
	Timestamp  time.Time              `json:"timestamp" db:"timestamp"`
	TraceID    *string                `json:"traceId,omitempty" db:"trace_id"`
	SpanID     *string                `json:"spanId,omitempty" db:"span_id"`
	Commit     *string                `json:"commit,omitempty" db:"commit"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	// internal fields for processing
	Indexed      bool       `json:"-" db:"indexed"`
	ProcessingAt *time.Time `json:"-" db:"processing_at"`
}

// Validate checks if the log entry has all required fields
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

// ToJSON converts the log entry to JSON bytes
func (le *LogEntry) ToJSON() ([]byte, error) {
	return json.Marshal(le)
}

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field   string
	Message string
}

func (ve *ValidationError) Error() string {
	return ve.Message
}

// Batch represents a collection of log entries for batch processing
type Batch struct {
	Entries []*LogEntry
	Size    int
}

// NewBatch creates a new batch with the specified capacity
func NewBatch(capacity int) *Batch {
	return &Batch{
		Entries: make([]*LogEntry, 0, capacity),
		Size:    0,
	}
}

// Add adds a log entry to the batch
func (b *Batch) Add(entry *LogEntry) {
	b.Entries = append(b.Entries, entry)
	b.Size++
}

// IsFull checks if the batch has reached its capacity
func (b *Batch) IsFull(capacity int) bool {
	return b.Size >= capacity
}

// IsEmpty checks if the batch is empty
func (b *Batch) IsEmpty() bool {
	return b.Size == 0
}

// Clear clears the batch and resets the size
func (b *Batch) Clear() {
	b.Entries = b.Entries[:0]
	b.Size = 0
}

// Reset reuses the batch by clearing it
func (b *Batch) Reset() {
	b.Clear()
}

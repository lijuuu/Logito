package ingest

import (
	"time"

	"github.com/lijuuu/Logito/log-ingestor/pkg/logentry"
)

type Validator struct{}

func NewValidator() *Validator {
	return &Validator{}
}

func (v *Validator) ValidateLogEntry(entry *logentry.LogEntry) []string {
	var errors []string

	if entry.Level == "" {
		errors = append(errors, "level is required")
	}
	if entry.Message == "" {
		errors = append(errors, "message is required")
	}
	if entry.ResourceID == "" {
		errors = append(errors, "resourceId is required")
	}
	if entry.Timestamp.IsZero() {
		errors = append(errors, "timestamp is required")
	}

	if entry.Level != "" && !v.isValidLevel(entry.Level) {
		errors = append(errors, "level must be one of: debug, info, warn, error, fatal")
	}

	if !entry.Timestamp.IsZero() && entry.Timestamp.After(time.Now().Add(5*time.Minute)) {
		errors = append(errors, "timestamp cannot be more than 5 minutes in the future")
	}

	if !entry.Timestamp.IsZero() && entry.Timestamp.Before(time.Now().AddDate(-1, 0, 0)) {
		errors = append(errors, "timestamp cannot be more than 1 year old")
	}

	if len(entry.Message) > 10000 {
		errors = append(errors, "message cannot exceed 10000 characters")
	}

	if len(entry.ResourceID) > 255 {
		errors = append(errors, "resourceId cannot exceed 255 characters")
	}

	return errors
}

func (v *Validator) isValidLevel(level string) bool {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}
	return validLevels[level]
}

func (v *Validator) ValidateBatch(entries []*logentry.LogEntry) map[int][]string {
	errors := make(map[int][]string)

	for i, entry := range entries {
		if entryErrors := v.ValidateLogEntry(entry); len(entryErrors) > 0 {
			errors[i] = entryErrors
		}
	}

	return errors
}

package ingest

// validation.go — Ingest payload validation and sanitization
//
// Closes the audit gap: "Log injection, malformed payload DoS, no size limits".
//
// Every raw log line entering the pipeline MUST pass ValidateRawLine before
// being parsed or queued. This is a pure function — no allocations beyond the
// string itself — safe to call from multiple concurrent workers.

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	// MaxRawLineBytes is the maximum accepted size of a single raw log line.
	// Lines exceeding this are rejected to prevent heap exhaustion from crafted inputs.
	MaxRawLineBytes = 1 * 1024 * 1024 // 1 MB

	// MaxFieldValueBytes is the maximum value of any single extracted field.
	// Prevents individual fields (e.g. CommandLine, User) from becoming unbounded strings.
	MaxFieldValueBytes = 64 * 1024 // 64 KB

	// MaxMetadataKeys limits the number of parsed metadata keys per event.
	// Prevents deeply-nested JSON from becoming an O(n) map allocation bomb.
	MaxMetadataKeys = 256
)

var (
	// ErrPayloadTooLarge is returned when the raw line exceeds MaxRawLineBytes.
	ErrPayloadTooLarge = errors.New("ingest: payload exceeds maximum size limit (1MB)")

	// ErrInvalidUTF8 is returned when the payload contains invalid UTF-8 sequences.
	ErrInvalidUTF8 = errors.New("ingest: payload contains invalid UTF-8 encoding")

	// ErrNullByte is returned when the payload contains embedded null bytes.
	// Null bytes are used in some log-injection attacks to split parsers.
	ErrNullByte = errors.New("ingest: payload contains null byte (injection risk)")

	// ErrEmptyPayload is returned for zero-length or whitespace-only lines.
	ErrEmptyPayload = errors.New("ingest: empty payload rejected")
)

// ValidationError wraps a validation failure with the offending reason.
// Callers can type-assert to get the rejection reason for metrics.
type ValidationError struct {
	Reason error
	Offset int // byte offset where rejection was triggered (0 if not applicable)
}

func (e *ValidationError) Error() string {
	return e.Reason.Error()
}

func (e *ValidationError) Unwrap() error {
	return e.Reason
}

// ValidateRawLine validates a single raw log line before it enters the pipeline.
// It enforces:
//  1. Size limit — prevents heap exhaustion
//  2. UTF-8 validity — prevents parser confusion
//  3. No embedded null bytes — prevents log injection
//  4. Non-empty after trimming
//
// Returns nil if the line is safe to process, or a *ValidationError explaining
// the rejection. The rejection reason is tracked in pipeline drop metrics.
func ValidateRawLine(line string) error {
	// 1. Size limit — cheapest check first, O(1)
	if len(line) > MaxRawLineBytes {
		return &ValidationError{Reason: ErrPayloadTooLarge, Offset: MaxRawLineBytes}
	}

	// 2. Empty line
	if strings.TrimSpace(line) == "" {
		return &ValidationError{Reason: ErrEmptyPayload}
	}

	// 3. Null byte detection — strings.IndexByte is assembly-optimized
	if idx := strings.IndexByte(line, 0); idx >= 0 {
		return &ValidationError{Reason: ErrNullByte, Offset: idx}
	}

	// 4. UTF-8 validity — O(n), safe for all input
	if !utf8.ValidString(line) {
		// Find the first invalid byte for the offset field
		offset := 0
		for i, r := range line {
			if r == utf8.RuneError {
				offset = i
				break
			}
		}
		return &ValidationError{Reason: ErrInvalidUTF8, Offset: offset}
	}

	return nil
}

// SanitizeFieldValue truncates any extracted field value that exceeds
// MaxFieldValueBytes. Used by parsers when populating SovereignEvent fields.
// Returns the (possibly truncated) string and a bool indicating truncation.
func SanitizeFieldValue(value string) (string, bool) {
	if len(value) <= MaxFieldValueBytes {
		return value, false
	}
	return value[:MaxFieldValueBytes], true
}

// SanitizeMetadata enforces MaxMetadataKeys on a parsed metadata map.
// Excess keys (beyond limit) are silently dropped — the most common keys
// are typically populated first by parsers, so truncation is safe.
func SanitizeMetadata(meta map[string]string) map[string]string {
	if len(meta) <= MaxMetadataKeys {
		return meta
	}
	out := make(map[string]string, MaxMetadataKeys)
	count := 0
	for k, v := range meta {
		out[k] = v
		count++
		if count >= MaxMetadataKeys {
			break
		}
	}
	return out
}

// IsKnownInjectionPattern returns true if the line contains patterns commonly
// used in log injection attacks to insert fake log entries or split parsers.
// This is a best-effort heuristic, not a security boundary on its own.
func IsKnownInjectionPattern(line string) bool {
	for _, pattern := range injectionPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

// injectionPatterns are common log injection markers.
// Entries are checked via strings.Contains — keep this list short and high-signal.
var injectionPatterns = []string{
	"\nALERT:",      // Log line splitting
	"\r\nALERT:",
	"\n[CRITICAL]",
	"\r\n[CRITICAL]",
	"\n<14>",        // Fake syslog priority injection
	"\x1b[",         // ANSI escape sequences (terminal control injection)
	"%0a",           // URL-encoded newline
	"%0d%0a",        // URL-encoded CRLF
}

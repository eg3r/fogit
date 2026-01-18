package common

import (
	"strings"
)

// ContainsIgnoreCase checks if text contains substr (case-insensitive)
func ContainsIgnoreCase(text, substr string) bool {
	return strings.Contains(
		strings.ToLower(text),
		strings.ToLower(substr),
	)
}

// GetSnippet extracts a snippet around the query match
func GetSnippet(text, query string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 100
	}

	lower := strings.ToLower(text)
	queryLower := strings.ToLower(query)

	idx := strings.Index(lower, queryLower)
	if idx == -1 {
		// No match, return beginning
		if len(text) <= maxLen {
			return text
		}
		return text[:maxLen] + "..."
	}

	// Calculate window around match
	start := idx - maxLen/4
	if start < 0 {
		start = 0
	}

	end := start + maxLen
	if end > len(text) {
		end = len(text)
	}

	snippet := text[start:end]

	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}

	return snippet
}

// SplitKeyValue splits a string on the first occurrence of separator
func SplitKeyValue(s, sep string) (key, value string) {
	idx := strings.Index(s, sep)
	if idx == -1 {
		return strings.TrimSpace(s), ""
	}
	return strings.TrimSpace(s[:idx]), strings.TrimSpace(s[idx+len(sep):])
}

// SplitKeyValueEquals is a convenience function that splits on "="
func SplitKeyValueEquals(s string) (key, value string) {
	return SplitKeyValue(s, "=")
}

// TruncateWithEllipsis truncates text to maxLen and adds ellipsis
func TruncateWithEllipsis(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

// IsBlank returns true if string is empty or only whitespace
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotBlank returns true if string has non-whitespace content
func IsNotBlank(s string) bool {
	return !IsBlank(s)
}

// Coalesce returns the first non-empty string from the arguments
func Coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// CoalesceBlank returns the first non-blank string from the arguments
func CoalesceBlank(values ...string) string {
	for _, v := range values {
		if IsNotBlank(v) {
			return v
		}
	}
	return ""
}

// ContainsAny returns true if text contains any of the substrings
func ContainsAny(text string, substrings ...string) bool {
	for _, substr := range substrings {
		if strings.Contains(text, substr) {
			return true
		}
	}
	return false
}

// ContainsAnyIgnoreCase returns true if text contains any of the substrings (case-insensitive)
func ContainsAnyIgnoreCase(text string, substrings ...string) bool {
	lower := strings.ToLower(text)
	for _, substr := range substrings {
		if strings.Contains(lower, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// FirstNonEmpty returns the first non-empty string, or empty if all are empty
func FirstNonEmpty(strings ...string) string {
	return Coalesce(strings...)
}

// EqualFold compares two strings case-insensitively
func EqualFold(a, b string) bool {
	return strings.EqualFold(a, b)
}

// HasPrefix checks if s has the given prefix (case-insensitive)
func HasPrefixFold(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return strings.EqualFold(s[:len(prefix)], prefix)
}

// HasSuffix checks if s has the given suffix (case-insensitive)
func HasSuffixFold(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return strings.EqualFold(s[len(s)-len(suffix):], suffix)
}

// JoinNonEmpty joins non-empty strings with the given separator
func JoinNonEmpty(sep string, parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if p != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, sep)
}

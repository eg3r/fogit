package commands

import (
	"testing"
)

// Note: Version sorting tests are in pkg/fogit/feature_test.go (TestFeature_GetSortedVersionKeys)
// Note: Time formatting tests are in internal/common/time_test.go (TestFormatDateTime, TestFormatDurationLong)

// TestFormatVersionAuthors tests the local helper function for author display
func TestFormatVersionAuthors(t *testing.T) {
	tests := []struct {
		name     string
		authors  []string
		expected string
	}{
		{
			name:     "no authors",
			authors:  []string{},
			expected: "",
		},
		{
			name:     "single author",
			authors:  []string{"Alice"},
			expected: "Alice",
		},
		{
			name:     "multiple authors",
			authors:  []string{"Alice", "Bob", "Charlie"},
			expected: "Alice and 2 others",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatVersionAuthors(tt.authors)
			if result != tt.expected {
				t.Errorf("formatVersionAuthors() = %q, want %q", result, tt.expected)
			}
		})
	}
}

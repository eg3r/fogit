package common

import (
	"testing"
	"time"
)

func TestFormatDateTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "standard time",
			input:    time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC),
			expected: "2024-06-15 14:30:45",
		},
		{
			name:     "midnight",
			input:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: "2024-01-01 00:00:00",
		},
		{
			name:     "end of day",
			input:    time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: "2024-12-31 23:59:59",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDateTime(tt.input)
			if result != tt.expected {
				t.Errorf("FormatDateTime() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatDurationLong(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "days and hours",
			duration: 50 * time.Hour,
			expected: "2 days, 2 hours",
		},
		{
			name:     "hours only",
			duration: 5 * time.Hour,
			expected: "5 hours",
		},
		{
			name:     "minutes only",
			duration: 45 * time.Minute,
			expected: "45 minutes",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "0 minutes",
		},
		{
			name:     "one day",
			duration: 24 * time.Hour,
			expected: "1 days, 0 hours",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDurationLong(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDurationLong() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "seconds",
			duration: 30 * time.Second,
			expected: "30.0s",
		},
		{
			name:     "minutes",
			duration: 5 * time.Minute,
			expected: "5.0m",
		},
		{
			name:     "hours",
			duration: 2 * time.Hour,
			expected: "2.0h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "just now",
			input:    now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			input:    now.Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "5 minutes ago",
			input:    now.Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			input:    now.Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			input:    now.Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(tt.input)
			if result != tt.expected {
				t.Errorf("FormatTimeAgo() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	input := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	expected := "2024-06-15"
	result := FormatDate(input)
	if result != expected {
		t.Errorf("FormatDate() = %q, want %q", result, expected)
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  time.Duration
		expectErr bool
	}{
		{
			name:     "standard Go duration",
			input:    "2h30m",
			expected: 2*time.Hour + 30*time.Minute,
		},
		{
			name:     "days",
			input:    "2d",
			expected: 48 * time.Hour,
		},
		{
			name:     "weeks",
			input:    "1w",
			expected: 7 * 24 * time.Hour,
		},
		{
			name:      "invalid",
			input:     "invalid",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseDuration(tt.input)
			if tt.expectErr {
				if err == nil {
					t.Errorf("ParseDuration() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration() unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("ParseDuration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

package storage

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Basic cases
		{
			name:  "simple name",
			input: "User Authentication",
			want:  "user-authentication",
		},
		{
			name:  "lowercase preserved",
			input: "user authentication",
			want:  "user-authentication",
		},
		{
			name:  "uppercase converted",
			input: "USER AUTHENTICATION",
			want:  "user-authentication",
		},

		// Special characters
		{
			name:  "oauth version",
			input: "OAuth 2.0",
			want:  "oauth-2-0",
		},
		{
			name:  "parentheses removed",
			input: "Payment (Stripe)",
			want:  "payment-stripe",
		},
		{
			name:  "version numbers",
			input: "API v2.5.1",
			want:  "api-v2-5-1",
		},
		{
			name:  "slash removed",
			input: "API/REST",
			want:  "apirest",
		},
		{
			name:  "ampersand removed",
			input: "User & Admin",
			want:  "user-admin",
		},
		{
			name:  "plus signs removed",
			input: "C++ Module",
			want:  "c-module",
		},
		{
			name:  "hyphen preserved",
			input: "Server-Side",
			want:  "server-side",
		},
		{
			name:  "dots removed",
			input: "node.js API",
			want:  "node-js-api",
		},

		// Multiple spaces and hyphens
		{
			name:  "multiple spaces",
			input: "User   Authentication",
			want:  "user-authentication",
		},
		{
			name:  "leading/trailing hyphens",
			input: "-User Authentication-",
			want:  "user-authentication",
		},
		{
			name:  "leading/trailing spaces",
			input: "  User Authentication  ",
			want:  "user-authentication",
		},

		// Empty and special only
		{
			name:  "only special characters",
			input: "@#$%",
			want:  "",
		},
		{
			name:  "only spaces",
			input: "   ",
			want:  "",
		},
		{
			name:  "only hyphens",
			input: "---",
			want:  "",
		},

		// Unicode (non-ASCII removed)
		{
			name:  "unicode characters",
			input: "Café Module",
			want:  "caf-module",
		},
		{
			name:  "only unicode",
			input: "日本語",
			want:  "",
		},

		// Long names (truncation)
		{
			name:  "exactly 100 chars",
			input: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
			want:  "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
		},
		{
			name:  "over 100 chars truncated",
			input: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			want:  "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuv",
		},
		{
			name:  "truncate and trim hyphens",
			input: "very-long-feature-name-that-exceeds-the-maximum-length-limit-and-needs-to-be-truncated-properly-with-trailing-hyphens-removed",
			want:  "very-long-feature-name-that-exceeds-the-maximum-length-limit-and-needs-to-be-truncated-properly-with",
		},

		// Real world examples from spec
		{
			name:  "rest api version",
			input: "REST API v2",
			want:  "rest-api-v2",
		},
		{
			name:  "oauth integration",
			input: "OAuth 2.0 Integration",
			want:  "oauth-2-0-integration",
		},
		{
			name:  "dashboard with year",
			input: "New Dashboard (2025)",
			want:  "new-dashboard-2025",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		name          string
		featureName   string
		featureID     string
		existingFiles map[string]bool
		want          string
	}{
		{
			name:          "no collision",
			featureName:   "User Authentication",
			featureID:     "550e8400-e29b-41d4-a716-446655440000",
			existingFiles: map[string]bool{},
			want:          "user-authentication.yml",
		},
		{
			name:        "collision - append uuid prefix",
			featureName: "User Authentication",
			featureID:   "660e8400-e29b-41d4-a716-446655440001",
			existingFiles: map[string]bool{
				"user-authentication.yml": true,
			},
			want: "user-authentication-660e8400.yml",
		},
		{
			name:        "double collision - use full uuid",
			featureName: "User Authentication",
			featureID:   "770e8400-e29b-41d4-a716-446655440002",
			existingFiles: map[string]bool{
				"user-authentication.yml":          true,
				"user-authentication-770e8400.yml": true,
			},
			want: "user-authentication-770e8400-e29b-41d4-a716-446655440002.yml",
		},
		{
			name:          "empty slug - use feature prefix",
			featureName:   "@#$%",
			featureID:     "880e8400-e29b-41d4-a716-446655440003",
			existingFiles: map[string]bool{},
			want:          "feature-880e8400.yml",
		},
		{
			name:          "unicode only - use feature prefix",
			featureName:   "日本語",
			featureID:     "990e8400-e29b-41d4-a716-446655440004",
			existingFiles: map[string]bool{},
			want:          "feature-990e8400.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateFilename(tt.featureName, tt.featureID, tt.existingFiles)
			if got != tt.want {
				t.Errorf("generateFilename(%q, %q, ...) = %q, want %q", tt.featureName, tt.featureID, got, tt.want)
			}
		})
	}
}

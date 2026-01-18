package search

import (
	"testing"

	"github.com/lithammer/fuzzysearch/fuzzy"

	"github.com/eg3r/fogit/pkg/fogit"
)

// TestLevenshteinDistance tests the library's Levenshtein implementation
func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		expected int
	}{
		{"identical", "hello", "hello", 0},
		{"one insertion", "hello", "helo", 1},
		{"one deletion", "hello", "helloo", 1},
		{"one substitution", "hello", "hallo", 1},
		{"multiple changes", "kitten", "sitting", 3},
		{"empty strings", "", "", 0},
		{"one empty", "hello", "", 5},
		{"completely different", "abc", "xyz", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fuzzy.LevenshteinDistance(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("fuzzy.LevenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}

func TestCalculateSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		minScore float64 // Minimum expected score
	}{
		{"identical", "hello", "hello", 100.0},
		{"case insensitive", "Hello", "HELLO", 100.0},
		{"whitespace trimmed", " hello ", "hello", 100.0},
		{"typo - one char", "authentication", "authenticatian", 90.0},
		{"typo - two chars", "authentication", "authentcaton", 80.0},
		{"completely different", "abc", "xyz", 0.0},
		{"empty strings", "", "", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateSimilarity(tt.s1, tt.s2)
			if result < tt.minScore {
				t.Errorf("CalculateSimilarity(%q, %q) = %.2f, want >= %.2f", tt.s1, tt.s2, result, tt.minScore)
			}
		})
	}
}

func TestTokenBasedSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		s1       string
		s2       string
		minScore float64
	}{
		{"identical", "user authentication", "user authentication", 100.0},
		{"reordered words", "user authentication", "authentication user", 100.0},
		{"partial match", "user authentication system", "user authentication", 60.0},
		{"one matching word", "add user", "user login", 30.0},
		{"no matching words", "payment gateway", "user authentication", 0.0},
		{"with hyphens", "user-auth", "user auth", 80.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TokenBasedSimilarity(tt.s1, tt.s2)
			if result < tt.minScore {
				t.Errorf("TokenBasedSimilarity(%q, %q) = %.2f, want >= %.2f", tt.s1, tt.s2, result, tt.minScore)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"simple words", "user authentication", []string{"user", "authentication"}},
		{"with hyphens", "user-authentication", []string{"user", "authentication"}},
		{"with underscores", "user_authentication", []string{"user", "authentication"}},
		{"with slashes", "api/rest", []string{"api", "rest"}},
		{"filters short tokens", "a user b auth c", []string{"user", "auth"}},
		{"mixed separators", "user-auth_system/api", []string{"user", "auth", "system", "api"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenize(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("tokenize(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("tokenize(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestFindSimilar(t *testing.T) {
	// Create test features using NewFeature to properly initialize
	features := []*fogit.Feature{
		createTestFeature("1", "User Authentication", "Login and registration system", fogit.StateOpen),
		createTestFeature("2", "User Authorization", "Permission system", fogit.StateInProgress),
		createTestFeature("3", "Payment Gateway", "Process payments", fogit.StateClosed),
		createTestFeature("4", "Authentication API", "REST API for auth", fogit.StateOpen),
	}

	tests := []struct {
		name           string
		query          string
		config         SearchConfig
		expectMinCount int
		expectMaxCount int
		expectFirst    string // Expected first match name
	}{
		{
			name:           "typo in authentication",
			query:          "User Authenticatian",
			config:         DefaultSearchConfig(),
			expectMinCount: 1,
			expectMaxCount: 3,
			expectFirst:    "User Authentication",
		},
		{
			name:           "partial match",
			query:          "Auth",
			config:         DefaultSearchConfig(),
			expectMinCount: 0, // Short query may not match well
			expectMaxCount: 5,
			expectFirst:    "", // Don't care about order for short queries
		},
		{
			name:  "high similarity threshold",
			query: "User Auth",
			config: SearchConfig{
				FuzzyMatch:     true,
				MinSimilarity:  90.0,
				MaxSuggestions: 5,
			},
			expectMinCount: 0,
			expectMaxCount: 2,
			expectFirst:    "",
		},
		{
			name:  "fuzzy disabled",
			query: "User Authentication",
			config: SearchConfig{
				FuzzyMatch:     false,
				MinSimilarity:  60.0,
				MaxSuggestions: 5,
			},
			expectMinCount: 0,
			expectMaxCount: 0,
			expectFirst:    "",
		},
		{
			name:  "max suggestions limit",
			query: "authentication",
			config: SearchConfig{
				FuzzyMatch:     true,
				MinSimilarity:  50.0,
				MaxSuggestions: 2,
			},
			expectMinCount: 0,
			expectMaxCount: 2,
			expectFirst:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := FindSimilar(tt.query, features, tt.config)

			if len(matches) < tt.expectMinCount {
				t.Errorf("FindSimilar() returned %d matches, want >= %d", len(matches), tt.expectMinCount)
			}
			if len(matches) > tt.expectMaxCount {
				t.Errorf("FindSimilar() returned %d matches, want <= %d", len(matches), tt.expectMaxCount)
			}

			if tt.expectFirst != "" && len(matches) > 0 {
				if matches[0].Feature.Name != tt.expectFirst {
					t.Errorf("First match = %q, want %q", matches[0].Feature.Name, tt.expectFirst)
				}
			}

			// Verify matches are sorted by score (highest first)
			for i := 1; i < len(matches); i++ {
				if matches[i].Score > matches[i-1].Score {
					t.Errorf("Matches not sorted by score: match[%d] (%.2f) > match[%d] (%.2f)",
						i, matches[i].Score, i-1, matches[i-1].Score)
				}
			}

			// Verify all scores meet minimum threshold
			for i, match := range matches {
				if match.Score < tt.config.MinSimilarity {
					t.Errorf("Match[%d] score %.2f < minimum %.2f", i, match.Score, tt.config.MinSimilarity)
				}
			}
		})
	}
}

func TestSortMatches(t *testing.T) {
	matches := []Match{
		{Score: 50.0, Feature: &fogit.Feature{Name: "C"}},
		{Score: 90.0, Feature: &fogit.Feature{Name: "A"}},
		{Score: 70.0, Feature: &fogit.Feature{Name: "B"}},
	}

	sortMatches(matches)

	expected := []float64{90.0, 70.0, 50.0}
	for i, match := range matches {
		if match.Score != expected[i] {
			t.Errorf("After sort, match[%d].Score = %.2f, want %.2f", i, match.Score, expected[i])
		}
	}
}

// createTestFeature creates a feature with the given state for testing
func createTestFeature(id, name, description string, state fogit.State) *fogit.Feature {
	f := fogit.NewFeature(name)
	f.ID = id
	f.Description = description
	if state == fogit.StateInProgress {
		// Make it in-progress by updating modified timestamp
		v := f.GetCurrentVersion()
		v.ModifiedAt = v.CreatedAt.Add(1)
	} else if state == fogit.StateClosed {
		// Make it closed by setting closed_at
		f.UpdateState(fogit.StateClosed)
	}
	return f
}

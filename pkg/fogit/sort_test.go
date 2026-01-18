package fogit

import (
	"testing"
	"time"
)

func TestSortFeatures(t *testing.T) {
	now := time.Now()

	// Create test features with proper initialization
	f1 := NewFeature("Zulu")
	f1.SetPriority(PriorityLow)
	v1 := f1.GetCurrentVersion()
	v1.CreatedAt = now.Add(-3 * time.Hour)
	v1.ModifiedAt = now.Add(-1 * time.Hour)

	f2 := NewFeature("Alpha")
	f2.SetPriority(PriorityCritical)
	v2 := f2.GetCurrentVersion()
	v2.CreatedAt = now.Add(-1 * time.Hour)
	v2.ModifiedAt = now.Add(-3 * time.Hour)

	f3 := NewFeature("Bravo")
	f3.SetPriority(PriorityHigh)
	v3 := f3.GetCurrentVersion()
	v3.CreatedAt = now.Add(-2 * time.Hour)
	v3.ModifiedAt = now.Add(-2 * time.Hour)

	features := []*Feature{f1, f2, f3}

	tests := []struct {
		name     string
		filter   *Filter
		expected []string // Expected order of names
	}{
		{
			name:     "nil filter",
			filter:   nil,
			expected: []string{"Zulu", "Alpha", "Bravo"}, // No change
		},
		{
			name: "empty sort field",
			filter: &Filter{
				SortBy: "",
			},
			expected: []string{"Zulu", "Alpha", "Bravo"}, // No change
		},
		{
			name: "sort by name ascending",
			filter: &Filter{
				SortBy:    SortByName,
				SortOrder: SortAscending,
			},
			expected: []string{"Alpha", "Bravo", "Zulu"},
		},
		{
			name: "sort by name descending",
			filter: &Filter{
				SortBy:    SortByName,
				SortOrder: SortDescending,
			},
			expected: []string{"Zulu", "Bravo", "Alpha"},
		},
		{
			name: "sort by priority ascending",
			filter: &Filter{
				SortBy:    SortByPriority,
				SortOrder: SortAscending,
			},
			expected: []string{"Zulu", "Bravo", "Alpha"}, // low, high, critical
		},
		{
			name: "sort by priority descending",
			filter: &Filter{
				SortBy:    SortByPriority,
				SortOrder: SortDescending,
			},
			expected: []string{"Alpha", "Bravo", "Zulu"}, // critical, high, low
		},
		{
			name: "sort by created ascending",
			filter: &Filter{
				SortBy:    SortByCreated,
				SortOrder: SortAscending,
			},
			expected: []string{"Zulu", "Bravo", "Alpha"}, // oldest first
		},
		{
			name: "sort by created descending",
			filter: &Filter{
				SortBy:    SortByCreated,
				SortOrder: SortDescending,
			},
			expected: []string{"Alpha", "Bravo", "Zulu"}, // newest first
		},
		{
			name: "sort by modified ascending",
			filter: &Filter{
				SortBy:    SortByModified,
				SortOrder: SortAscending,
			},
			expected: []string{"Alpha", "Bravo", "Zulu"}, // oldest update first
		},
		{
			name: "sort by modified descending",
			filter: &Filter{
				SortBy:    SortByModified,
				SortOrder: SortDescending,
			},
			expected: []string{"Zulu", "Bravo", "Alpha"}, // newest update first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid mutating the original
			testFeatures := make([]*Feature, len(features))
			copy(testFeatures, features)

			SortFeatures(testFeatures, tt.filter)

			for i, expectedName := range tt.expected {
				if testFeatures[i].Name != expectedName {
					t.Errorf("position %d: got %s, want %s", i, testFeatures[i].Name, expectedName)
				}
			}
		})
	}
}

package features

import (
	"testing"
	"time"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestCalculateVersionDiff_NoDifferences(t *testing.T) {
	now := time.Now().UTC()

	feature := &fogit.Feature{
		ID:   "test-123",
		Name: "Test Feature",
	}

	v1 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		Branch:     "feature/test",
		Authors:    []string{"alice@example.com"},
		Notes:      "Test notes",
	}

	// Same version data
	v2 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		Branch:     "feature/test",
		Authors:    []string{"alice@example.com"},
		Notes:      "Test notes",
	}

	diff := CalculateVersionDiff(feature, "1", v1, "2", v2)

	if diff.HasDifferences {
		t.Errorf("expected no differences, but got %d changes", len(diff.Changes))
	}

	if diff.FeatureID != "test-123" {
		t.Errorf("expected feature ID %q, got %q", "test-123", diff.FeatureID)
	}

	if diff.FeatureName != "Test Feature" {
		t.Errorf("expected feature name %q, got %q", "Test Feature", diff.FeatureName)
	}
}

func TestCalculateVersionDiff_AllFieldsChanged(t *testing.T) {
	now := time.Now().UTC()
	later := now.Add(1 * time.Hour)
	closedTime := later.Add(2 * time.Hour)

	feature := &fogit.Feature{
		ID:   "test-456",
		Name: "Changed Feature",
	}

	v1 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		ClosedAt:   nil,
		Branch:     "feature/v1",
		Authors:    []string{"alice@example.com"},
		Notes:      "Version 1 notes",
	}

	v2 := &fogit.FeatureVersion{
		CreatedAt:  later,
		ModifiedAt: later.Add(30 * time.Minute),
		ClosedAt:   &closedTime,
		Branch:     "feature/v2",
		Authors:    []string{"bob@example.com", "charlie@example.com"},
		Notes:      "Version 2 notes",
	}

	diff := CalculateVersionDiff(feature, "1", v1, "2", v2)

	if !diff.HasDifferences {
		t.Error("expected differences, got none")
	}

	// Should have 6 changes: created_at, modified_at, closed_at, branch, authors, notes
	if len(diff.Changes) != 6 {
		t.Errorf("expected 6 changes, got %d", len(diff.Changes))
	}

	// Verify specific changes
	changeMap := make(map[string]FieldDiff)
	for _, change := range diff.Changes {
		changeMap[change.Field] = change
	}

	// Check closed_at was added
	if closedChange, ok := changeMap["closed_at"]; ok {
		if closedChange.ChangeType != "added" {
			t.Errorf("closed_at change type: expected 'added', got %q", closedChange.ChangeType)
		}
	} else {
		t.Error("missing closed_at change")
	}

	// Check branch was modified
	if branchChange, ok := changeMap["branch"]; ok {
		if branchChange.OldValue != "feature/v1" {
			t.Errorf("branch old value: expected 'feature/v1', got %q", branchChange.OldValue)
		}
		if branchChange.NewValue != "feature/v2" {
			t.Errorf("branch new value: expected 'feature/v2', got %q", branchChange.NewValue)
		}
	} else {
		t.Error("missing branch change")
	}
}

func TestCalculateVersionDiff_ClosedAtRemoved(t *testing.T) {
	now := time.Now().UTC()
	closedTime := now.Add(1 * time.Hour)

	feature := &fogit.Feature{
		ID:   "test-789",
		Name: "Reopened Feature",
	}

	v1 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		ClosedAt:   &closedTime, // Was closed
	}

	v2 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		ClosedAt:   nil, // Now open
	}

	diff := CalculateVersionDiff(feature, "1", v1, "2", v2)

	if !diff.HasDifferences {
		t.Error("expected differences")
	}

	// Find closed_at change
	var closedChange *FieldDiff
	for i := range diff.Changes {
		if diff.Changes[i].Field == "closed_at" {
			closedChange = &diff.Changes[i]
			break
		}
	}

	if closedChange == nil {
		t.Fatal("missing closed_at change")
	}

	if closedChange.ChangeType != "removed" {
		t.Errorf("expected change type 'removed', got %q", closedChange.ChangeType)
	}

	if closedChange.NewValue != "(open)" {
		t.Errorf("expected new value '(open)', got %q", closedChange.NewValue)
	}
}

func TestCalculateVersionDiff_BranchAdded(t *testing.T) {
	now := time.Now().UTC()

	feature := &fogit.Feature{
		ID:   "test-branch",
		Name: "Branch Test",
	}

	v1 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		Branch:     "", // No branch
	}

	v2 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		Branch:     "feature/new-branch", // Branch added
	}

	diff := CalculateVersionDiff(feature, "1", v1, "2", v2)

	// Find branch change
	var branchChange *FieldDiff
	for i := range diff.Changes {
		if diff.Changes[i].Field == "branch" {
			branchChange = &diff.Changes[i]
			break
		}
	}

	if branchChange == nil {
		t.Fatal("missing branch change")
	}

	if branchChange.ChangeType != "added" {
		t.Errorf("expected change type 'added', got %q", branchChange.ChangeType)
	}
}

func TestCalculateVersionDiff_AuthorsChanged(t *testing.T) {
	now := time.Now().UTC()

	feature := &fogit.Feature{
		ID:   "test-authors",
		Name: "Authors Test",
	}

	v1 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		Authors:    []string{"alice@example.com"},
	}

	v2 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		Authors:    []string{"alice@example.com", "bob@example.com"},
	}

	diff := CalculateVersionDiff(feature, "1", v1, "2", v2)

	// Find authors change
	var authorsChange *FieldDiff
	for i := range diff.Changes {
		if diff.Changes[i].Field == "authors" {
			authorsChange = &diff.Changes[i]
			break
		}
	}

	if authorsChange == nil {
		t.Fatal("missing authors change")
	}

	if authorsChange.OldValue != "alice@example.com" {
		t.Errorf("expected old value 'alice@example.com', got %q", authorsChange.OldValue)
	}

	if authorsChange.NewValue != "alice@example.com, bob@example.com" {
		t.Errorf("expected new value 'alice@example.com, bob@example.com', got %q", authorsChange.NewValue)
	}
}

func TestFormatClosedAt(t *testing.T) {
	tests := []struct {
		name     string
		closedAt *time.Time
		expected string
	}{
		{
			name:     "nil (open)",
			closedAt: nil,
			expected: "(open)",
		},
		{
			name: "closed",
			closedAt: func() *time.Time {
				t := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
				return &t
			}(),
			expected: "2024-06-15 14:30:45",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatClosedAt(tt.closedAt)
			if result != tt.expected {
				t.Errorf("FormatClosedAt() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatAuthors(t *testing.T) {
	tests := []struct {
		name     string
		authors  []string
		expected string
	}{
		{
			name:     "no authors",
			authors:  []string{},
			expected: "(none)",
		},
		{
			name:     "nil authors",
			authors:  nil,
			expected: "(none)",
		},
		{
			name:     "single author",
			authors:  []string{"alice@example.com"},
			expected: "alice@example.com",
		},
		{
			name:     "multiple authors",
			authors:  []string{"alice@example.com", "bob@example.com"},
			expected: "alice@example.com, bob@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatAuthors(tt.authors)
			if result != tt.expected {
				t.Errorf("FormatAuthors() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetChangeSymbol(t *testing.T) {
	tests := []struct {
		changeType string
		expected   string
	}{
		{"added", "[+]"},
		{"removed", "[-]"},
		{"modified", "[~]"},
		{"unknown", "[~]"},
	}

	for _, tt := range tests {
		t.Run(tt.changeType, func(t *testing.T) {
			result := GetChangeSymbol(tt.changeType)
			if result != tt.expected {
				t.Errorf("GetChangeSymbol(%q) = %q, want %q", tt.changeType, result, tt.expected)
			}
		})
	}
}

func TestVersionDiffVersionMetadata(t *testing.T) {
	now := time.Now().UTC()

	feature := &fogit.Feature{
		ID:   "meta-test-123",
		Name: "Metadata Test Feature",
	}

	v1 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
	}

	v2 := &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
	}

	diff := CalculateVersionDiff(feature, "1.0.0", v1, "2.0.0", v2)

	if diff.Version1 != "1.0.0" {
		t.Errorf("expected version1 '1.0.0', got %q", diff.Version1)
	}

	if diff.Version2 != "2.0.0" {
		t.Errorf("expected version2 '2.0.0', got %q", diff.Version2)
	}
}

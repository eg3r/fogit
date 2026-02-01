package validator

import (
	"context"
	"testing"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestValidator_EmptyRepository(t *testing.T) {
	cfg := fogit.DefaultConfig()

	v := New(nil, cfg) // repo not needed when features provided directly
	result, err := v.ValidateFeatures(context.Background(), []*fogit.Feature{})

	if err != nil {
		t.Fatalf("ValidateFeatures() error = %v, want nil", err)
	}

	if result.FeaturesCount != 0 {
		t.Errorf("FeaturesCount = %d, want 0", result.FeaturesCount)
	}

	if result.Errors != 0 {
		t.Errorf("Errors = %d, want 0", result.Errors)
	}

	if result.Warnings != 0 {
		t.Errorf("Warnings = %d, want 0", result.Warnings)
	}
}

func TestValidator_ValidFeature(t *testing.T) {
	feature := fogit.NewFeature("Test Feature")
	feature.SetPriority(fogit.PriorityMedium)

	features := []*fogit.Feature{feature}
	cfg := fogit.DefaultConfig()

	v := New(nil, cfg) // repo not needed when features provided directly
	result, err := v.ValidateFeatures(context.Background(), features)

	if err != nil {
		t.Fatalf("ValidateFeatures() error = %v, want nil", err)
	}

	if result.FeaturesCount != 1 {
		t.Errorf("FeaturesCount = %d, want 1", result.FeaturesCount)
	}

	if result.Errors != 0 {
		t.Errorf("Errors = %d, want 0", result.Errors)
	}
}

func TestValidator_OrphanedRelationship(t *testing.T) {
	feature := fogit.NewFeature("Feature with orphan")
	feature.Relationships = []fogit.Relationship{
		{
			Type:     "blocks",
			TargetID: "non-existent-id",
		},
	}

	features := []*fogit.Feature{feature}
	cfg := fogit.DefaultConfig()

	v := New(nil, cfg) // repo not needed when features provided directly
	result, err := v.ValidateFeatures(context.Background(), features)

	if err != nil {
		t.Fatalf("ValidateFeatures() error = %v, want nil", err)
	}

	if result.Errors == 0 {
		t.Error("Expected at least one error for orphaned relationship")
	}

	// Check for E001 issue
	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeOrphanedRelationship {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected E001 (orphaned relationship) issue")
	}
}

func TestValidationResult_HasErrors(t *testing.T) {
	tests := []struct {
		name   string
		result *ValidationResult
		want   bool
	}{
		{
			name:   "no errors",
			result: &ValidationResult{Errors: 0},
			want:   false,
		},
		{
			name:   "has errors",
			result: &ValidationResult{Errors: 1},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasErrors(); got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationResult_HasFixableIssues(t *testing.T) {
	tests := []struct {
		name   string
		result *ValidationResult
		want   bool
	}{
		{
			name:   "no issues",
			result: &ValidationResult{Issues: []ValidationIssue{}},
			want:   false,
		},
		{
			name: "non-fixable issue",
			result: &ValidationResult{
				Issues: []ValidationIssue{
					{Fixable: false},
				},
			},
			want: false,
		},
		{
			name: "fixable issue",
			result: &ValidationResult{
				Issues: []ValidationIssue{
					{Fixable: true},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasFixableIssues(); got != tt.want {
				t.Errorf("HasFixableIssues() = %v, want %v", got, tt.want)
			}
		})
	}
}

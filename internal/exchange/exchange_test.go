package exchange

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestConvertToExportFeature(t *testing.T) {
	now := time.Now().UTC()
	closedAt := now.Add(-1 * time.Hour)

	feature := &fogit.Feature{
		ID:          "test-id-123",
		Name:        "Test Feature",
		Description: "A test feature",
		Tags:        []string{"test", "example"},
		Files:       []string{"src/test.go"},
		Metadata: map[string]interface{}{
			"priority": "high",
			"type":     "enhancement",
		},
		Versions: map[string]*fogit.FeatureVersion{
			"1": {
				CreatedAt:  now.Add(-24 * time.Hour),
				ModifiedAt: now.Add(-12 * time.Hour),
				ClosedAt:   &closedAt,
				Branch:     "feature/test",
				Authors:    []string{"alice@test.com"},
				Notes:      "Initial version",
			},
			"2": {
				CreatedAt:  now,
				ModifiedAt: now,
				Branch:     "feature/test-v2",
			},
		},
		Relationships: []fogit.Relationship{
			{
				ID:          "rel-1",
				Type:        "depends-on",
				TargetID:    "target-id-456",
				TargetName:  "Target Feature",
				Description: "Depends on target",
				CreatedAt:   now,
				VersionConstraint: &fogit.VersionConstraint{
					Operator: ">=",
					Version:  2,
				},
			},
		},
	}

	featureIDs := map[string]bool{
		"test-id-123":   true,
		"target-id-456": true,
	}

	result := ConvertToExportFeature(feature, featureIDs)

	// Check basic fields
	if result.ID != feature.ID {
		t.Errorf("ID mismatch: got %s, want %s", result.ID, feature.ID)
	}
	if result.Name != feature.Name {
		t.Errorf("Name mismatch: got %s, want %s", result.Name, feature.Name)
	}
	if result.Description != feature.Description {
		t.Errorf("Description mismatch: got %s, want %s", result.Description, feature.Description)
	}
	if result.State != string(feature.DeriveState()) {
		t.Errorf("State mismatch: got %s, want %s", result.State, feature.DeriveState())
	}
	if result.CurrentVersion != feature.GetCurrentVersionKey() {
		t.Errorf("CurrentVersion mismatch: got %s, want %s", result.CurrentVersion, feature.GetCurrentVersionKey())
	}

	// Check tags
	if len(result.Tags) != len(feature.Tags) {
		t.Errorf("Tags count mismatch: got %d, want %d", len(result.Tags), len(feature.Tags))
	}

	// Check versions
	if len(result.Versions) != len(feature.Versions) {
		t.Errorf("Versions count mismatch: got %d, want %d", len(result.Versions), len(feature.Versions))
	}
	if v, ok := result.Versions["1"]; ok {
		if v.ClosedAt == "" {
			t.Error("Version 1 should have ClosedAt set")
		}
		if v.Branch != "feature/test" {
			t.Errorf("Version 1 branch mismatch: got %s, want feature/test", v.Branch)
		}
	} else {
		t.Error("Version 1 not found in export")
	}

	// Check relationships
	if len(result.Relationships) != 1 {
		t.Fatalf("Relationships count mismatch: got %d, want 1", len(result.Relationships))
	}
	rel := result.Relationships[0]
	if rel.Type != "depends-on" {
		t.Errorf("Relationship type mismatch: got %s, want depends-on", rel.Type)
	}
	if !rel.TargetExists {
		t.Error("TargetExists should be true")
	}
	if rel.VersionConstraint == nil {
		t.Error("VersionConstraint should not be nil")
	} else if rel.VersionConstraint.Operator != ">=" {
		t.Errorf("VersionConstraint operator mismatch: got %s, want >=", rel.VersionConstraint.Operator)
	}
}

func TestConvertToExportFeature_MissingTarget(t *testing.T) {
	feature := &fogit.Feature{
		ID:   "test-id",
		Name: "Test",
		Versions: map[string]*fogit.FeatureVersion{
			"1": {CreatedAt: time.Now()},
		},
		Relationships: []fogit.Relationship{
			{
				ID:         "rel-1",
				Type:       "depends-on",
				TargetID:   "missing-target",
				TargetName: "Missing",
			},
		},
	}

	// Only include the source feature, not the target
	featureIDs := map[string]bool{
		"test-id": true,
	}

	result := ConvertToExportFeature(feature, featureIDs)

	if len(result.Relationships) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(result.Relationships))
	}
	if result.Relationships[0].TargetExists {
		t.Error("TargetExists should be false for missing target")
	}
}

func TestExportDataJSONSerialization(t *testing.T) {
	exportData := ExportData{
		FogitVersion: "1.0",
		ExportedAt:   time.Now().UTC().Format(time.RFC3339),
		Repository:   "test-repo",
		Features: []*ExportFeature{
			{
				ID:             "test-1",
				Name:           "Test Feature",
				State:          "open",
				CurrentVersion: "1",
				Versions: map[string]*ExportVersion{
					"1": {
						CreatedAt: time.Now().UTC().Format(time.RFC3339),
					},
				},
			},
		},
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal export data: %v", err)
	}

	// Deserialize back
	var parsed ExportData
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal export data: %v", err)
	}

	if parsed.FogitVersion != exportData.FogitVersion {
		t.Errorf("FogitVersion mismatch: got %s, want %s", parsed.FogitVersion, exportData.FogitVersion)
	}
	if len(parsed.Features) != 1 {
		t.Fatalf("Features count mismatch: got %d, want 1", len(parsed.Features))
	}
	if parsed.Features[0].Name != "Test Feature" {
		t.Errorf("Feature name mismatch: got %s, want Test Feature", parsed.Features[0].Name)
	}
}

func TestValidateImportData(t *testing.T) {
	tests := []struct {
		name    string
		data    *ExportData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid data",
			data: &ExportData{
				FogitVersion: "1.0",
				Features: []*ExportFeature{
					{ID: "test-1", Name: "Feature 1"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			data: &ExportData{
				Features: []*ExportFeature{
					{ID: "test-1", Name: "Feature 1"},
				},
			},
			wantErr: true,
			errMsg:  "missing fogit_version",
		},
		{
			name: "no features",
			data: &ExportData{
				FogitVersion: "1.0",
				Features:     []*ExportFeature{},
			},
			wantErr: true,
			errMsg:  "no features",
		},
		{
			name: "feature missing ID",
			data: &ExportData{
				FogitVersion: "1.0",
				Features: []*ExportFeature{
					{Name: "Feature 1"},
				},
			},
			wantErr: true,
			errMsg:  "missing ID",
		},
		{
			name: "feature missing name",
			data: &ExportData{
				FogitVersion: "1.0",
				Features: []*ExportFeature{
					{ID: "test-1"},
				},
			},
			wantErr: true,
			errMsg:  "missing name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateImportData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateImportData() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("error message should contain %q, got %q", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestValidateRelationshipTargets(t *testing.T) {
	features := []*ExportFeature{
		{
			ID:   "feature-1",
			Name: "Feature 1",
			Relationships: []ExportRelationship{
				{
					TargetID:   "feature-2",
					TargetName: "Feature 2",
				},
			},
		},
		{
			ID:   "feature-2",
			Name: "Feature 2",
		},
	}

	allIDs := map[string]bool{
		"feature-1": true,
		"feature-2": true,
	}

	// Should not return error when all targets exist
	err := ValidateRelationshipTargets(features, allIDs)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestConvertFromExportFeature(t *testing.T) {
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	closedStr := now.Add(-1 * time.Hour).Format(time.RFC3339)

	ef := &ExportFeature{
		ID:             "test-id",
		Name:           "Test Feature",
		Description:    "A test",
		Tags:           []string{"tag1", "tag2"},
		Files:          []string{"file1.go"},
		State:          "in-progress",
		CurrentVersion: "2",
		Metadata: map[string]interface{}{
			"priority": "high",
		},
		Versions: map[string]*ExportVersion{
			"1": {
				CreatedAt:  nowStr,
				ModifiedAt: nowStr,
				ClosedAt:   closedStr,
				Branch:     "feature/v1",
				Authors:    []string{"alice@test.com"},
				Notes:      "Version 1",
			},
			"2": {
				CreatedAt:  nowStr,
				ModifiedAt: nowStr,
				Branch:     "feature/v2",
			},
		},
		Relationships: []ExportRelationship{
			{
				ID:          "rel-1",
				Type:        "depends-on",
				TargetID:    "target-id",
				TargetName:  "Target",
				Description: "Depends",
				CreatedAt:   nowStr,
				VersionConstraint: &ExportVersionConstraint{
					Operator: ">=",
					Version:  2,
				},
			},
		},
	}

	result := ConvertFromExportFeature(ef)

	// Check basic fields
	if result.ID != ef.ID {
		t.Errorf("ID mismatch: got %s, want %s", result.ID, ef.ID)
	}
	if result.Name != ef.Name {
		t.Errorf("Name mismatch: got %s, want %s", result.Name, ef.Name)
	}
	if result.Description != ef.Description {
		t.Errorf("Description mismatch: got %s, want %s", result.Description, ef.Description)
	}

	// Check tags
	if len(result.Tags) != 2 {
		t.Errorf("Tags count mismatch: got %d, want 2", len(result.Tags))
	}

	// Check versions
	if len(result.Versions) != 2 {
		t.Fatalf("Versions count mismatch: got %d, want 2", len(result.Versions))
	}
	v1 := result.Versions["1"]
	if v1 == nil {
		t.Fatal("Version 1 not found")
	}
	if v1.ClosedAt == nil {
		t.Error("Version 1 ClosedAt should not be nil")
	}
	if v1.Branch != "feature/v1" {
		t.Errorf("Version 1 branch mismatch: got %s, want feature/v1", v1.Branch)
	}

	// Check relationships
	if len(result.Relationships) != 1 {
		t.Fatalf("Relationships count mismatch: got %d, want 1", len(result.Relationships))
	}
	rel := result.Relationships[0]
	if rel.ID != "rel-1" {
		t.Errorf("Relationship ID mismatch: got %s, want rel-1", rel.ID)
	}
	if rel.Type != "depends-on" {
		t.Errorf("Relationship type mismatch: got %s, want depends-on", string(rel.Type))
	}
	if rel.VersionConstraint == nil {
		t.Error("VersionConstraint should not be nil")
	} else if rel.VersionConstraint.Operator != ">=" {
		t.Errorf("VersionConstraint operator mismatch: got %s, want >=", rel.VersionConstraint.Operator)
	}
}

func TestConvertFromExportFeature_MinimalData(t *testing.T) {
	ef := &ExportFeature{
		ID:   "minimal-id",
		Name: "Minimal Feature",
	}

	result := ConvertFromExportFeature(ef)

	if result.ID != "minimal-id" {
		t.Errorf("ID mismatch: got %s, want minimal-id", result.ID)
	}
	if result.Name != "Minimal Feature" {
		t.Errorf("Name mismatch: got %s, want Minimal Feature", result.Name)
	}
	if len(result.Versions) > 0 {
		t.Error("Versions should be nil or empty for minimal feature")
	}
	if len(result.Relationships) > 0 {
		t.Error("Relationships should be nil or empty for minimal feature")
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

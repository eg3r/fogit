package features

import (
	"testing"
	"time"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestGetRecentChanges(t *testing.T) {
	tests := []struct {
		name     string
		features []*fogit.Feature
		window   time.Duration
		limit    int
		wantLen  int
	}{
		{
			name:     "empty features",
			features: []*fogit.Feature{},
			window:   7 * 24 * time.Hour,
			limit:    5,
			wantLen:  0,
		},
		{
			name: "recent features within window",
			features: func() []*fogit.Feature {
				f1 := fogit.NewFeature("Recent Feature")
				return []*fogit.Feature{f1}
			}(),
			window:  7 * 24 * time.Hour,
			limit:   5,
			wantLen: 1,
		},
		{
			name: "limit applies",
			features: func() []*fogit.Feature {
				var features []*fogit.Feature
				for i := 0; i < 10; i++ {
					f := fogit.NewFeature("Feature")
					features = append(features, f)
				}
				return features
			}(),
			window:  7 * 24 * time.Hour,
			limit:   3,
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changes := GetRecentChanges(tt.features, tt.window, tt.limit)
			if len(changes) != tt.wantLen {
				t.Errorf("GetRecentChanges() returned %d changes, want %d", len(changes), tt.wantLen)
			}
		})
	}
}

func TestFindFeaturesOnBranch(t *testing.T) {
	tests := []struct {
		name     string
		features []*fogit.Feature
		branch   string
		wantLen  int
	}{
		{
			name:     "empty features",
			features: []*fogit.Feature{},
			branch:   "main",
			wantLen:  0,
		},
		{
			name: "feature on matching branch",
			features: func() []*fogit.Feature {
				f := fogit.NewFeature("Test Feature")
				v := f.GetCurrentVersion()
				v.Branch = "feature/test"
				return []*fogit.Feature{f}
			}(),
			branch:  "feature/test",
			wantLen: 1,
		},
		{
			name: "feature on different branch",
			features: func() []*fogit.Feature {
				f := fogit.NewFeature("Test Feature")
				v := f.GetCurrentVersion()
				v.Branch = "feature/other"
				return []*fogit.Feature{f}
			}(),
			branch:  "main",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := FindFeaturesOnBranch(tt.features, tt.branch)
			if len(names) != tt.wantLen {
				t.Errorf("FindFeaturesOnBranch() returned %d names, want %d", len(names), tt.wantLen)
			}
		})
	}
}

func TestBuildStatusReport(t *testing.T) {
	cfg := fogit.DefaultConfig()

	// Create test features
	f1 := fogit.NewFeature("Open Feature")

	f2 := fogit.NewFeature("InProgress Feature")
	f2.UpdateState(fogit.StateInProgress)

	f3 := fogit.NewFeature("Closed Feature")
	f3.UpdateState(fogit.StateClosed)

	// Add relationship to f1
	f1.Relationships = append(f1.Relationships, fogit.Relationship{
		ID:       "rel-1",
		Type:     "depends-on",
		TargetID: f2.ID,
	})

	featuresList := []*fogit.Feature{f1, f2, f3}

	opts := DefaultStatusOptions()
	report := BuildStatusReport(featuresList, cfg, opts)

	if report.Repository.TotalFeatures != 3 {
		t.Errorf("TotalFeatures = %d, want 3", report.Repository.TotalFeatures)
	}

	if report.FeatureCounts.Open != 1 {
		t.Errorf("Open = %d, want 1", report.FeatureCounts.Open)
	}

	if report.FeatureCounts.InProgress != 1 {
		t.Errorf("InProgress = %d, want 1", report.FeatureCounts.InProgress)
	}

	if report.FeatureCounts.Closed != 1 {
		t.Errorf("Closed = %d, want 1", report.FeatureCounts.Closed)
	}

	if report.RelationshipStats.TotalRelationships != 1 {
		t.Errorf("TotalRelationships = %d, want 1", report.RelationshipStats.TotalRelationships)
	}
}

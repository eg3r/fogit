// Package features provides business logic for feature operations.
package features

import (
	"time"

	"github.com/eg3r/fogit/pkg/fogit"
)

// StatusReport contains the full status information for a repository
type StatusReport struct {
	Repository        RepositoryStatus     `json:"repository"`
	CurrentBranch     string               `json:"current_branch"`
	FeatureCounts     FeatureCountsByState `json:"feature_counts"`
	RecentChanges     []RecentChange       `json:"recent_changes"`
	RelationshipStats RelationshipStatus   `json:"relationship_stats"`
	FeaturesOnBranch  []string             `json:"features_on_branch,omitempty"`
}

// RepositoryStatus contains repository metadata
type RepositoryStatus struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	TotalFeatures int    `json:"total_features"`
	TotalFiles    int    `json:"total_files"`
}

// FeatureCountsByState contains counts grouped by state
type FeatureCountsByState struct {
	Open       int `json:"open"`
	InProgress int `json:"in_progress"`
	Closed     int `json:"closed"`
}

// RecentChange represents a recently modified feature
type RecentChange struct {
	FeatureName string    `json:"feature_name"`
	FeatureID   string    `json:"feature_id"`
	State       string    `json:"state"`
	ModifiedAt  time.Time `json:"modified_at"`
}

// RelationshipStatus contains relationship statistics
type RelationshipStatus struct {
	TotalRelationships int            `json:"total_relationships"`
	ByType             map[string]int `json:"by_type"`
	ByCategory         map[string]int `json:"by_category"`
}

// StatusOptions configures status report generation
type StatusOptions struct {
	RecentChangeWindow time.Duration // How far back to look for recent changes
	RecentChangeLimit  int           // Maximum number of recent changes to include
}

// DefaultStatusOptions returns sensible defaults for status options
func DefaultStatusOptions() StatusOptions {
	return StatusOptions{
		RecentChangeWindow: 7 * 24 * time.Hour, // 7 days
		RecentChangeLimit:  5,
	}
}

// BuildStatusReport generates a status report for the given features
func BuildStatusReport(featuresList []*fogit.Feature, cfg *fogit.Config, opts StatusOptions) *StatusReport {
	if opts.RecentChangeWindow == 0 {
		opts = DefaultStatusOptions()
	}

	report := &StatusReport{
		Repository: RepositoryStatus{
			Name:          cfg.Repository.Name,
			Version:       cfg.Repository.Version,
			TotalFeatures: len(featuresList),
		},
		RelationshipStats: RelationshipStatus{
			ByType:     make(map[string]int),
			ByCategory: make(map[string]int),
		},
	}

	// Count by state
	for _, f := range featuresList {
		state := f.DeriveState()
		switch state {
		case fogit.StateOpen:
			report.FeatureCounts.Open++
		case fogit.StateInProgress:
			report.FeatureCounts.InProgress++
		case fogit.StateClosed:
			report.FeatureCounts.Closed++
		}

		// Count files
		report.Repository.TotalFiles += len(f.Files)

		// Count relationships
		for _, rel := range f.Relationships {
			report.RelationshipStats.TotalRelationships++
			report.RelationshipStats.ByType[string(rel.Type)]++

			// Get category for this relationship type
			category := rel.GetCategory(cfg)
			report.RelationshipStats.ByCategory[category]++
		}
	}

	// Get recent changes
	report.RecentChanges = GetRecentChanges(featuresList, opts.RecentChangeWindow, opts.RecentChangeLimit)

	return report
}

// GetRecentChanges finds features modified within the given time window
func GetRecentChanges(featuresList []*fogit.Feature, window time.Duration, limit int) []RecentChange {
	cutoff := time.Now().Add(-window)
	var changes []RecentChange

	for _, f := range featuresList {
		modifiedAt := f.GetModifiedAt()
		if modifiedAt.After(cutoff) {
			changes = append(changes, RecentChange{
				FeatureName: f.Name,
				FeatureID:   f.ID,
				State:       string(f.DeriveState()),
				ModifiedAt:  modifiedAt,
			})
		}
	}

	// Sort by modified time (most recent first)
	for i := 0; i < len(changes)-1; i++ {
		for j := i + 1; j < len(changes); j++ {
			if changes[j].ModifiedAt.After(changes[i].ModifiedAt) {
				changes[i], changes[j] = changes[j], changes[i]
			}
		}
	}

	// Limit results
	if len(changes) > limit {
		changes = changes[:limit]
	}

	return changes
}

// FindFeaturesOnBranch returns feature names that have a version associated with the given branch
func FindFeaturesOnBranch(featuresList []*fogit.Feature, branch string) []string {
	var names []string
	for _, f := range featuresList {
		// Check if any version is associated with this branch
		for _, v := range f.Versions {
			if v.Branch == branch {
				names = append(names, f.Name)
				break
			}
		}
	}
	return names
}

package features

import (
	"context"
	"fmt"

	"github.com/eg3r/fogit/pkg/fogit"
)

// ImpactedFeature represents a feature affected by changes
type ImpactedFeature struct {
	Name         string   `json:"name" yaml:"name"`
	ID           string   `json:"id" yaml:"id"`
	Relationship string   `json:"relationship" yaml:"relationship"`
	Depth        int      `json:"depth" yaml:"depth"`
	Path         []string `json:"path" yaml:"path"`
	Warning      string   `json:"warning,omitempty" yaml:"warning,omitempty"`
}

// ImpactResult contains the impact analysis results
type ImpactResult struct {
	Feature            string            `json:"feature" yaml:"feature"`
	ImpactedFeatures   []ImpactedFeature `json:"impacted_features" yaml:"impacted_features"`
	TotalAffected      int               `json:"total_affected" yaml:"total_affected"`
	CategoriesIncluded []string          `json:"categories_included" yaml:"categories_included"`
}

// ImpactOptions configures the impact analysis
type ImpactOptions struct {
	MaxDepth          int      // Maximum traversal depth (0 = unlimited)
	IncludeCategories []string // Categories to include (empty = use config defaults)
	ExcludeCategories []string // Categories to exclude
	AllCategories     bool     // Include all categories regardless of config
}

// GetIncludedCategories determines which relationship categories to include in impact analysis
// based on the provided options and configuration
func GetIncludedCategories(cfg *fogit.Config, opts ImpactOptions) []string {
	if opts.AllCategories {
		var all []string
		for name := range cfg.Relationships.Categories {
			all = append(all, name)
		}
		return all
	}

	// Start with categories that have include_in_impact: true
	var included []string
	for name, cat := range cfg.Relationships.Categories {
		if cat.IncludeInImpact {
			included = append(included, name)
		}
	}

	// Add explicitly included
	for _, cat := range opts.IncludeCategories {
		found := false
		for _, inc := range included {
			if inc == cat {
				found = true
				break
			}
		}
		if !found {
			included = append(included, cat)
		}
	}

	// Remove explicitly excluded
	var filtered []string
	for _, inc := range included {
		excluded := false
		for _, exc := range opts.ExcludeCategories {
			if inc == exc {
				excluded = true
				break
			}
		}
		if !excluded {
			filtered = append(filtered, inc)
		}
	}

	return filtered
}

// AnalyzeImpacts performs a BFS traversal to find all features impacted by changes to the given feature.
// It follows reverse relationships (features that depend on the target) through the specified categories.
func AnalyzeImpacts(ctx context.Context, feature *fogit.Feature, repo fogit.Repository, cfg *fogit.Config, categories []string, maxDepth int) (*ImpactResult, error) {
	result := &ImpactResult{
		Feature:            feature.Name,
		CategoriesIncluded: categories,
	}

	// Get all features for lookup
	allFeatures, err := repo.List(ctx, &fogit.Filter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	featureMap := make(map[string]*fogit.Feature)
	for _, f := range allFeatures {
		featureMap[f.ID] = f
	}

	// Build reverse relationship map (who depends on what)
	// Key: target ID, Value: list of features that have relationships TO this target
	reverseMap := make(map[string][]struct {
		feature           *fogit.Feature
		relType           string
		versionConstraint *fogit.VersionConstraint
	})

	for _, f := range allFeatures {
		for _, rel := range f.Relationships {
			// Check if this relationship is in an included category
			category := rel.GetCategory(cfg)
			if !containsString(categories, category) {
				continue
			}

			reverseMap[rel.TargetID] = append(reverseMap[rel.TargetID], struct {
				feature           *fogit.Feature
				relType           string
				versionConstraint *fogit.VersionConstraint
			}{f, string(rel.Type), rel.VersionConstraint})
		}
	}

	// BFS to find all impacted features
	visited := make(map[string]bool)
	visited[feature.ID] = true

	type queueItem struct {
		feature *fogit.Feature
		depth   int
		path    []string
		relType string
	}

	queue := []queueItem{{feature, 0, []string{feature.Name}, ""}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Check depth limit
		if maxDepth > 0 && current.depth >= maxDepth {
			continue
		}

		// Find features that depend on this one
		for _, dep := range reverseMap[current.feature.ID] {
			if visited[dep.feature.ID] {
				continue
			}
			visited[dep.feature.ID] = true

			newPath := make([]string, len(current.path))
			copy(newPath, current.path)
			newPath = append(newPath, dep.feature.Name)

			// Check if version constraint is satisfied
			var warning string
			if dep.versionConstraint != nil {
				targetVersion := current.feature.GetCurrentVersionKey()
				if !dep.versionConstraint.IsSatisfiedBy(targetVersion) {
					warning = fmt.Sprintf("version constraint %s%s not satisfied (current: %s)",
						dep.versionConstraint.Operator,
						dep.versionConstraint.GetVersionString(),
						targetVersion)
				}
			}

			result.ImpactedFeatures = append(result.ImpactedFeatures, ImpactedFeature{
				Name:         dep.feature.Name,
				ID:           dep.feature.ID,
				Relationship: dep.relType,
				Depth:        current.depth + 1,
				Path:         newPath,
				Warning:      warning,
			})

			queue = append(queue, queueItem{
				feature: dep.feature,
				depth:   current.depth + 1,
				path:    newPath,
				relType: dep.relType,
			})
		}
	}

	result.TotalAffected = len(result.ImpactedFeatures)
	return result, nil
}

// containsString checks if a string is in a slice
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

package features

import (
	"context"
	"fmt"

	"github.com/eg3r/fogit/pkg/fogit"
)

// RecursiveRelationship represents a relationship found during recursive traversal
type RecursiveRelationship struct {
	SourceID   string   `json:"source_id" yaml:"source_id"`
	SourceName string   `json:"source_name" yaml:"source_name"`
	TargetID   string   `json:"target_id" yaml:"target_id"`
	TargetName string   `json:"target_name" yaml:"target_name"`
	Type       string   `json:"type" yaml:"type"`
	Depth      int      `json:"depth" yaml:"depth"`
	Path       []string `json:"path" yaml:"path"`
}

// TraversalResult contains the results of a recursive relationship traversal
type TraversalResult struct {
	Feature       *fogit.Feature          `json:"feature" yaml:"feature"`
	Relationships []RecursiveRelationship `json:"relationships" yaml:"relationships"`
	MaxDepth      int                     `json:"max_depth" yaml:"max_depth"`
	Direction     string                  `json:"direction" yaml:"direction"`
	Types         []string                `json:"types,omitempty" yaml:"types,omitempty"`
	Total         int                     `json:"total" yaml:"total"`
}

// TraversalOptions configures the recursive relationship traversal
type TraversalOptions struct {
	Direction string   // "incoming", "outgoing", or "both"
	Types     []string // Relationship types to include (empty = all)
	MaxDepth  int      // Maximum traversal depth (0 = unlimited)
}

// TraverseRelationshipsRecursive performs a BFS traversal of relationships from the given feature.
// It can traverse outgoing relationships, incoming relationships, or both, filtered by type.
func TraverseRelationshipsRecursive(ctx context.Context, repo fogit.Repository, feature *fogit.Feature, opts TraversalOptions) (*TraversalResult, error) {
	// Load all features for lookup
	allFeatures, err := repo.List(ctx, &fogit.Filter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	featureMap := make(map[string]*fogit.Feature)
	for _, f := range allFeatures {
		featureMap[f.ID] = f
	}

	var results []RecursiveRelationship
	visited := make(map[string]bool)
	visited[feature.ID] = true

	type queueItem struct {
		feature *fogit.Feature
		depth   int
		path    []string
	}

	queue := []queueItem{{feature, 0, []string{feature.Name}}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Check depth limit
		if opts.MaxDepth > 0 && current.depth >= opts.MaxDepth {
			continue
		}

		// Process outgoing relationships
		if opts.Direction == "outgoing" || opts.Direction == "both" {
			rels := filterRelationshipsByTypes(current.feature.GetRelationships(""), opts.Types)
			for _, rel := range rels {
				targetName := rel.TargetName
				if targetName == "" {
					if target, ok := featureMap[rel.TargetID]; ok {
						targetName = target.Name
					} else {
						targetName = rel.TargetID
					}
				}

				newPath := make([]string, len(current.path))
				copy(newPath, current.path)
				newPath = append(newPath, targetName)

				results = append(results, RecursiveRelationship{
					SourceID:   current.feature.ID,
					SourceName: current.feature.Name,
					TargetID:   rel.TargetID,
					TargetName: targetName,
					Type:       string(rel.Type),
					Depth:      current.depth + 1,
					Path:       newPath,
				})

				// Add target to queue if not visited
				if target, ok := featureMap[rel.TargetID]; ok && !visited[rel.TargetID] {
					visited[rel.TargetID] = true
					queue = append(queue, queueItem{
						feature: target,
						depth:   current.depth + 1,
						path:    newPath,
					})
				}
			}
		}

		// Process incoming relationships
		if opts.Direction == "incoming" || opts.Direction == "both" {
			for _, f := range allFeatures {
				if visited[f.ID] && f.ID != feature.ID {
					continue // Skip already visited (except initial feature for incoming)
				}
				for _, rel := range f.Relationships {
					if rel.TargetID != current.feature.ID {
						continue
					}
					if !typeMatchesFilter(string(rel.Type), opts.Types) {
						continue
					}

					newPath := make([]string, len(current.path))
					copy(newPath, current.path)
					newPath = append(newPath, f.Name)

					results = append(results, RecursiveRelationship{
						SourceID:   f.ID,
						SourceName: f.Name,
						TargetID:   current.feature.ID,
						TargetName: current.feature.Name,
						Type:       string(rel.Type),
						Depth:      current.depth + 1,
						Path:       newPath,
					})

					// Add source to queue if not visited
					if !visited[f.ID] {
						visited[f.ID] = true
						queue = append(queue, queueItem{
							feature: f,
							depth:   current.depth + 1,
							path:    newPath,
						})
					}
				}
			}
		}
	}

	return &TraversalResult{
		Feature:       feature,
		Relationships: results,
		MaxDepth:      opts.MaxDepth,
		Direction:     opts.Direction,
		Types:         opts.Types,
		Total:         len(results),
	}, nil
}

// filterRelationshipsByTypes filters relationships by type list (empty = all)
func filterRelationshipsByTypes(rels []fogit.Relationship, types []string) []fogit.Relationship {
	if len(types) == 0 {
		return rels
	}
	var filtered []fogit.Relationship
	for _, rel := range rels {
		for _, t := range types {
			if string(rel.Type) == t {
				filtered = append(filtered, rel)
				break
			}
		}
	}
	return filtered
}

// typeMatchesFilter checks if a type is in the filter list (empty = all types match)
func typeMatchesFilter(relType string, typeFilter []string) bool {
	if len(typeFilter) == 0 {
		return true
	}
	for _, t := range typeFilter {
		if t == relType {
			return true
		}
	}
	return false
}

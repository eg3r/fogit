package fogit

import (
	"context"
	"fmt"
	"log"
)

// DetectCycleWithConfig checks if adding a relationship would create a cycle
// based on category-specific cycle detection settings
func DetectCycleWithConfig(ctx context.Context, source *Feature, rel *Relationship, repo Repository, config *Config) error {
	// Self-reference check (always disallowed)
	if source.ID == rel.TargetID {
		return fmt.Errorf("cannot create relationship to self")
	}

	// Get relationship category
	category := rel.GetCategory(config)

	// Get category config
	catConfig, exists := config.Relationships.Categories[category]
	if !exists {
		return fmt.Errorf("unknown category: %s", category)
	}

	// Skip if cycles allowed
	if catConfig.AllowCycles {
		return nil
	}

	// Check for cycles
	// When adding "source -> target", we check if "target -> source" exists (directly or transitively)
	// This prevents creating a cycle
	err := checkTransitivePath(ctx, rel.TargetID, source.ID, category, repo, config)
	if err != nil {
		// Handle based on detection mode
		switch catConfig.CycleDetection {
		case CycleDetectionStrict:
			return err
		case CycleDetectionWarn:
			// Log warning but don't fail
			log.Printf("WARNING: Cycle detected in %s relationship: %v", category, err)
			return nil
		case CycleDetectionNone:
			return nil
		default:
			return err
		}
	}

	return nil
}

// checkTransitivePath checks if target is reachable from start via relationships in the same category
func checkTransitivePath(ctx context.Context, start, target, category string, repo Repository, config *Config) error {
	visited := make(map[string]bool)
	return dfsSearchByCategory(ctx, start, target, category, repo, config, visited)
}

// dfsSearchByCategory performs DFS searching only relationships in the specified category
func dfsSearchByCategory(ctx context.Context, start, target, category string, repo Repository, config *Config, visited map[string]bool) error {
	if visited[start] {
		return nil // Already checked this path
	}
	visited[start] = true

	// Get the feature
	feature, err := repo.Get(ctx, start)
	if err != nil {
		return nil // Feature not found, can't continue search
	}

	// Check all relationships in the same category
	for _, rel := range feature.Relationships {
		relCategory := rel.GetCategory(config)
		if relCategory == category {
			// Found the target - cycle detected
			if rel.TargetID == target {
				return fmt.Errorf("cycle detected: adding this relationship would create a circular dependency in %s relationships", category)
			}

			// Recursively search from the related feature
			if err := dfsSearchByCategory(ctx, rel.TargetID, target, category, repo, config, visited); err != nil {
				return err
			}
		}
	}

	return nil
}

package features

import (
	"context"
	"fmt"
	"strings"

	"github.com/eg3r/fogit/internal/search"
	"github.com/eg3r/fogit/pkg/fogit"
)

// FindResult contains the result of a feature search
type FindResult struct {
	Feature     *fogit.Feature
	Suggestions []search.Match
}

// Find looks up a feature by ID or name, with optional fuzzy matching
func Find(ctx context.Context, repo fogit.Repository, identifier string, cfg *fogit.Config) (*FindResult, error) {
	// First, try to get by ID
	feature, err := repo.Get(ctx, identifier)
	if err == nil {
		return &FindResult{Feature: feature}, nil
	}

	// If not found by ID, search by name
	filter := &fogit.Filter{}
	features, err := repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search features: %w", err)
	}

	// Find by name (case-insensitive)
	var matches []*fogit.Feature
	lowerIdentifier := strings.ToLower(identifier)
	for _, f := range features {
		if strings.ToLower(f.Name) == lowerIdentifier {
			matches = append(matches, f)
		}
	}

	if len(matches) == 0 {
		// No exact match - try fuzzy search
		if cfg.FeatureSearch.FuzzyMatch {
			searchCfg := search.SearchConfig{
				FuzzyMatch:     cfg.FeatureSearch.FuzzyMatch,
				MinSimilarity:  cfg.FeatureSearch.MinSimilarity,
				MaxSuggestions: cfg.FeatureSearch.MaxSuggestions,
			}

			similarMatches := search.FindSimilar(identifier, features, searchCfg)
			if len(similarMatches) > 0 {
				return &FindResult{Suggestions: similarMatches}, fogit.ErrNotFound
			}
		}
		return nil, fogit.ErrNotFound
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple features found with name %q, use ID instead", identifier)
	}

	return &FindResult{Feature: matches[0]}, nil
}

// FindForBranch finds the feature associated with the given branch
// First tries to match branch name, then returns most recently updated open feature
func FindForBranch(ctx context.Context, repo fogit.Repository, branch string) (*fogit.Feature, error) {
	features, err := FindAllForBranch(ctx, repo, branch)
	if err != nil {
		return nil, err
	}

	if len(features) == 0 {
		return nil, nil
	}

	// Return the first one (for backward compatibility)
	return features[0], nil
}

// FindAllForBranch finds ALL features associated with the given branch
// Per spec: features on shared branches share the branch lifecycle
// Returns features with matching branch metadata, or most recently modified if none match
func FindAllForBranch(ctx context.Context, repo fogit.Repository, branch string) ([]*fogit.Feature, error) {
	// List all open features
	filter := &fogit.Filter{
		State: fogit.StateOpen,
	}

	features, err := repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	if len(features) == 0 {
		return nil, nil
	}

	// Find ALL features with matching branch in metadata
	var branchFeatures []*fogit.Feature
	for _, feature := range features {
		if branchMeta, ok := feature.Metadata["branch"].(string); ok {
			if branchMeta == branch {
				branchFeatures = append(branchFeatures, feature)
			}
		}
	}

	// If features found on this branch, return all of them
	if len(branchFeatures) > 0 {
		return branchFeatures, nil
	}

	// If no match, return most recently modified open feature (single feature)
	// This assumes the user is working on their most recent feature
	// When timestamps are equal, use ID as tiebreaker for deterministic behavior
	var mostRecent *fogit.Feature
	for _, feature := range features {
		if mostRecent == nil {
			mostRecent = feature
		} else if feature.GetModifiedAt().After(mostRecent.GetModifiedAt()) {
			mostRecent = feature
		} else if feature.GetModifiedAt().Equal(mostRecent.GetModifiedAt()) && feature.ID > mostRecent.ID {
			// Tiebreaker: when timestamps are equal, use lexicographically greater ID
			// This ensures deterministic behavior regardless of iteration order
			mostRecent = feature
		}
	}

	if mostRecent != nil {
		return []*fogit.Feature{mostRecent}, nil
	}

	return nil, nil
}

// FindForFile finds all features that reference the given file path
// Supports partial matching (suffix or substring)
func FindForFile(ctx context.Context, repo fogit.Repository, filePath string) ([]*fogit.Feature, error) {
	// List all features
	// Note: We might want to accept a filter here if we want to filter by state
	filter := &fogit.Filter{}
	features, err := repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	// Normalize file path for comparison
	// We need to handle path separators correctly
	normalizedPath := strings.ReplaceAll(filePath, "\\", "/")

	var matchingFeatures []*fogit.Feature
	for _, f := range features {
		for _, file := range f.Files {
			normalizedFile := strings.ReplaceAll(file, "\\", "/")
			if strings.EqualFold(normalizedFile, normalizedPath) ||
				strings.HasSuffix(normalizedFile, normalizedPath) ||
				strings.Contains(normalizedFile, normalizedPath) {
				matchingFeatures = append(matchingFeatures, f)
				break
			}
		}
	}

	return matchingFeatures, nil
}

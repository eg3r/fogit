package features

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/search"
	"github.com/eg3r/fogit/internal/storage"
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

// ============================================================================
// Cross-Branch Feature Discovery
// Per spec/specification/07-git-integration.md#cross-branch-feature-discovery
// ============================================================================

// CrossBranchFindResult contains the result of a cross-branch feature search
type CrossBranchFindResult struct {
	Feature     *fogit.Feature
	Branch      string         // Branch where feature was found
	IsRemote    bool           // True if only found on remote
	Suggestions []search.Match // Fuzzy match suggestions if not found
}

// FindAcrossBranches searches for a feature across all branches.
// Per spec, discovery order:
// 1. Current branch - Check local .fogit/features/ directory first
// 2. Main/trunk branch - Check configured trunk branch (main, master)
// 3. Other local branches - Use git ls-tree to scan
// 4. Remote branches - Scan origin/* and other remotes
//
// Returns the feature and the branch it was found on.
func FindAcrossBranches(ctx context.Context, repo fogit.Repository, gitRepo *git.Repository, identifier string, cfg *fogit.Config) (*CrossBranchFindResult, error) {
	// Get current branch once for reuse
	currentBranch, _ := gitRepo.GetCurrentBranch()

	// Step 1: Try local find first (current branch)
	findResult, err := Find(ctx, repo, identifier, cfg)
	if err == nil && findResult.Feature != nil {
		// Found on current branch
		return &CrossBranchFindResult{
			Feature:  findResult.Feature,
			Branch:   currentBranch,
			IsRemote: false,
		}, nil
	}

	// Step 2: Check trunk branch (main/master)
	trunkBranch, err := gitRepo.GetTrunkBranch()
	if err == nil {
		if trunkBranch != currentBranch {
			feature, branch, err := findFeatureOnBranch(gitRepo, trunkBranch, identifier)
			if err == nil && feature != nil {
				return &CrossBranchFindResult{
					Feature:  feature,
					Branch:   branch,
					IsRemote: false,
				}, nil
			}
		}
	}

	// Step 3: Check other local branches
	branches, err := gitRepo.ListBranches()
	if err == nil {
		for _, branch := range branches {
			// Skip current branch and trunk (already checked)
			if branch == currentBranch || branch == trunkBranch {
				continue
			}

			feature, foundBranch, err := findFeatureOnBranch(gitRepo, branch, identifier)
			if err == nil && feature != nil {
				return &CrossBranchFindResult{
					Feature:  feature,
					Branch:   foundBranch,
					IsRemote: false,
				}, nil
			}
		}
	}

	// Step 4: Check remote branches
	// Remote branches may have content that local branches don't have yet
	// (e.g., after fetch but before pull/merge), so we always check them
	remoteBranches, err := gitRepo.ListRemoteBranches()
	if err == nil {
		for _, remoteBranch := range remoteBranches {
			feature, foundBranch, err := findFeatureOnBranch(gitRepo, remoteBranch, identifier)
			if err == nil && feature != nil {
				return &CrossBranchFindResult{
					Feature:  feature,
					Branch:   foundBranch,
					IsRemote: true,
				}, nil
			}
		}
	}

	// Not found anywhere - try to provide fuzzy suggestions from current branch
	if cfg.FeatureSearch.FuzzyMatch {
		features, err := repo.List(ctx, &fogit.Filter{})
		if err == nil && len(features) > 0 {
			searchCfg := search.SearchConfig{
				FuzzyMatch:     cfg.FeatureSearch.FuzzyMatch,
				MinSimilarity:  cfg.FeatureSearch.MinSimilarity,
				MaxSuggestions: cfg.FeatureSearch.MaxSuggestions,
			}
			suggestions := search.FindSimilar(identifier, features, searchCfg)
			if len(suggestions) > 0 {
				return &CrossBranchFindResult{
					Suggestions: suggestions,
				}, fogit.ErrNotFound
			}
		}
	}

	return nil, fogit.ErrNotFound
}

// findFeatureOnBranch searches for a feature on a specific branch without checkout.
// Returns the feature, branch name, and any error.
func findFeatureOnBranch(gitRepo *git.Repository, branch string, identifier string) (*fogit.Feature, string, error) {
	featuresPath := ".fogit/features"

	// List feature files on the branch
	files, err := gitRepo.ListFilesOnBranch(branch, featuresPath)
	if err != nil {
		return nil, "", err
	}

	// Check each feature file
	for _, filePath := range files {
		// Only process YAML files
		if !isYAMLFile(filePath) {
			continue
		}

		// Read the feature file from the branch
		data, err := gitRepo.ReadFileOnBranch(branch, filePath)
		if err != nil {
			continue
		}

		// Parse the feature
		feature, err := storage.UnmarshalFeature(data)
		if err != nil {
			continue
		}

		// Check if this is the feature we're looking for
		if matchesIdentifier(feature, identifier) {
			return feature, branch, nil
		}
	}

	return nil, "", fogit.ErrNotFound
}

// matchesIdentifier checks if a feature matches the given identifier (ID or name)
func matchesIdentifier(feature *fogit.Feature, identifier string) bool {
	// Check ID match
	if feature.ID == identifier {
		return true
	}

	// Check name match (case-insensitive)
	if strings.EqualFold(feature.Name, identifier) {
		return true
	}

	return false
}

// isYAMLFile checks if a file has a YAML extension
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}

// extractLocalBranchName extracts the local branch name from a remote branch reference
// e.g., "origin/feature/auth" -> "feature/auth"
func extractLocalBranchName(remoteBranch string) string {
	parts := strings.SplitN(remoteBranch, "/", 2)
	if len(parts) >= 2 {
		return parts[1]
	}
	return remoteBranch
}

// ListFeaturesAcrossBranches returns all features from all branches.
// This is useful for fogit list and fogit search commands.
// When a feature exists on multiple branches (e.g., stale snapshots from branch creation),
// the version with the most recent modified_at timestamp wins to ensure consistent state.
func ListFeaturesAcrossBranches(ctx context.Context, repo fogit.Repository, gitRepo *git.Repository) ([]*CrossBranchFeature, error) {
	// Use map to track best version of each feature (most recent modified_at wins)
	featureMap := make(map[string]*CrossBranchFeature)

	// Helper to add or update feature in map
	addOrUpdateFeature := func(f *fogit.Feature, branch string, isRemote bool) {
		cbf := &CrossBranchFeature{
			Feature:  f,
			Branch:   branch,
			IsRemote: isRemote,
		}
		existing, seen := featureMap[f.ID]
		if !seen {
			featureMap[f.ID] = cbf
		} else if f.GetModifiedAt().After(existing.Feature.GetModifiedAt()) {
			// Prefer version with most recent modified_at (authoritative branch has latest changes)
			featureMap[f.ID] = cbf
		}
	}

	// Step 1: Get features from current branch
	currentBranch, _ := gitRepo.GetCurrentBranch()
	localFeatures, err := repo.List(ctx, &fogit.Filter{})
	if err == nil {
		for _, f := range localFeatures {
			addOrUpdateFeature(f, currentBranch, false)
		}
	}

	// Step 2: Get features from trunk branch
	trunkBranch, err := gitRepo.GetTrunkBranch()
	if err == nil && trunkBranch != currentBranch {
		trunkFeatures, err := listFeaturesOnBranch(gitRepo, trunkBranch)
		if err == nil {
			for _, f := range trunkFeatures {
				addOrUpdateFeature(f, trunkBranch, false)
			}
		}
	}

	// Step 3: Get features from other local branches
	branches, err := gitRepo.ListBranches()
	if err == nil {
		for _, branch := range branches {
			if branch == currentBranch || branch == trunkBranch {
				continue
			}

			branchFeatures, err := listFeaturesOnBranch(gitRepo, branch)
			if err == nil {
				for _, f := range branchFeatures {
					addOrUpdateFeature(f, branch, false)
				}
			}
		}
	}

	// Step 4: Get features from remote branches
	// Remote branches may have content that local branches don't have yet
	// (e.g., after fetch but before pull/merge), so we always check them
	remoteBranches, err := gitRepo.ListRemoteBranches()
	if err == nil {
		for _, remoteBranch := range remoteBranches {
			remoteFeatures, err := listFeaturesOnBranch(gitRepo, remoteBranch)
			if err == nil {
				for _, f := range remoteFeatures {
					addOrUpdateFeature(f, remoteBranch, true)
				}
			}
		}
	}

	// Convert map to slice
	allFeatures := make([]*CrossBranchFeature, 0, len(featureMap))
	for _, cbf := range featureMap {
		allFeatures = append(allFeatures, cbf)
	}

	return allFeatures, nil
}

// CrossBranchFeature wraps a feature with its branch information
type CrossBranchFeature struct {
	Feature  *fogit.Feature
	Branch   string
	IsRemote bool
}

// listFeaturesOnBranch lists all features on a specific branch
func listFeaturesOnBranch(gitRepo *git.Repository, branch string) ([]*fogit.Feature, error) {
	featuresPath := ".fogit/features"
	var features []*fogit.Feature

	// List feature files on the branch
	files, err := gitRepo.ListFilesOnBranch(branch, featuresPath)
	if err != nil {
		return nil, err
	}

	// Read and parse each feature file
	for _, filePath := range files {
		if !isYAMLFile(filePath) {
			continue
		}

		data, err := gitRepo.ReadFileOnBranch(branch, filePath)
		if err != nil {
			continue
		}

		feature, err := storage.UnmarshalFeature(data)
		if err != nil {
			continue
		}

		features = append(features, feature)
	}

	return features, nil
}

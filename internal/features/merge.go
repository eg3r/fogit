package features

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/eg3r/fogit/internal/common"
	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/pkg/fogit"
)

// ErrMergeInProgress indicates a merge is in progress and needs --continue or --abort
var ErrMergeInProgress = errors.New("merge in progress")

// ErrNoMergeInProgress indicates --continue/--abort used without an active merge
var ErrNoMergeInProgress = errors.New("no merge in progress")

// ErrConflictsRemaining indicates there are still unresolved conflicts
var ErrConflictsRemaining = errors.New("conflicts remaining")

// MergeOptions contains options for the Merge operation
type MergeOptions struct {
	FeatureName string // Specific feature to close (optional, empty = all on branch)
	NoDelete    bool   // Keep branch after merge
	Squash      bool   // Squash commits
	BaseBranch  string // Target branch to merge into (default: main)
	Continue    bool   // Continue after conflict resolution
	Abort       bool   // Abort the current merge
	FogitDir    string // Path to .fogit directory (required for state management)
}

// MergeResult contains the result of a Merge operation
type MergeResult struct {
	ClosedFeatures   []*fogit.Feature
	Branch           string // Original feature branch
	BaseBranch       string // Target branch merged into
	IsMainBranch     bool   // Was already on main (trunk-based)
	NoDelete         bool   // Keep branch flag
	BranchDeleted    bool   // Whether branch was deleted
	MergePerformed   bool   // Whether Git merge was performed
	ConflictDetected bool   // Whether merge had conflicts (needs resolution)
	Aborted          bool   // Whether merge was aborted
}

// Merge closes features and merges branch (in branch-per-feature mode)
// Per spec:
//   - Branch-per-feature mode: checkout base, merge feature branch, close feature, delete branch
//   - Trunk-based mode: just close the feature (no Git merge needed)
//
// Conflict workflow:
//  1. Merge detects conflict → saves state, returns ConflictDetected=true
//  2. User resolves conflicts manually
//  3. User runs `fogit merge --continue` to complete
//  4. Or `fogit merge --abort` to cancel
func Merge(ctx context.Context, repo fogit.Repository, gitRepo *git.Repository, opts MergeOptions) (*MergeResult, error) {
	// Handle --abort
	if opts.Abort {
		return handleMergeAbort(ctx, repo, gitRepo, opts)
	}

	// Handle --continue
	if opts.Continue {
		return handleMergeContinue(ctx, repo, gitRepo, opts)
	}

	// Check if there's already a merge in progress
	if HasMergeState(opts.FogitDir) {
		return nil, fmt.Errorf("%w: use 'fogit merge --continue' to finish or 'fogit merge --abort' to cancel", ErrMergeInProgress)
	}

	result := &MergeResult{
		NoDelete: opts.NoDelete,
	}

	// Set default base branch
	if opts.BaseBranch == "" {
		opts.BaseBranch = "main"
	}
	result.BaseBranch = opts.BaseBranch

	// Get current branch
	branch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	result.Branch = branch
	result.IsMainBranch = isMainBranch(branch)

	// Validate base branch exists BEFORE making any changes (atomic operation)
	// This prevents closing features when merge target doesn't exist
	if !result.IsMainBranch && !gitRepo.BranchExists(opts.BaseBranch) {
		return nil, fmt.Errorf("target branch '%s' does not exist. Configure with:\n  fogit config set workflow.base_branch <branch_name>\nor create it with:\n  git branch %s", opts.BaseBranch, opts.BaseBranch)
	}

	// Find features to close
	featuresToClose, err := findFeaturesToClose(ctx, repo, opts.FeatureName, branch)
	if err != nil {
		return nil, err
	}

	if len(featuresToClose) == 0 {
		return nil, fmt.Errorf("no open features found to close")
	}

	// Check for uncommitted changes
	changedFiles, err := gitRepo.GetChangedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to check for changes: %w", err)
	}

	if len(changedFiles) > 0 {
		return nil, fmt.Errorf("you have uncommitted changes. Commit or stash them first:\n  %v", changedFiles)
	}

	// Close each feature by setting ClosedAt on current version
	now := time.Now().UTC()
	for _, feature := range featuresToClose {
		if err := closeFeature(ctx, repo, feature, now); err != nil {
			return nil, err
		}
	}
	result.ClosedFeatures = featuresToClose

	// If already on main branch (trunk-based mode), we're done
	if result.IsMainBranch {
		return result, nil
	}

	// Branch-per-feature mode: perform actual Git merge
	featureBranch := branch

	// Commit the feature file changes (ClosedAt update) before checkout
	_, commitErr := gitRepo.Commit("Close feature(s) for merge", nil)
	if commitErr != nil && commitErr != git.ErrNothingToCommit {
		return nil, fmt.Errorf("failed to commit feature closure: %w", commitErr)
	}

	// Checkout base branch
	if err := gitRepo.CheckoutBranch(opts.BaseBranch); err != nil {
		return nil, fmt.Errorf("failed to checkout %s: %w", opts.BaseBranch, err)
	}

	// Perform merge
	var mergeErr error
	if opts.Squash {
		mergeErr = gitRepo.MergeBranchSquash(featureBranch)
	} else {
		mergeErr = gitRepo.MergeBranch(featureBranch)
	}

	if mergeErr != nil {
		// Check if it's a conflict
		if errors.Is(mergeErr, git.ErrMergeConflict) {
			// Save merge state for --continue
			featureIDs := make([]string, len(featuresToClose))
			for i, f := range featuresToClose {
				featureIDs[i] = f.ID
			}

			state := &MergeState{
				FeatureBranch: featureBranch,
				BaseBranch:    opts.BaseBranch,
				FeatureIDs:    featureIDs,
				NoDelete:      opts.NoDelete,
				Squash:        opts.Squash,
			}

			if err := SaveMergeState(opts.FogitDir, state); err != nil {
				// Non-fatal, but log it
				fmt.Printf("Warning: could not save merge state: %v\n", err)
			}

			result.ConflictDetected = true
			result.BaseBranch = opts.BaseBranch
			result.Branch = featureBranch
			return result, nil
		}

		// Other merge error - abort and return to feature branch
		_ = gitRepo.AbortMerge()
		_ = gitRepo.CheckoutBranch(featureBranch)
		return nil, fmt.Errorf("merge failed: %w", mergeErr)
	}
	result.MergePerformed = true

	// Delete feature branch if requested
	if !opts.NoDelete {
		if err := gitRepo.DeleteBranch(featureBranch); err != nil {
			// Non-fatal: branch deletion failure doesn't fail the merge
			// Just don't set BranchDeleted
		} else {
			result.BranchDeleted = true
		}
	}

	return result, nil
}

// findFeaturesToClose finds features that should be closed
func findFeaturesToClose(ctx context.Context, repo fogit.Repository, featureName, branch string) ([]*fogit.Feature, error) {
	var featuresToClose []*fogit.Feature

	if featureName != "" {
		// Close specific feature - use Find to support both ID and name lookup
		// Use a default config for searching (no fuzzy matching for explicit lookups)
		cfg := &fogit.Config{
			FeatureSearch: fogit.FeatureSearchConfig{
				FuzzyMatch: false,
			},
		}
		result, err := Find(ctx, repo, featureName, cfg)
		if err != nil {
			return nil, fmt.Errorf("feature not found: %w", err)
		}
		featuresToClose = append(featuresToClose, result.Feature)
	} else {
		// Find all non-closed features (both open and in-progress) on current branch
		// First try open features
		openFilter := &fogit.Filter{
			State: fogit.StateOpen,
		}
		openFeatures, err := repo.List(ctx, openFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to list features: %w", err)
		}

		// Then try in-progress features
		inProgressFilter := &fogit.Filter{
			State: fogit.StateInProgress,
		}
		inProgressFeatures, err := repo.List(ctx, inProgressFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to list features: %w", err)
		}

		// Combine all non-closed features
		allFeatures := append(openFeatures, inProgressFeatures...)

		// Filter features on current branch or use most recent
		for _, feature := range allFeatures {
			if branchMeta, ok := feature.Metadata["branch"].(string); ok {
				if branchMeta == branch {
					featuresToClose = append(featuresToClose, feature)
				}
			}
		}

		// If no features found with branch metadata, use most recently modified
		if len(featuresToClose) == 0 && len(allFeatures) > 0 {
			mostRecent := findMostRecentFeature(allFeatures)
			if mostRecent != nil {
				featuresToClose = append(featuresToClose, mostRecent)
			}
		}
	}

	return featuresToClose, nil
}

// closeFeature sets the closed state on a feature's current version
func closeFeature(ctx context.Context, repo fogit.Repository, feature *fogit.Feature, closeTime time.Time) error {
	// Set closed state on current version (spec-compliant)
	if currentVersion := feature.GetCurrentVersion(); currentVersion != nil {
		currentVersion.ClosedAt = &closeTime
		currentVersion.ModifiedAt = closeTime
	}

	// Save feature
	if err := repo.Update(ctx, feature); err != nil {
		return fmt.Errorf("failed to update feature %s: %w", feature.Name, err)
	}

	return nil
}

// findMostRecentFeature finds the most recently modified feature
func findMostRecentFeature(features []*fogit.Feature) *fogit.Feature {
	var mostRecent *fogit.Feature
	for _, feature := range features {
		if mostRecent == nil || feature.GetModifiedAt().After(mostRecent.GetModifiedAt()) {
			mostRecent = feature
		}
	}
	return mostRecent
}

// isMainBranch checks if the branch is a main/master/trunk branch
func isMainBranch(branch string) bool {
	return common.IsTrunkBranch(branch)
}

// handleMergeAbort handles `fogit merge --abort`
func handleMergeAbort(ctx context.Context, repo fogit.Repository, gitRepo *git.Repository, opts MergeOptions) (*MergeResult, error) {
	// Check if there's a fogit merge in progress
	state, err := LoadMergeState(opts.FogitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load merge state: %w", err)
	}

	if state == nil {
		return nil, ErrNoMergeInProgress
	}

	result := &MergeResult{
		Aborted:    true,
		Branch:     state.FeatureBranch,
		BaseBranch: state.BaseBranch,
	}

	// Abort git merge if in progress
	if gitRepo.IsMerging() {
		if err := gitRepo.AbortMerge(); err != nil {
			return nil, fmt.Errorf("failed to abort git merge: %w", err)
		}
	}

	// Return to feature branch
	if err := gitRepo.CheckoutBranch(state.FeatureBranch); err != nil {
		// Try harder - reset and checkout
		return nil, fmt.Errorf("failed to return to feature branch %s: %w", state.FeatureBranch, err)
	}

	// Reopen the features that were closed
	for _, featureID := range state.FeatureIDs {
		feature, err := repo.Get(ctx, featureID)
		if err != nil {
			continue // Feature may have been deleted
		}

		// Clear ClosedAt to reopen
		if v := feature.GetCurrentVersion(); v != nil {
			v.ClosedAt = nil
		}

		if err := repo.Update(ctx, feature); err != nil {
			return nil, fmt.Errorf("failed to reopen feature %s: %w", feature.Name, err)
		}
	}

	// Clear merge state
	if err := ClearMergeState(opts.FogitDir); err != nil {
		return nil, fmt.Errorf("failed to clear merge state: %w", err)
	}

	return result, nil
}

// handleMergeContinue handles `fogit merge --continue`
func handleMergeContinue(ctx context.Context, repo fogit.Repository, gitRepo *git.Repository, opts MergeOptions) (*MergeResult, error) {
	// Check if there's a fogit merge in progress
	state, err := LoadMergeState(opts.FogitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load merge state: %w", err)
	}

	if state == nil {
		return nil, ErrNoMergeInProgress
	}

	result := &MergeResult{
		Branch:     state.FeatureBranch,
		BaseBranch: state.BaseBranch,
		NoDelete:   state.NoDelete,
	}

	// Check if git merge is still in progress (conflicts not resolved)
	if gitRepo.IsMerging() {
		// Check for remaining conflicts
		hasConflicts, err := gitRepo.HasConflicts()
		if err != nil {
			return nil, fmt.Errorf("failed to check for conflicts: %w", err)
		}

		if hasConflicts {
			return nil, fmt.Errorf("%w: resolve all conflicts and stage the changes, then run 'fogit merge --continue'", ErrConflictsRemaining)
		}

		// All conflicts resolved, commit the merge
		_, err = gitRepo.Commit("Merge branch '"+state.FeatureBranch+"'", nil)
		if err != nil && err != git.ErrNothingToCommit {
			return nil, fmt.Errorf("failed to commit merge: %w", err)
		}
	}
	// If git merge is not in progress but we have fogit merge state,
	// the user may have already committed manually - that's fine, we just clean up

	result.MergePerformed = true

	// Load closed features for the result
	for _, featureID := range state.FeatureIDs {
		feature, err := repo.Get(ctx, featureID)
		if err == nil {
			result.ClosedFeatures = append(result.ClosedFeatures, feature)
		}
	}

	// Delete feature branch if requested
	if !state.NoDelete {
		if err := gitRepo.DeleteBranch(state.FeatureBranch); err != nil {
			// Non-fatal
		} else {
			result.BranchDeleted = true
		}
	}

	// Clear merge state
	if err := ClearMergeState(opts.FogitDir); err != nil {
		return nil, fmt.Errorf("failed to clear merge state: %w", err)
	}

	return result, nil
}

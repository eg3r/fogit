// Package features provides business logic for feature operations.
package features

import (
	"context"
	"fmt"
	"strings"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/search"
	"github.com/eg3r/fogit/pkg/fogit"
)

// SwitchOptions contains options for switching to a feature
type SwitchOptions struct {
	Identifier string // Feature name, ID, or partial match
	FogitDir   string // Path to .fogit directory
}

// SwitchResult contains the result of a switch operation
type SwitchResult struct {
	Feature            *fogit.Feature
	PreviousBranch     string
	TargetBranch       string
	AlreadyOnBranch    bool
	IsTrunkBased       bool
	FoundOnOtherBranch string         // Non-empty if feature was found on a different branch
	Suggestions        []search.Match // Fuzzy match suggestions if not found
}

// Switch switches to a feature's branch (branch-per-feature mode)
// or sets the active feature context (trunk-based mode)
// Per spec: Uses cross-branch discovery to find features on other branches
func Switch(ctx context.Context, repo fogit.Repository, gitRepo *git.Repository, cfg *fogit.Config, opts SwitchOptions) (*SwitchResult, error) {
	var feature *fogit.Feature
	var foundOnBranch string

	// First, try to find locally on current branch
	findResult, err := Find(ctx, repo, opts.Identifier, cfg)
	if err != nil {
		// Not found locally - try cross-branch discovery
		// Per spec/specification/07-git-integration.md#cross-branch-feature-discovery
		crossResult, crossErr := FindAcrossBranches(ctx, repo, gitRepo, opts.Identifier, cfg)
		if crossErr != nil {
			// Collect suggestions for better error messages
			var suggestions []search.Match
			if crossResult != nil && len(crossResult.Suggestions) > 0 {
				suggestions = crossResult.Suggestions
			} else if findResult != nil && len(findResult.Suggestions) > 0 {
				suggestions = findResult.Suggestions
			}
			return &SwitchResult{Suggestions: suggestions}, err
		}

		if crossResult.IsRemote {
			return nil, fmt.Errorf("feature '%s' found on remote branch '%s'. Run 'git fetch' and 'git checkout %s' first",
				opts.Identifier, crossResult.Branch, extractLocalBranchNameSwitch(crossResult.Branch))
		}

		feature = crossResult.Feature
		foundOnBranch = crossResult.Branch
	} else {
		feature = findResult.Feature
	}

	// Check feature state - can only switch to open features
	// Closed features need to be reopened with 'fogit feature <name> --new-version'
	state := feature.DeriveState()
	if state == fogit.StateClosed {
		return nil, fmt.Errorf("cannot switch to closed feature '%s'. Use 'fogit feature %s --new-version' to reopen it", feature.Name, feature.Name)
	}

	// Handle trunk-based mode
	if cfg.Workflow.Mode == "trunk-based" {
		return &SwitchResult{
			Feature:      feature,
			IsTrunkBased: true,
		}, nil
	}

	// Branch-per-feature mode
	result, err := switchToBranch(feature, gitRepo)
	if err != nil {
		return nil, err
	}

	// Include cross-branch info if feature was found on another branch
	if foundOnBranch != "" {
		result.FoundOnOtherBranch = foundOnBranch
	}

	return result, nil
}

// extractLocalBranchNameSwitch extracts the local branch name from a remote branch reference
// e.g., "origin/feature/auth" -> "feature/auth"
func extractLocalBranchNameSwitch(remoteBranch string) string {
	parts := strings.SplitN(remoteBranch, "/", 2)
	if len(parts) >= 2 {
		return parts[1]
	}
	return remoteBranch
}

// switchToBranch handles the git branch switching logic
func switchToBranch(feature *fogit.Feature, gitRepo *git.Repository) (*SwitchResult, error) {
	// Get current branch
	currentBranch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Determine target branch
	targetBranch := GetFeatureBranch(feature)
	if targetBranch == "" {
		// Feature doesn't have a branch recorded - generate one
		targetBranch = SanitizeBranchName(feature.Name)
	}

	// Check if already on the target branch
	if currentBranch == targetBranch {
		return &SwitchResult{
			Feature:         feature,
			PreviousBranch:  currentBranch,
			TargetBranch:    targetBranch,
			AlreadyOnBranch: true,
		}, nil
	}

	// Check for uncommitted changes
	changedFiles, err := gitRepo.GetChangedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to check for changes: %w", err)
	}

	if len(changedFiles) > 0 {
		return nil, fmt.Errorf("uncommitted changes detected. Please commit or stash your changes before switching:\n  git stash\n  fogit switch %s\n  git stash pop", feature.Name)
	}

	// Switch to the feature's branch
	if err := gitRepo.CheckoutBranch(targetBranch); err != nil {
		// Branch might not exist - try to create it
		if err := gitRepo.CreateBranch(targetBranch); err != nil {
			if err != git.ErrBranchExists {
				return nil, fmt.Errorf("failed to create branch '%s': %w", targetBranch, err)
			}
		}
		// Try checkout again
		if err := gitRepo.CheckoutBranch(targetBranch); err != nil {
			return nil, fmt.Errorf("failed to switch to branch '%s': %w", targetBranch, err)
		}
	}

	return &SwitchResult{
		Feature:        feature,
		PreviousBranch: currentBranch,
		TargetBranch:   targetBranch,
	}, nil
}

// GetFeatureBranch returns the branch associated with a feature
// Checks metadata["branch"] first, then the current version's Branch field
func GetFeatureBranch(feature *fogit.Feature) string {
	// Check metadata first
	if branch, ok := feature.Metadata["branch"].(string); ok && branch != "" {
		return branch
	}

	// Check current version
	currentVersion := feature.GetCurrentVersion()
	if currentVersion != nil && currentVersion.Branch != "" {
		return currentVersion.Branch
	}

	return ""
}

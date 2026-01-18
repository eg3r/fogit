// Package features provides business logic for feature operations.
package features

import (
	"context"
	"fmt"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/pkg/fogit"
)

// SwitchOptions contains options for switching to a feature
type SwitchOptions struct {
	Identifier string // Feature name, ID, or partial match
	FogitDir   string // Path to .fogit directory
}

// SwitchResult contains the result of a switch operation
type SwitchResult struct {
	Feature         *fogit.Feature
	PreviousBranch  string
	TargetBranch    string
	AlreadyOnBranch bool
	IsTrunkBased    bool
}

// Switch switches to a feature's branch (branch-per-feature mode)
// or sets the active feature context (trunk-based mode)
func Switch(ctx context.Context, repo fogit.Repository, gitRepo *git.Repository, cfg *fogit.Config, opts SwitchOptions) (*SwitchResult, error) {
	// Find feature
	findResult, err := Find(ctx, repo, opts.Identifier, cfg)
	if err != nil {
		return &SwitchResult{}, err
	}
	feature := findResult.Feature

	// Check feature state
	state := feature.DeriveState()
	if state == fogit.StateClosed {
		return nil, fmt.Errorf("cannot switch to closed feature '%s'. Use 'fogit feature %s' to reopen it with a new version", feature.Name, feature.Name)
	}

	// Handle trunk-based mode
	if cfg.Workflow.Mode == "trunk-based" {
		return &SwitchResult{
			Feature:      feature,
			IsTrunkBased: true,
		}, nil
	}

	// Branch-per-feature mode
	return switchToBranch(feature, gitRepo)
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

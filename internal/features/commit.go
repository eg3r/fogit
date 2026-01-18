package features

import (
	"context"
	"fmt"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/eg3r/fogit/internal/common"
	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/pkg/fogit"
)

// CommitOptions contains options for the Commit operation
type CommitOptions struct {
	Message    string
	Author     string // Override author (optional)
	AutoLink   bool   // Auto-link changed files to primary feature
	AllowDirty bool   // Allow commit with uncommitted changes in .fogit/
}

// CommitResult contains the result of a Commit operation
type CommitResult struct {
	Hash            string
	Author          *object.Signature
	Branch          string
	Features        []*fogit.Feature
	PrimaryFeature  *fogit.Feature
	ChangedFiles    []string
	LinkedFiles     []string
	NothingToCommit bool
}

// Commit performs a Git commit and updates all features on the current branch
// Per spec: features on shared branches share the branch lifecycle
// Any commit on the branch updates modified_at for ALL features on that branch
func Commit(ctx context.Context, repo fogit.Repository, gitRepo *git.Repository, opts CommitOptions) (*CommitResult, error) {
	result := &CommitResult{}

	// Get current branch
	branch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	result.Branch = branch

	// Find ALL features associated with current branch
	branchFeatures, err := FindAllForBranch(ctx, repo, branch)
	if err != nil {
		return nil, fmt.Errorf("no feature found for current branch: %w", err)
	}

	if len(branchFeatures) == 0 {
		return nil, fmt.Errorf("no active feature found. Create a feature with 'fogit feature'")
	}

	result.Features = branchFeatures
	result.PrimaryFeature = branchFeatures[0]

	// Get changed files
	changedFiles, err := gitRepo.GetChangedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}
	result.ChangedFiles = changedFiles

	if len(changedFiles) == 0 {
		result.NothingToCommit = true
		return result, nil
	}

	// Create author signature
	author, err := resolveAuthor(gitRepo, opts.Author)
	if err != nil {
		return nil, err
	}
	result.Author = author

	// Update ALL features on this branch BEFORE committing
	// Per spec: features on shared branches share the branch lifecycle
	// Per spec 06-data-model.md: creator/contributors are computed from Git history, not stored
	for _, feature := range branchFeatures {
		feature.UpdateModifiedAt()

		// Auto-link files to feature (only primary feature gets file links)
		if opts.AutoLink && feature == result.PrimaryFeature {
			for _, file := range changedFiles {
				// Skip .fogit files
				if common.ShouldSkipFogitFile(file) {
					continue
				}
				feature.AddFile(file)
				result.LinkedFiles = append(result.LinkedFiles, file)
			}
		}

		// Save updated feature to .fogit/ directory
		if err := repo.Update(ctx, feature); err != nil {
			return nil, fmt.Errorf("failed to update feature %s: %w", feature.Name, err)
		}
	}

	// Now commit changes (includes .fogit/ directory)
	hash, err := gitRepo.Commit(opts.Message, author)
	if err != nil {
		if err == git.ErrNothingToCommit {
			result.NothingToCommit = true
			return result, nil
		}
		return nil, fmt.Errorf("failed to commit: %w", err)
	}
	result.Hash = hash

	return result, nil
}

// resolveAuthor resolves the commit author from the provided override or git config
func resolveAuthor(gitRepo *git.Repository, authorOverride string) (*object.Signature, error) {
	if authorOverride != "" {
		return git.ParseAuthor(authorOverride), nil
	}

	// Get from git config
	name, email, err := gitRepo.GetUserConfig()
	if err != nil || email == "" {
		return nil, fmt.Errorf("no author specified and git user.email not configured")
	}

	return &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}, nil
}

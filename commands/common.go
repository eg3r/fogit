package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/logger"
	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// CommandContext holds common dependencies for commands
type CommandContext struct {
	FogitDir string
	Repo     fogit.Repository
	Config   *fogit.Config
	Git      *features.GitIntegration // nil if not in git repo
}

// GetCommandContext initializes all common dependencies for a command
// This reduces boilerplate in individual commands
func GetCommandContext() (*CommandContext, error) {
	fogitDir, err := getFogitDir()
	if err != nil {
		return nil, err
	}

	repo := storage.NewFileRepository(fogitDir)

	cfg, err := config.Load(fogitDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Try to initialize git integration (nil if not in git repo)
	// Derive cwd from fogitDir to avoid duplicate os.Getwd() call
	cwd := filepath.Dir(fogitDir)
	gitIntegration, gitErr := features.NewGitIntegration(cwd, cfg)
	if gitErr != nil {
		logger.Debug("git integration not available", "error", gitErr)
	}

	return &CommandContext{
		FogitDir: fogitDir,
		Repo:     repo,
		Config:   cfg,
		Git:      gitIntegration,
	}, nil
}

// GetCommandContextWithoutConfig initializes dependencies without loading config
// Useful for commands that don't need configuration
func GetCommandContextWithoutConfig() (*CommandContext, error) {
	fogitDir, err := getFogitDir()
	if err != nil {
		return nil, err
	}

	repo := storage.NewFileRepository(fogitDir)

	return &CommandContext{
		FogitDir: fogitDir,
		Repo:     repo,
	}, nil
}

// getRepository creates a file repository for the given .fogit directory
// This is the legacy helper - prefer using GetCommandContext for new code
func getRepository(fogitDir string) fogit.Repository {
	return storage.NewFileRepository(fogitDir)
}

// getFogitDir returns the .fogit directory for the current working directory
// Returns an error if fogit is not initialized
func getFogitDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	fogitDir := filepath.Join(cwd, ".fogit")

	// Check if repository is initialized
	if _, err := os.Stat(fogitDir); os.IsNotExist(err) {
		return "", fmt.Errorf("fogit repository not initialized. Run 'fogit init' first")
	}

	return fogitDir, nil
}

// FindFeatureWithSuggestions finds a feature by ID/name and handles error display with suggestions.
// This consolidates the repeated error handling pattern across commands.
// The suggestCmd parameter is used in the suggestion message (e.g., "fogit show <id>").
func FindFeatureWithSuggestions(ctx context.Context, repo fogit.Repository, identifier string, cfg *fogit.Config, suggestCmd string) (*fogit.Feature, error) {
	result, err := features.Find(ctx, repo, identifier, cfg)
	if err != nil {
		if err == fogit.ErrNotFound && result != nil && len(result.Suggestions) > 0 {
			printer.PrintSuggestions(os.Stdout, identifier, result.Suggestions, suggestCmd)
			return nil, fmt.Errorf("feature not found")
		}
		if err == fogit.ErrNotFound {
			return nil, fmt.Errorf("feature not found: %s", identifier)
		}
		return nil, fmt.Errorf("failed to find feature: %w", err)
	}
	return result.Feature, nil
}

// FindFeatureCrossBranch finds a feature using cross-branch discovery in branch-per-feature mode.
// Falls back to current-branch-only search in trunk-based mode or if git is not available.
func FindFeatureCrossBranch(ctx context.Context, cmdCtx *CommandContext, identifier string, suggestCmd string) (*fogit.Feature, error) {
	result, err := FindFeatureCrossBranchWithResult(ctx, cmdCtx, identifier, suggestCmd)
	if err != nil {
		return nil, err
	}
	return result.Feature, nil
}

// FindFeatureCrossBranchWithResult is like FindFeatureCrossBranch but returns the full result
// including branch information. Use this when you need to know which branch a feature is on.
func FindFeatureCrossBranchWithResult(ctx context.Context, cmdCtx *CommandContext, identifier string, suggestCmd string) (*features.CrossBranchFindResult, error) {
	cfg := cmdCtx.Config

	// Use cross-branch discovery in branch-per-feature mode
	if cfg.Workflow.Mode == "branch-per-feature" && cmdCtx.Git != nil && cmdCtx.Git.GetGitRepo() != nil {
		result, err := features.FindAcrossBranches(ctx, cmdCtx.Repo, cmdCtx.Git.GetGitRepo(), identifier, cfg)
		if err != nil {
			if err == fogit.ErrNotFound && result != nil && len(result.Suggestions) > 0 {
				printer.PrintSuggestions(os.Stdout, identifier, result.Suggestions, suggestCmd)
				return nil, fmt.Errorf("feature not found")
			}
			if err == fogit.ErrNotFound {
				return nil, fmt.Errorf("feature not found: %s", identifier)
			}
			return nil, fmt.Errorf("failed to find feature: %w", err)
		}
		return result, nil
	}

	// Fallback to current branch only
	findResult, err := features.Find(ctx, cmdCtx.Repo, identifier, cfg)
	if err != nil {
		if err == fogit.ErrNotFound && findResult != nil && len(findResult.Suggestions) > 0 {
			printer.PrintSuggestions(os.Stdout, identifier, findResult.Suggestions, suggestCmd)
			return nil, fmt.Errorf("feature not found")
		}
		if err == fogit.ErrNotFound {
			return nil, fmt.Errorf("feature not found: %s", identifier)
		}
		return nil, fmt.Errorf("failed to find feature: %w", err)
	}

	// Get current branch for the result
	currentBranch := ""
	if cmdCtx.Git != nil && cmdCtx.Git.GetGitRepo() != nil {
		currentBranch, _ = cmdCtx.Git.GetGitRepo().GetCurrentBranch()
	}

	return &features.CrossBranchFindResult{
		Feature:  findResult.Feature,
		Branch:   currentBranch,
		IsRemote: false,
	}, nil
}

// ListFeaturesCrossBranch lists all features using cross-branch discovery in branch-per-feature mode.
// Falls back to current-branch-only listing in trunk-based mode or if git is not available.
func ListFeaturesCrossBranch(ctx context.Context, cmdCtx *CommandContext, filter *fogit.Filter) ([]*fogit.Feature, error) {
	cfg := cmdCtx.Config

	// Use cross-branch discovery in branch-per-feature mode
	if cfg.Workflow.Mode == "branch-per-feature" && cmdCtx.Git != nil && cmdCtx.Git.GetGitRepo() != nil {
		crossBranchFeatures, err := features.ListFeaturesAcrossBranches(ctx, cmdCtx.Repo, cmdCtx.Git.GetGitRepo())
		if err != nil {
			return nil, fmt.Errorf("failed to list features across branches: %w", err)
		}

		// Extract features and apply filter
		var featuresList []*fogit.Feature
		for _, cbf := range crossBranchFeatures {
			if filter == nil || filter.Matches(cbf.Feature) {
				featuresList = append(featuresList, cbf.Feature)
			}
		}
		return featuresList, nil
	}

	// Fallback to current branch only
	return cmdCtx.Repo.List(ctx, filter)
}

package features

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/logger"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// GitIntegration provides a unified service for feature-aware git operations
type GitIntegration struct {
	workDir  string
	gitRoot  string
	gitRepo  *git.Repository
	config   *fogit.Config
	fogitDir string
}

// NewGitIntegration creates a new git integration service
// Returns nil if not in a git repository (which is not an error)
func NewGitIntegration(workDir string, config *fogit.Config) (*GitIntegration, error) {
	gitRoot, err := git.FindGitRoot(workDir)
	if err != nil {
		// Not in a git repo - return nil without error
		return nil, nil
	}

	gitRepo, err := git.OpenRepository(gitRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	return &GitIntegration{
		workDir:  workDir,
		gitRoot:  gitRoot,
		gitRepo:  gitRepo,
		config:   config,
		fogitDir: filepath.Join(workDir, ".fogit"),
	}, nil
}

// IsAvailable returns true if git integration is available
func (gi *GitIntegration) IsAvailable() bool {
	return gi != nil && gi.gitRepo != nil
}

// GetGitRepo returns the underlying git repository
func (gi *GitIntegration) GetGitRepo() *git.Repository {
	if gi == nil {
		return nil
	}
	return gi.gitRepo
}

// GetCurrentBranch returns the current git branch name
func (gi *GitIntegration) GetCurrentBranch() (string, error) {
	if !gi.IsAvailable() {
		return "", fmt.Errorf("not in a git repository")
	}
	return gi.gitRepo.GetCurrentBranch()
}

// CreateFeatureBranch creates a branch for a feature using the configured naming convention
func (gi *GitIntegration) CreateFeatureBranch(feature *fogit.Feature, checkout bool) error {
	if !gi.IsAvailable() {
		return nil // Skip silently if not in git repo
	}

	branchName := gi.GenerateBranchName(feature)

	if err := gi.gitRepo.CreateBranch(branchName); err != nil {
		if err == git.ErrBranchExists {
			return fmt.Errorf("branch %s already exists", branchName)
		}
		return fmt.Errorf("failed to create branch: %w", err)
	}

	if checkout {
		if err := gi.gitRepo.CheckoutBranch(branchName); err != nil {
			return fmt.Errorf("failed to checkout branch: %w", err)
		}
	}

	// Update feature version with branch info
	if cv := feature.GetCurrentVersion(); cv != nil {
		cv.Branch = branchName
	}

	return nil
}

// SwitchToFeatureBranch switches to a feature's branch if it exists
func (gi *GitIntegration) SwitchToFeatureBranch(feature *fogit.Feature) error {
	if !gi.IsAvailable() {
		return fmt.Errorf("not in a git repository")
	}

	cv := feature.GetCurrentVersion()
	if cv == nil || cv.Branch == "" {
		// Try to find branch by feature name
		branchName := gi.GenerateBranchName(feature)
		if err := gi.gitRepo.CheckoutBranch(branchName); err != nil {
			return fmt.Errorf("feature has no associated branch and %s not found", branchName)
		}
		return nil
	}

	return gi.gitRepo.CheckoutBranch(cv.Branch)
}

// GetChangedFeatureFiles returns a list of feature files that have been modified
func (gi *GitIntegration) GetChangedFeatureFiles(ctx context.Context, repo fogit.Repository) ([]*fogit.Feature, error) {
	if !gi.IsAvailable() {
		return nil, nil
	}

	changedFiles, err := gi.gitRepo.GetChangedFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get changed files: %w", err)
	}

	// Filter for .fogit/features/*.yml files
	var featureFiles []string
	for _, file := range changedFiles {
		if strings.HasPrefix(file, ".fogit/features/") && strings.HasSuffix(file, ".yml") {
			featureFiles = append(featureFiles, file)
		}
	}

	if len(featureFiles) == 0 {
		return nil, nil
	}

	// Load all features and match by filename
	allFeatures, err := repo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	// Build filename -> feature map
	var changedFeatures []*fogit.Feature
	for _, f := range allFeatures {
		fileName := storage.GenerateFeatureFilename(f.Name, f.ID, nil)
		for _, changedFile := range featureFiles {
			if strings.HasSuffix(changedFile, fileName) {
				changedFeatures = append(changedFeatures, f)
				break
			}
		}
	}

	return changedFeatures, nil
}

// CommitFeatureChange commits changes related to a feature
func (gi *GitIntegration) CommitFeatureChange(feature *fogit.Feature, action string) error {
	if !gi.IsAvailable() {
		return nil // Skip silently if not in git repo
	}

	// Get author from git config
	name, email, err := gi.gitRepo.GetUserConfig()
	if err != nil || email == "" {
		return fmt.Errorf("git user.email not configured")
	}

	author := &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}

	// Generate commit message
	message := gi.formatCommitMessage(feature, action)

	// Commit
	hash, err := gi.gitRepo.Commit(message, author)
	if err != nil {
		if err == git.ErrNothingToCommit {
			return nil // Not an error
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	fmt.Printf("Committed: %s\n", hash[:8])
	return nil
}

// AutoCommitAndPush commits and optionally pushes changes
func (gi *GitIntegration) AutoCommitAndPush(feature *fogit.Feature, action string) error {
	if !gi.IsAvailable() {
		return nil
	}

	if err := gi.CommitFeatureChange(feature, action); err != nil {
		return err
	}

	// Auto-push if enabled
	if gi.config.AutoPush {
		if err := gi.gitRepo.Push("origin"); err != nil {
			logger.Warn("failed to auto-push", "error", err)
		} else {
			fmt.Println("Pushed to origin")
		}
	}

	return nil
}

// HasUncommittedChanges checks if there are uncommitted changes
func (gi *GitIntegration) HasUncommittedChanges() (bool, error) {
	if !gi.IsAvailable() {
		return false, nil
	}

	files, err := gi.gitRepo.GetChangedFiles()
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

// HasUncommittedFeatureChanges checks if there are uncommitted changes in the .fogit directory
func (gi *GitIntegration) HasUncommittedFeatureChanges() (bool, error) {
	if !gi.IsAvailable() {
		return false, nil
	}

	files, err := gi.gitRepo.GetChangedFiles()
	if err != nil {
		return false, err
	}

	for _, file := range files {
		if strings.HasPrefix(file, ".fogit/") {
			return true, nil
		}
	}
	return false, nil
}

// GenerateBranchName generates a branch name for a feature
func (gi *GitIntegration) GenerateBranchName(feature *fogit.Feature) string {
	opts := storage.SlugifyOptions{
		MaxLength:        240,
		AllowSlashes:     true,
		NormalizeUnicode: true,
		EmptyFallback:    "unnamed",
	}

	slug := storage.Slugify(feature.Name, opts)

	// Include version if available
	if versionKey := feature.GetCurrentVersionKey(); versionKey != "" {
		return fmt.Sprintf("feature/%s-v%s", slug, versionKey)
	}

	return fmt.Sprintf("feature/%s", slug)
}

// formatCommitMessage generates a commit message using the template
func (gi *GitIntegration) formatCommitMessage(feature *fogit.Feature, action string) string {
	template := gi.config.CommitTemplate
	if template == "" {
		template = "feat: {title} ({id})"
	}

	msg := template
	msg = strings.ReplaceAll(msg, "{title}", feature.Name)
	msg = strings.ReplaceAll(msg, "{name}", feature.Name)
	msg = strings.ReplaceAll(msg, "{id}", feature.ID[:8])
	msg = strings.ReplaceAll(msg, "{action}", action)

	return msg
}

// GetFileHistory returns the commit history for a feature file
func (gi *GitIntegration) GetFileHistory(feature *fogit.Feature, limit int) ([]git.CommitInfo, error) {
	if !gi.IsAvailable() {
		return nil, fmt.Errorf("not in a git repository")
	}

	// Get the feature filename
	fileName := storage.GenerateFeatureFilename(feature.Name, feature.ID, nil)
	filePath := filepath.Join(".fogit", "features", fileName)

	return gi.gitRepo.GetCommitHistory(filePath, limit)
}

// GetAuthors returns all authors who have modified a feature file
func (gi *GitIntegration) GetAuthors(feature *fogit.Feature) ([]string, error) {
	if !gi.IsAvailable() {
		return nil, nil
	}

	fileName := storage.GenerateFeatureFilename(feature.Name, feature.ID, nil)
	filePath := filepath.Join(".fogit", "features", fileName)

	return gi.gitRepo.GetAuthorsForFile(filePath)
}

// GetGitRoot returns the git repository root
func (gi *GitIntegration) GetGitRoot() string {
	if !gi.IsAvailable() {
		return ""
	}
	return gi.gitRoot
}

// GetRepositoryName returns the repository name from the remote
func (gi *GitIntegration) GetRepositoryName() string {
	if !gi.IsAvailable() {
		return ""
	}
	return gi.gitRepo.GetRepositoryName()
}

// InitGitIntegration is a helper that creates a GitIntegration from the current directory
func InitGitIntegration(config *fogit.Config) (*GitIntegration, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return NewGitIntegration(cwd, config)
}

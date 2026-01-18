package features

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/logger"
	"github.com/eg3r/fogit/pkg/fogit"
)

// AutoCommitFeature commits the feature file to Git if in a Git repository
func AutoCommitFeature(feature *fogit.Feature, action string, cfg *fogit.Config) error {
	// Find git root
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if we're in a Git repository
	gitRoot, err := git.FindGitRoot(cwd)
	if err != nil {
		// Not in a Git repo - skip auto-commit
		return nil
	}

	// Open git repository
	gitRepo, err := git.OpenRepository(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to open git repository: %w", err)
	}

	// Get author from Git config
	name, email, err := gitRepo.GetUserConfig()
	if err != nil || email == "" {
		return fmt.Errorf("git user.email not configured")
	}

	author := &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}

	// Generate commit message using template
	commitMsg := generateCommitMessage(cfg.CommitTemplate, feature, action)

	// Commit
	hash, err := gitRepo.Commit(commitMsg, author)
	if err != nil {
		if err == git.ErrNothingToCommit {
			// Already committed or no changes
			return nil
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	fmt.Printf("✓ Committed: %s\n", hash[:8])

	// Auto-push if enabled
	if cfg.AutoPush {
		if err := gitRepo.Push("origin"); err != nil {
			// Don't fail on push error, just warn
			logger.Warn("failed to auto-push", "error", err)
		} else {
			fmt.Println("✓ Pushed to origin")
		}
	}

	return nil
}

// generateCommitMessage creates a commit message from the template
func generateCommitMessage(template string, feature *fogit.Feature, action string) string {
	msg := template
	msg = strings.ReplaceAll(msg, "{title}", feature.Name)
	msg = strings.ReplaceAll(msg, "{name}", feature.Name)
	msg = strings.ReplaceAll(msg, "{id}", feature.ID)
	msg = strings.ReplaceAll(msg, "{action}", action)

	// If no template placeholders, use default format
	if msg == template {
		return fmt.Sprintf("[FoGit] %s feature: %s", action, feature.Name)
	}

	return msg
}

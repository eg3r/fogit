package commands

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/pkg/fogit"
)

// TestBranchCreation_Integration tests actual Git branch creation
func TestBranchCreation_Integration(t *testing.T) {
	tests := []struct {
		name          string
		featureName   string
		mode          string
		sameFlag      bool
		isolateFlag   bool
		allowShared   bool
		initialBranch string
		expectBranch  string
		expectError   bool
		setupGit      bool
	}{
		{
			name:          "creates branch in branch-per-feature mode",
			featureName:   "User Authentication",
			mode:          "branch-per-feature",
			sameFlag:      false,
			isolateFlag:   false,
			allowShared:   true,
			initialBranch: "main",
			expectBranch:  "feature/user-authentication",
			setupGit:      true,
		},
		{
			name:          "stays on current branch with --same",
			featureName:   "Quick Fix",
			mode:          "branch-per-feature",
			sameFlag:      true,
			isolateFlag:   false,
			allowShared:   true,
			initialBranch: "feature/existing-feature",
			expectBranch:  "feature/existing-feature",
			setupGit:      true,
		},
		{
			name:          "creates branch with --isolate",
			featureName:   "New Feature",
			mode:          "branch-per-feature",
			sameFlag:      false,
			isolateFlag:   true,
			allowShared:   false,
			initialBranch: "main",
			expectBranch:  "feature/new-feature",
			setupGit:      true,
		},
		{
			name:          "trunk-based mode stays on main",
			featureName:   "Hotfix",
			mode:          "trunk-based",
			sameFlag:      false,
			isolateFlag:   false,
			allowShared:   true,
			initialBranch: "main",
			expectBranch:  "master", // Git creates master by default
			setupGit:      true,
		},
		{
			name:          "handles non-git repo gracefully",
			featureName:   "Test",
			mode:          "branch-per-feature",
			sameFlag:      false,
			isolateFlag:   false,
			allowShared:   true,
			initialBranch: "",
			expectBranch:  "",
			setupGit:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Setup Git repo if needed
			if tt.setupGit {
				if err := setupTestGitRepo(tmpDir, tt.initialBranch); err != nil {
					t.Fatalf("failed to setup git repo: %v", err)
				}
			}

			// Setup .fogit directory
			fogitDir := filepath.Join(tmpDir, ".fogit")
			if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
				t.Fatalf("failed to create .fogit dir: %v", err)
			}

			// Create config
			cfg := &fogit.Config{
				Workflow: fogit.WorkflowConfig{
					Mode:                tt.mode,
					BaseBranch:          "main",
					CreateBranchFrom:    "current", // Use current to avoid branch switching in tests
					AllowSharedBranches: tt.allowShared,
				},
			}

			// Save config
			if err := saveTestConfig(fogitDir, cfg); err != nil {
				t.Fatalf("failed to save config: %v", err)
			}

			// Build command args with -C flag
			args := []string{"-C", tmpDir, "feature", tt.featureName}
			if tt.sameFlag {
				args = append(args, "--same")
			}
			if tt.isolateFlag {
				args = append(args, "--isolate")
			}

			// Reset flags and run with -C flag
			ResetFlags()
			rootCmd.SetArgs(args)
			err := ExecuteRootCmd()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify current branch if Git repo
			if tt.setupGit && tt.expectBranch != "" {
				repo, err := git.OpenRepository(tmpDir)
				if err != nil {
					t.Fatalf("failed to open repo: %v", err)
				}

				currentBranch, err := repo.GetCurrentBranch()
				if err != nil {
					t.Fatalf("failed to get current branch: %v", err)
				}

				if currentBranch != tt.expectBranch {
					t.Errorf("expected branch %q, got %q", tt.expectBranch, currentBranch)
				}
			}
		})
	}
}

// TestBranchCreation_BranchExists tests handling of existing branches
func TestBranchCreation_BranchExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup Git repo
	if err := setupTestGitRepo(tmpDir, "main"); err != nil {
		t.Fatalf("failed to setup git repo: %v", err)
	}

	// Create a feature branch that already exists
	repo, err := git.OpenRepository(tmpDir)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	if err := repo.CreateBranch("feature/existing-feature"); err != nil {
		t.Fatalf("failed to create existing branch: %v", err)
	}

	// Setup .fogit
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatalf("failed to create .fogit dir: %v", err)
	}

	cfg := &fogit.Config{
		Workflow: fogit.WorkflowConfig{
			Mode:                "branch-per-feature",
			BaseBranch:          "main",
			CreateBranchFrom:    "current", // Use current to avoid branch switching in tests
			AllowSharedBranches: true,
		},
	}

	if err := saveTestConfig(fogitDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "feature", "Existing Feature"})

	// Try to create a feature with same name (should error about branch exists)
	err = ExecuteRootCmd()
	if err == nil {
		t.Error("expected error about branch already exists")
	}
	if err != nil && !containsString(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

// TestBranchCreation_WithUnstagedChanges tests that creating a new branch works
// with unstaged changes (changes are carried over to the new branch, which is
// standard git behavior)
func TestBranchCreation_WithUnstagedChanges(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup Git repo
	if err := setupTestGitRepo(tmpDir, "main"); err != nil {
		t.Fatalf("failed to setup git repo: %v", err)
	}

	// Open the repo and modify an existing tracked file
	repo, err := gogit.PlainOpen(tmpDir)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}

	// Modify the README that was created in setup
	readmePath := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Modified"), 0644); err != nil {
		t.Fatalf("failed to modify README: %v", err)
	}

	// Verify there are unstaged changes
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	status, err := wt.Status()
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status.IsClean() {
		t.Fatal("expected unstaged changes but repo is clean")
	}

	// Setup .fogit
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatalf("failed to create .fogit dir: %v", err)
	}

	cfg := &fogit.Config{
		Workflow: fogit.WorkflowConfig{
			Mode:                "branch-per-feature",
			BaseBranch:          "main",
			CreateBranchFrom:    "current", // Use current to avoid branch switching in tests
			AllowSharedBranches: true,
		},
	}

	if err := saveTestConfig(fogitDir, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "feature", "New Feature"})

	// Creating a new branch with unstaged changes should succeed
	// (this is standard git behavior - changes carry over to the new branch)
	err = ExecuteRootCmd()
	if err != nil {
		t.Errorf("expected success with unstaged changes, got error: %v", err)
	}
}

// Helper functions

func setupTestGitRepo(dir string, initialBranch string) error {
	// Initialize Git repo
	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		return err
	}

	// Create initial commit
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	// Create a file to commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		return err
	}

	if _, err := wt.Add("README.md"); err != nil {
		return err
	}

	if _, err := wt.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		return err
	}

	// Create and checkout initial branch if not main
	if initialBranch != "main" && initialBranch != "master" {
		headRef, err := repo.Head()
		if err != nil {
			return err
		}

		ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(initialBranch), headRef.Hash())
		if err := repo.Storer.SetReference(ref); err != nil {
			return err
		}

		if err := wt.Checkout(&gogit.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(initialBranch),
		}); err != nil {
			return err
		}
	}

	return nil
}

func saveTestConfig(fogitDir string, cfg *fogit.Config) error {
	return config.Save(fogitDir, cfg)
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		len(s) >= len(substr) &&
		findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

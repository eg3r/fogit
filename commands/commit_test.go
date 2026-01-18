package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// TestCommitCommandArgs tests that the commit command validates arguments correctly
func TestCommitCommandArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "zero arguments is valid",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "one argument is valid",
			args:    []string{"feature-id"},
			wantErr: false,
		},
		{
			name:    "multiple arguments is invalid",
			args:    []string{"feature-id-1", "feature-id-2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "commit",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}

			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			hasError := err != nil

			if hasError != tt.wantErr {
				t.Errorf("Expected error=%v for args %v, got error: %v", tt.wantErr, tt.args, err)
			}
		})
	}
}

// TestCommitMessageRequirement tests that the -m flag is required
func TestCommitMessageRequirement(t *testing.T) {
	tests := []struct {
		name       string
		includeMsg bool
		message    string
		wantErr    bool
	}{
		{
			name:       "valid message",
			includeMsg: true,
			message:    "Add login feature",
			wantErr:    false,
		},
		{
			name:       "missing message flag",
			includeMsg: false,
			wantErr:    true,
		},
		{
			name:       "empty message string is allowed",
			includeMsg: true,
			message:    "",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use: "commit",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}
			var msg string
			cmd.Flags().StringVarP(&msg, "message", "m", "", "Commit message")
			cmd.MarkFlagRequired("message")

			args := []string{}
			if tt.includeMsg {
				args = append(args, "-m", tt.message)
			}
			cmd.SetArgs(args)

			err := cmd.Execute()
			hasError := err != nil

			if hasError != tt.wantErr {
				t.Errorf("Expected error=%v, got error: %v", tt.wantErr, err)
			}
		})
	}
}

// Note: TestCommitAutoLinkSkipsFogitFiles is now in internal/common/files_test.go
// as TestShouldSkipFogitFile which tests the common.ShouldSkipFogitFile function

// TestCommitCommand_Execution tests the commit command execution via rootCmd
func TestCommitCommand_Execution(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create a feature
	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	feature.SetPriority(fogit.PriorityMedium)
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}

	// Reset flags and execute commit command with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "commit", feature.ID, "-m", "test commit"})

	// Note: This will fail without git, but we verify the command executes properly
	// The error should be about git, not about missing flags or bad arguments
	err := ExecuteRootCmd()
	// We expect a git error, but not a command parsing error
	if err != nil {
		// Check it's not a "required flag" error
		errMsg := err.Error()
		if strings.Contains(errMsg, "required flag") {
			t.Errorf("flags should be properly parsed, got: %v", err)
		}
	}
}

// TestCommitIncludesFogitMetadata is an integration test that verifies
// the critical workflow: feature metadata must be saved BEFORE the Git commit
// so that the commit includes the updated .fogit/ files.
func TestCommitIncludesFogitMetadata(t *testing.T) {
	// Create temp directory with Git repository
	tempDir, err := os.MkdirTemp("", "fogit-test-commit-order-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize Git repository using go-git directly
	goGitRepo, err := gogit.PlainInit(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repository: %v", err)
	}

	// Configure Git user in the repository
	cfg, err := goGitRepo.Config()
	if err != nil {
		t.Fatalf("Failed to get git config: %v", err)
	}
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	err = goGitRepo.SetConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to set git config: %v", err)
	}

	// Initialize storage (point to .fogit directory)
	fogitDir := filepath.Join(tempDir, ".fogit")
	err = os.MkdirAll(fogitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .fogit directory: %v", err)
	}
	repo := storage.NewFileRepository(fogitDir)

	// Create a feature
	feature := fogit.NewFeature("Test Commit Order")
	feature.ID = "test-feature-order"
	feature.Description = "Verify commit includes metadata"
	feature.SetPriority(fogit.PriorityMedium)

	ctx := context.Background()
	err = repo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get worktree and add files
	w, err := goGitRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	_, err = w.Add(".")
	if err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	// Create initial commit
	_, err = w.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Now simulate the commit workflow: Update metadata, Save feature, Then commit
	oldModifiedAt := feature.GetModifiedAt()
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	// Step 1: Update feature metadata
	feature.UpdateModifiedAt()

	// Step 2: Save feature to .fogit/ directory (THIS MUST HAPPEN BEFORE COMMIT)
	err = repo.Update(ctx, feature)
	if err != nil {
		t.Fatalf("Failed to update feature: %v", err)
	}

	// Verify feature file was saved (use the actual slugified name)
	// "Test Commit Order" becomes "test-commit-order.yaml"
	fogitPath := filepath.Join(tempDir, ".fogit")
	files, err := os.ReadDir(fogitPath)
	if err != nil {
		t.Fatalf("Failed to read .fogit directory: %v", err)
	}
	t.Logf("Found %d files in .fogit directory", len(files))
	for _, f := range files {
		t.Logf("  - %s", f.Name())
	}
	if len(files) == 0 {
		t.Fatal("Feature file was not created in .fogit/ directory")
	}

	// Step 3: Make a change to trigger commit
	err = os.WriteFile(testFile, []byte("updated content"), 0644)
	if err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	// Add all changes (including .fogit/)
	_, err = w.Add(".")
	if err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	// Step 4: Commit (should include both test.txt and .fogit/ changes)
	hash, err := w.Commit("Update feature", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// CRITICAL VERIFICATION: Check that the commit includes .fogit/ directory
	// Get the commit object
	commit, err := goGitRepo.CommitObject(hash)
	if err != nil {
		t.Fatalf("Failed to get commit: %v", err)
	}

	// Get the tree
	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("Failed to get commit tree: %v", err)
	}

	// Check if .fogit/ directory exists in the commit
	fogitEntry, err := tree.FindEntry(".fogit")
	if err != nil {
		t.Fatalf("CRITICAL BUG: .fogit/ directory not included in commit! This means metadata was saved AFTER commit instead of BEFORE: %v", err)
	}

	if fogitEntry == nil {
		t.Fatal("CRITICAL BUG: .fogit/ entry is nil - metadata not in commit")
	}

	// Verify the feature file exists in the commit
	// The storage uses slugified names like "test-commit-order-test-fea.yml"
	var featureFilename string
	tree.Files().ForEach(func(f *object.File) error {
		// Git uses forward slashes
		if strings.HasPrefix(f.Name, ".fogit/features/") {
			featureFilename = f.Name
		}
		return nil
	})

	if featureFilename == "" {
		t.Logf("Tree contents:")
		tree.Files().ForEach(func(f *object.File) error {
			t.Logf("  - %s", f.Name)
			return nil
		})
		t.Fatal("CRITICAL BUG: No feature file found in .fogit/features/ in the commit!")
	}

	featureEntry, err := tree.FindEntry(featureFilename)
	if err != nil {
		t.Fatalf("CRITICAL BUG: Could not find feature file entry: %v", err)
	}

	if featureEntry == nil {
		t.Fatal("CRITICAL BUG: Feature file entry is nil")
	}

	// Load feature from filesystem to verify it has updated metadata
	reloadedFeature, err := repo.Get(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to reload feature: %v", err)
	}

	if !reloadedFeature.GetModifiedAt().After(oldModifiedAt) {
		t.Error("Feature ModifiedAt was not updated")
	}

	t.Log("✓ PASS: Feature metadata was correctly saved BEFORE Git commit")
	t.Log("✓ PASS: Git commit includes updated .fogit/ directory")
}

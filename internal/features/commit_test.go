package features

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// setupTestGitRepo creates a temp directory with a Git repo and .fogit directory
func setupTestGitRepo(t *testing.T) (string, *git.Repository, fogit.Repository, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "fogit-commit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	// Initialize Git repository
	goGitRepo, err := gogit.PlainInit(tempDir, false)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to init git repository: %v", err)
	}

	// Configure Git user
	cfg, err := goGitRepo.Config()
	if err != nil {
		cleanup()
		t.Fatalf("Failed to get git config: %v", err)
	}
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	if err = goGitRepo.SetConfig(cfg); err != nil {
		cleanup()
		t.Fatalf("Failed to set git config: %v", err)
	}

	// Create initial commit (needed for branches to work)
	testFile := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		cleanup()
		t.Fatalf("Failed to create test file: %v", err)
	}

	w, err := goGitRepo.Worktree()
	if err != nil {
		cleanup()
		t.Fatalf("Failed to get worktree: %v", err)
	}

	if _, err = w.Add("."); err != nil {
		cleanup()
		t.Fatalf("Failed to add files: %v", err)
	}

	if _, err = w.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	}); err != nil {
		cleanup()
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Initialize .fogit directory
	fogitDir := filepath.Join(tempDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		cleanup()
		t.Fatalf("Failed to create .fogit directory: %v", err)
	}

	// Open git repository using our wrapper
	gitRepo, err := git.OpenRepository(tempDir)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to open git repository: %v", err)
	}

	repo := storage.NewFileRepository(fogitDir)

	return tempDir, gitRepo, repo, cleanup
}

func TestCommit_UpdatesFeatureMetadata(t *testing.T) {
	tempDir, gitRepo, repo, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a feature
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	oldModifiedAt := feature.GetModifiedAt()
	time.Sleep(10 * time.Millisecond) // Ensure time difference

	// Create a change to commit (Commit method auto-adds all changes)
	testFile := filepath.Join(tempDir, "new_file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Execute commit (Commit service calls gitRepo.Commit which auto-stages files)
	opts := CommitOptions{
		Message: "Test commit",
	}

	result, err := Commit(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify result
	if result.NothingToCommit {
		t.Error("Expected changes to commit")
	}
	if result.Hash == "" {
		t.Error("Expected commit hash")
	}
	if result.Branch == "" {
		t.Error("Expected branch name")
	}
	if len(result.Features) == 0 {
		t.Error("Expected at least one feature")
	}
	if result.PrimaryFeature == nil {
		t.Error("Expected primary feature")
	}

	// Verify feature metadata was updated
	updatedFeature, err := repo.Get(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get updated feature: %v", err)
	}

	if !updatedFeature.GetModifiedAt().After(oldModifiedAt) {
		t.Error("Feature ModifiedAt was not updated")
	}
}

func TestCommit_AutoLinkFiles(t *testing.T) {
	tempDir, gitRepo, repo, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a feature
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create files to commit
	srcFile := filepath.Join(tempDir, "src", "main.go")
	if err := os.MkdirAll(filepath.Dir(srcFile), 0755); err != nil {
		t.Fatalf("Failed to create src dir: %v", err)
	}
	if err := os.WriteFile(srcFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Execute commit with auto-link (Commit service auto-stages files)
	opts := CommitOptions{
		Message:  "Add main.go",
		AutoLink: true,
	}

	result, err := Commit(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify files were linked
	if len(result.LinkedFiles) == 0 {
		t.Error("Expected files to be linked")
	}

	// Verify .fogit files are not linked
	for _, file := range result.LinkedFiles {
		if strings.HasPrefix(file, ".fogit/") {
			t.Errorf("Expected .fogit/ files to be excluded, got: %s", file)
		}
	}

	// Verify feature has linked files
	updatedFeature, err := repo.Get(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get updated feature: %v", err)
	}

	if len(updatedFeature.Files) == 0 {
		t.Error("Expected feature to have linked files")
	}
}

func TestCommit_NothingToCommit(t *testing.T) {
	_, gitRepo, repo, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a feature
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// First commit to clear the feature file from staging
	if _, err := gitRepo.Commit("Add feature", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	// Now try to commit without any changes (working tree is clean)
	opts := CommitOptions{
		Message: "Empty commit",
	}

	result, err := Commit(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if !result.NothingToCommit {
		t.Error("Expected NothingToCommit to be true")
	}
}

func TestCommit_NoFeatureOnBranch(t *testing.T) {
	_, gitRepo, repo, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Don't create any feature

	opts := CommitOptions{
		Message: "Test commit",
	}

	_, err := Commit(ctx, repo, gitRepo, opts)
	if err == nil {
		t.Error("Expected error when no feature exists")
	}

	if !strings.Contains(err.Error(), "no active feature") {
		t.Errorf("Expected 'no active feature' error, got: %v", err)
	}
}

func TestCommit_CustomAuthor(t *testing.T) {
	tempDir, gitRepo, repo, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a feature
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create a change (Commit service auto-stages files)
	testFile := filepath.Join(tempDir, "new_file.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Execute commit with custom author
	opts := CommitOptions{
		Message: "Test commit",
		Author:  "Custom User <custom@example.com>",
	}

	result, err := Commit(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	if result.Author == nil {
		t.Fatal("Expected author in result")
	}
	if result.Author.Email != "custom@example.com" {
		t.Errorf("Expected custom email, got: %s", result.Author.Email)
	}
}

func TestCommit_MultipleFeaturesOnBranch(t *testing.T) {
	tempDir, gitRepo, repo, cleanup := setupTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Get current branch
	branch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}

	// Create multiple features on the same branch
	feature1 := fogit.NewFeature("Feature 1")
	feature1.Metadata["branch"] = branch
	if err := repo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create feature1: %v", err)
	}

	feature2 := fogit.NewFeature("Feature 2")
	feature2.Metadata["branch"] = branch
	if err := repo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create feature2: %v", err)
	}

	oldModified1 := feature1.GetModifiedAt()
	oldModified2 := feature2.GetModifiedAt()
	time.Sleep(10 * time.Millisecond)

	// Create a change (Commit service auto-stages files)
	testFile := filepath.Join(tempDir, "shared_file.txt")
	if err := os.WriteFile(testFile, []byte("shared"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Execute commit
	opts := CommitOptions{
		Message: "Shared commit",
	}

	result, err := Commit(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	// Verify both features were updated
	if len(result.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(result.Features))
	}

	// Verify both features have updated timestamps
	updated1, _ := repo.Get(ctx, feature1.ID)
	updated2, _ := repo.Get(ctx, feature2.ID)

	if !updated1.GetModifiedAt().After(oldModified1) {
		t.Error("Feature 1 ModifiedAt was not updated")
	}
	if !updated2.GetModifiedAt().After(oldModified2) {
		t.Error("Feature 2 ModifiedAt was not updated")
	}
}

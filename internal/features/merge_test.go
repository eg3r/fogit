package features

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// setupMergeTestGitRepo creates a temp directory with a Git repo and .fogit directory for merge tests
// Returns: tempDir, fogitDir, gitRepo, featureRepo, cleanup
func setupMergeTestGitRepo(t *testing.T) (string, string, *git.Repository, fogit.Repository, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "fogit-merge-test-*")
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

	// Create initial commit
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

	return tempDir, fogitDir, gitRepo, repo, cleanup
}

func TestMerge_ClosesFeature(t *testing.T) {
	_, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Get current branch
	branch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}

	// Create a feature on this branch
	feature := fogit.NewFeature("Test Feature")
	feature.Metadata["branch"] = branch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file so merge doesn't fail on uncommitted changes
	if _, err := gitRepo.Commit("Add feature file", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	// Verify feature is open
	if feature.DeriveState() != fogit.StateOpen {
		t.Errorf("Expected state open, got %s", feature.DeriveState())
	}

	// Execute merge
	opts := MergeOptions{FogitDir: fogitDir}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Verify result
	if len(result.ClosedFeatures) != 1 {
		t.Errorf("Expected 1 closed feature, got %d", len(result.ClosedFeatures))
	}
	if result.Branch == "" {
		t.Error("Expected branch name")
	}

	// Verify feature is closed
	closedFeature, err := repo.Get(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get closed feature: %v", err)
	}

	if closedFeature.DeriveState() != fogit.StateClosed {
		t.Errorf("Expected state closed, got %s", closedFeature.DeriveState())
	}
	if closedFeature.GetClosedAt() == nil {
		t.Error("Expected ClosedAt to be set")
	}
}

func TestMerge_ClosesMultipleFeaturesOnBranch(t *testing.T) {
	_, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Get current branch
	branch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}

	// Create multiple features on this branch
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

	feature3 := fogit.NewFeature("Feature 3")
	feature3.Metadata["branch"] = branch
	if err := repo.Create(ctx, feature3); err != nil {
		t.Fatalf("Failed to create feature3: %v", err)
	}

	// Commit the feature files so merge doesn't fail on uncommitted changes
	if _, err := gitRepo.Commit("Add feature files", nil); err != nil {
		t.Fatalf("Failed to commit features: %v", err)
	}

	// Execute merge
	opts := MergeOptions{FogitDir: fogitDir}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Verify all features were closed
	if len(result.ClosedFeatures) != 3 {
		t.Errorf("Expected 3 closed features, got %d", len(result.ClosedFeatures))
	}

	// Verify each feature is closed
	for _, f := range []*fogit.Feature{feature1, feature2, feature3} {
		closed, err := repo.Get(ctx, f.ID)
		if err != nil {
			t.Fatalf("Failed to get feature %s: %v", f.Name, err)
		}
		if closed.DeriveState() != fogit.StateClosed {
			t.Errorf("Feature %s expected closed, got %s", f.Name, closed.DeriveState())
		}
	}
}

func TestMerge_ClosesSpecificFeature(t *testing.T) {
	_, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create features
	feature1 := fogit.NewFeature("Feature 1")
	if err := repo.Create(ctx, feature1); err != nil {
		t.Fatalf("Failed to create feature1: %v", err)
	}

	feature2 := fogit.NewFeature("Feature 2")
	if err := repo.Create(ctx, feature2); err != nil {
		t.Fatalf("Failed to create feature2: %v", err)
	}

	// Commit the feature files so merge doesn't fail on uncommitted changes
	if _, err := gitRepo.Commit("Add feature files", nil); err != nil {
		t.Fatalf("Failed to commit features: %v", err)
	}

	// Close only feature1 by ID
	opts := MergeOptions{
		FogitDir:    fogitDir,
		FeatureName: feature1.ID,
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Verify only feature1 was closed
	if len(result.ClosedFeatures) != 1 {
		t.Errorf("Expected 1 closed feature, got %d", len(result.ClosedFeatures))
	}

	// Feature1 should be closed
	closed1, _ := repo.Get(ctx, feature1.ID)
	if closed1.DeriveState() != fogit.StateClosed {
		t.Error("Feature 1 should be closed")
	}

	// Feature2 should still be open
	open2, _ := repo.Get(ctx, feature2.ID)
	if open2.DeriveState() != fogit.StateOpen {
		t.Error("Feature 2 should still be open")
	}
}

func TestMerge_NoOpenFeatures(t *testing.T) {
	_, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Don't create any features

	opts := MergeOptions{FogitDir: fogitDir}

	_, err := Merge(ctx, repo, gitRepo, opts)
	if err == nil {
		t.Error("Expected error when no features exist")
	}
}

func TestMerge_FailsWithUncommittedChanges(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Get current branch
	branch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get branch: %v", err)
	}

	// Create a feature
	feature := fogit.NewFeature("Test Feature")
	feature.Metadata["branch"] = branch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create uncommitted change (untracked file counts as a change)
	newFile := filepath.Join(tempDir, "uncommitted.txt")
	if err := os.WriteFile(newFile, []byte("uncommitted"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Try to merge - should fail due to uncommitted changes
	opts := MergeOptions{FogitDir: fogitDir}

	_, err = Merge(ctx, repo, gitRepo, opts)
	if err == nil {
		t.Error("Expected error when uncommitted changes exist")
	}
}

func TestMerge_DetectsMainBranch(t *testing.T) {
	_, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a feature (will be on master/main after initial commit)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file so merge doesn't fail on uncommitted changes
	if _, err := gitRepo.Commit("Add feature file", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	opts := MergeOptions{FogitDir: fogitDir}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Initial branch is typically "master" in go-git
	// IsMainBranch should be true for master/main/trunk
	if result.Branch != "master" && result.Branch != "main" {
		t.Logf("Branch is %s (not main/master)", result.Branch)
	}
}

func TestMerge_PreservesNoDeleteFlag(t *testing.T) {
	_, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file so merge doesn't fail on uncommitted changes
	if _, err := gitRepo.Commit("Add feature file", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	opts := MergeOptions{
		FogitDir: fogitDir,
		NoDelete: true,
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if !result.NoDelete {
		t.Error("Expected NoDelete to be preserved in result")
	}
}

func TestIsMainBranch(t *testing.T) {
	tests := []struct {
		branch   string
		expected bool
	}{
		{"main", true},
		{"master", true},
		{"trunk", true},
		{"develop", false},
		{"feature/test", false},
		{"release/v1.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			result := isMainBranch(tt.branch)
			if result != tt.expected {
				t.Errorf("isMainBranch(%q) = %v, want %v", tt.branch, result, tt.expected)
			}
		})
	}
}

func TestFindMostRecentFeature(t *testing.T) {
	// Create features with different timestamps
	now := time.Now()

	f1 := fogit.NewFeature("Feature 1")
	f1.GetCurrentVersion().ModifiedAt = now.Add(-2 * time.Hour)

	f2 := fogit.NewFeature("Feature 2")
	f2.GetCurrentVersion().ModifiedAt = now.Add(-1 * time.Hour)

	f3 := fogit.NewFeature("Feature 3")
	f3.GetCurrentVersion().ModifiedAt = now

	features := []*fogit.Feature{f1, f2, f3}

	result := findMostRecentFeature(features)
	if result != f3 {
		t.Errorf("Expected most recent feature (f3), got %s", result.Name)
	}

	// Test with nil slice
	nilResult := findMostRecentFeature(nil)
	if nilResult != nil {
		t.Error("Expected nil for empty slice")
	}

	// Test with empty slice
	emptyResult := findMostRecentFeature([]*fogit.Feature{})
	if emptyResult != nil {
		t.Error("Expected nil for empty slice")
	}
}

// TestMerge_BranchPerFeatureMode tests actual Git merge in branch-per-feature mode
func TestMerge_BranchPerFeatureMode(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create and checkout a feature branch
	featureBranch := "feature/test-feature"
	if err := gitRepo.CreateBranch(featureBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Create a feature on this branch
	feature := fogit.NewFeature("Test Feature")
	feature.Metadata["branch"] = featureBranch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file
	if _, err := gitRepo.Commit("Add feature metadata", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	// Create a file change on the feature branch
	testFile := filepath.Join(tempDir, "feature.txt")
	if err := os.WriteFile(testFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Commit the changes
	if _, err := gitRepo.Commit("Add feature work", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Execute merge (should merge to master)
	opts := MergeOptions{
		FogitDir:   fogitDir,
		BaseBranch: "master", // go-git defaults to master
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Verify results
	if !result.MergePerformed {
		t.Error("Expected MergePerformed to be true")
	}
	if result.Branch != featureBranch {
		t.Errorf("Expected branch %s, got %s", featureBranch, result.Branch)
	}
	if result.BaseBranch != "master" {
		t.Errorf("Expected base branch master, got %s", result.BaseBranch)
	}
	if result.IsMainBranch {
		t.Error("Expected IsMainBranch to be false for feature branch")
	}

	// Verify we're now on master
	currentBranch, err := gitRepo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}
	if currentBranch != "master" {
		t.Errorf("Expected to be on master, got %s", currentBranch)
	}

	// Verify the file exists on master after merge
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Expected feature.txt to exist after merge")
	}

	// Verify feature is closed
	closedFeature, err := repo.Get(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Failed to get feature: %v", err)
	}
	if closedFeature.DeriveState() != fogit.StateClosed {
		t.Errorf("Expected feature to be closed, got %s", closedFeature.DeriveState())
	}
}

// TestMerge_BranchDeleted tests that feature branch is deleted after merge
func TestMerge_BranchDeleted(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create and checkout a feature branch
	featureBranch := "feature/to-delete"
	if err := gitRepo.CreateBranch(featureBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Create a feature
	feature := fogit.NewFeature("Feature to Delete")
	feature.Metadata["branch"] = featureBranch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file first
	if _, err := gitRepo.Commit("Add feature metadata", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	// Create a file and commit
	testFile := filepath.Join(tempDir, "delete-test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if _, err := gitRepo.Commit("Add test file", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Execute merge without --no-delete
	opts := MergeOptions{
		FogitDir:   fogitDir,
		BaseBranch: "master",
		NoDelete:   false,
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if !result.BranchDeleted {
		t.Error("Expected BranchDeleted to be true")
	}
}

// TestMerge_NoDeletePreservesBranch tests that --no-delete keeps the branch
func TestMerge_NoDeletePreservesBranch(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create and checkout a feature branch
	featureBranch := "feature/keep-branch"
	if err := gitRepo.CreateBranch(featureBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Create a feature
	feature := fogit.NewFeature("Keep Branch Feature")
	feature.Metadata["branch"] = featureBranch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file first
	if _, err := gitRepo.Commit("Add feature metadata", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	// Create a file and commit
	testFile := filepath.Join(tempDir, "keep-test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if _, err := gitRepo.Commit("Add test file", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Execute merge with --no-delete
	opts := MergeOptions{
		FogitDir:   fogitDir,
		BaseBranch: "master",
		NoDelete:   true,
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if result.BranchDeleted {
		t.Error("Expected BranchDeleted to be false with NoDelete=true")
	}
	if !result.NoDelete {
		t.Error("Expected NoDelete flag to be preserved")
	}
}

// TestMerge_TrunkBasedMode tests that no Git merge happens on main branch
func TestMerge_TrunkBasedMode(t *testing.T) {
	_, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// We're on master (trunk-based mode)
	branch, _ := gitRepo.GetCurrentBranch()

	// Create a feature on master
	feature := fogit.NewFeature("Trunk Feature")
	feature.Metadata["branch"] = branch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file
	if _, err := gitRepo.Commit("Add feature", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	opts := MergeOptions{FogitDir: fogitDir}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// In trunk-based mode, no merge should be performed
	if result.MergePerformed {
		t.Error("Expected MergePerformed to be false in trunk-based mode")
	}
	if result.BranchDeleted {
		t.Error("Expected BranchDeleted to be false in trunk-based mode")
	}
	if !result.IsMainBranch {
		t.Error("Expected IsMainBranch to be true")
	}

	// Feature should still be closed
	closedFeature, _ := repo.Get(ctx, feature.ID)
	if closedFeature.DeriveState() != fogit.StateClosed {
		t.Error("Expected feature to be closed")
	}
}

// TestMerge_CustomBaseBranch tests merging to a custom base branch
func TestMerge_CustomBaseBranch(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a develop branch from master
	if err := gitRepo.CreateBranch("develop"); err != nil {
		t.Fatalf("Failed to create develop branch: %v", err)
	}

	// Create and checkout a feature branch
	featureBranch := "feature/custom-base"
	if err := gitRepo.CreateBranch(featureBranch); err != nil {
		t.Fatalf("Failed to create feature branch: %v", err)
	}
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}

	// Create a feature
	feature := fogit.NewFeature("Custom Base Feature")
	feature.Metadata["branch"] = featureBranch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file first
	if _, err := gitRepo.Commit("Add feature metadata", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	// Create a file and commit
	testFile := filepath.Join(tempDir, "custom-base.txt")
	if err := os.WriteFile(testFile, []byte("custom"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if _, err := gitRepo.Commit("Add test file", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Merge to develop instead of master
	opts := MergeOptions{
		FogitDir:   fogitDir,
		BaseBranch: "develop",
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if result.BaseBranch != "develop" {
		t.Errorf("Expected base branch develop, got %s", result.BaseBranch)
	}

	// Verify we're on develop
	currentBranch, _ := gitRepo.GetCurrentBranch()
	if currentBranch != "develop" {
		t.Errorf("Expected to be on develop, got %s", currentBranch)
	}
}

// TestMerge_SquashOption tests the squash merge option
func TestMerge_SquashOption(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create and checkout a feature branch
	featureBranch := "feature/squash-test"
	if err := gitRepo.CreateBranch(featureBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Create a feature
	feature := fogit.NewFeature("Squash Feature")
	feature.Metadata["branch"] = featureBranch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file first
	if _, err := gitRepo.Commit("Add feature metadata", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	// Create a file and commit
	testFile := filepath.Join(tempDir, "squash.txt")
	if err := os.WriteFile(testFile, []byte("squash"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if _, err := gitRepo.Commit("Add squash file", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Execute squash merge
	opts := MergeOptions{
		FogitDir:   fogitDir,
		BaseBranch: "master",
		Squash:     true,
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if !result.MergePerformed {
		t.Error("Expected MergePerformed to be true")
	}

	// Note: Squash merge stages changes but doesn't auto-commit
	// The actual verification of squash behavior would require
	// checking git status or commit history
}

// TestMerge_DefaultBaseBranch tests that base branch defaults to "main"
func TestMerge_DefaultBaseBranch(t *testing.T) {
	_, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a feature on master
	feature := fogit.NewFeature("Default Base Test")
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature
	if _, err := gitRepo.Commit("Add feature", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	opts := MergeOptions{
		FogitDir: fogitDir,
		// BaseBranch not set - should default to "main"
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if result.BaseBranch != "main" {
		t.Errorf("Expected default base branch 'main', got %s", result.BaseBranch)
	}
}

// TestMerge_ConflictRecovery tests that merge conflict triggers conflict detection and state saving
func TestMerge_ConflictRecovery(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a file on master that will conflict
	conflictFile := filepath.Join(tempDir, "conflict.txt")
	if err := os.WriteFile(conflictFile, []byte("master content"), 0644); err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}
	if _, err := gitRepo.Commit("Add conflict file on master", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create and checkout a feature branch
	featureBranch := "feature/conflict-test"
	if err := gitRepo.CreateBranch(featureBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Modify the same file differently on feature branch
	if err := os.WriteFile(conflictFile, []byte("feature content - different"), 0644); err != nil {
		t.Fatalf("Failed to modify conflict file: %v", err)
	}

	// Create a feature
	feature := fogit.NewFeature("Conflict Feature")
	feature.Metadata["branch"] = featureBranch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature and conflict file changes
	if _, err := gitRepo.Commit("Add feature and change conflict file", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Go back to master and make a conflicting change
	if err := gitRepo.CheckoutBranch("master"); err != nil {
		t.Fatalf("Failed to checkout master: %v", err)
	}
	if err := os.WriteFile(conflictFile, []byte("master content - also different"), 0644); err != nil {
		t.Fatalf("Failed to modify conflict file on master: %v", err)
	}
	if _, err := gitRepo.Commit("Change conflict file on master", nil); err != nil {
		t.Fatalf("Failed to commit on master: %v", err)
	}

	// Go back to feature branch
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}

	// Execute merge - should detect conflict and return ConflictDetected=true
	opts := MergeOptions{
		FogitDir:   fogitDir,
		BaseBranch: "master",
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Expected no error on conflict (conflict is communicated via result): %v", err)
	}

	// Verify conflict was detected
	if !result.ConflictDetected {
		t.Error("Expected ConflictDetected to be true")
	}

	// Verify merge state was saved
	if !HasMergeState(fogitDir) {
		t.Error("Expected merge state to be saved")
	}

	// Clean up: abort the merge so other tests don't fail
	abortOpts := MergeOptions{
		FogitDir: fogitDir,
		Abort:    true,
	}
	Merge(ctx, repo, gitRepo, abortOpts)
}

// TestMerge_FastForwardMerge tests that a fast-forward merge works correctly
func TestMerge_FastForwardMerge(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create and checkout a feature branch
	featureBranch := "feature/fast-forward"
	if err := gitRepo.CreateBranch(featureBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Create a feature
	feature := fogit.NewFeature("Fast Forward Feature")
	feature.Metadata["branch"] = featureBranch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature file
	if _, err := gitRepo.Commit("Add feature metadata", nil); err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	// Create additional file changes
	testFile := filepath.Join(tempDir, "fastforward.txt")
	if err := os.WriteFile(testFile, []byte("fast forward content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if _, err := gitRepo.Commit("Add feature work", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Execute merge - should be a fast-forward since master hasn't changed
	opts := MergeOptions{
		FogitDir:   fogitDir,
		BaseBranch: "master",
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Verify merge completed
	if !result.MergePerformed {
		t.Error("Expected MergePerformed to be true")
	}

	// Verify we're on master
	currentBranch, _ := gitRepo.GetCurrentBranch()
	if currentBranch != "master" {
		t.Errorf("Expected to be on master, got %s", currentBranch)
	}

	// Verify the file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Expected fastforward.txt to exist after merge")
	}

	// Verify feature is closed
	closedFeature, _ := repo.Get(ctx, feature.ID)
	if closedFeature.DeriveState() != fogit.StateClosed {
		t.Error("Expected feature to be closed")
	}
}

// TestMerge_ConflictResolutionWorkflow tests the full conflict resolution workflow:
// 1. Create conflict situation
// 2. Attempt merge (detects conflict, saves state, returns ConflictDetected)
// 3. User resolves conflict by editing the file
// 4. Call merge --continue to complete
func TestMerge_ConflictResolutionWorkflow(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a file on master that will conflict
	conflictFile := filepath.Join(tempDir, "conflict.txt")
	if err := os.WriteFile(conflictFile, []byte("master content\nline 2\nline 3"), 0644); err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}
	if _, err := gitRepo.Commit("Add conflict file on master", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create and checkout a feature branch
	featureBranch := "feature/conflict-resolution"
	if err := gitRepo.CreateBranch(featureBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Modify the same file differently on feature branch
	if err := os.WriteFile(conflictFile, []byte("feature content\nline 2\nline 3"), 0644); err != nil {
		t.Fatalf("Failed to modify conflict file: %v", err)
	}

	// Create a feature
	feature := fogit.NewFeature("Conflict Resolution Feature")
	feature.Metadata["branch"] = featureBranch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit the feature and conflict file changes
	if _, err := gitRepo.Commit("Add feature and change conflict file", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Go back to master and make a conflicting change
	if err := gitRepo.CheckoutBranch("master"); err != nil {
		t.Fatalf("Failed to checkout master: %v", err)
	}
	if err := os.WriteFile(conflictFile, []byte("master updated content\nline 2\nline 3"), 0644); err != nil {
		t.Fatalf("Failed to modify conflict file on master: %v", err)
	}
	if _, err := gitRepo.Commit("Change conflict file on master", nil); err != nil {
		t.Fatalf("Failed to commit on master: %v", err)
	}

	// Go back to feature branch
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout feature branch: %v", err)
	}

	// Step 1: Attempt merge - should detect conflict
	opts := MergeOptions{
		FogitDir:   fogitDir,
		BaseBranch: "master",
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Expected no error, conflict should be in result: %v", err)
	}

	if !result.ConflictDetected {
		t.Fatal("Expected ConflictDetected to be true")
	}

	// Verify merge state was saved
	if !HasMergeState(fogitDir) {
		t.Error("Expected merge state to be saved")
	}

	// Verify we're on master (git checkout happened before merge)
	currentBranch, _ := gitRepo.GetCurrentBranch()
	if currentBranch != "master" {
		t.Logf("Note: On branch %s, expected master", currentBranch)
	}

	// Step 2: Resolve the conflict by editing the file (simulating user action)
	resolvedContent := "resolved: combined content\nline 2\nline 3"
	if err := os.WriteFile(conflictFile, []byte(resolvedContent), 0644); err != nil {
		t.Fatalf("Failed to write resolved content: %v", err)
	}

	// Stage the resolved file using git add (don't commit yet - let --continue handle that)
	stageCmd := exec.Command("git", "add", "conflict.txt")
	stageCmd.Dir = tempDir
	if out, err := stageCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to stage resolved file: %v\n%s", err, string(out))
	}

	// Step 3: Continue the merge using --continue
	continueOpts := MergeOptions{
		FogitDir: fogitDir,
		Continue: true,
	}

	continueResult, err := Merge(ctx, repo, gitRepo, continueOpts)
	if err != nil {
		t.Fatalf("Continue failed: %v", err)
	}

	// Verify continue succeeded
	if continueResult.ConflictDetected {
		t.Error("Expected no conflict after continue")
	}

	// Verify merge state was cleared
	if HasMergeState(fogitDir) {
		t.Error("Expected merge state to be cleared after continue")
	}

	// Verify we're on master
	currentBranch, _ = gitRepo.GetCurrentBranch()
	if currentBranch != "master" {
		t.Errorf("Expected to be on master after continue, got %s", currentBranch)
	}

	// Verify feature is closed
	finalFeature, _ := repo.Get(ctx, feature.ID)
	if finalFeature.DeriveState() != fogit.StateClosed {
		t.Error("Expected feature to be closed after continue")
	}
}

// TestMerge_AbortWorkflow tests the --abort flag
func TestMerge_AbortWorkflow(t *testing.T) {
	tempDir, fogitDir, gitRepo, repo, cleanup := setupMergeTestGitRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create a file on master that will conflict
	conflictFile := filepath.Join(tempDir, "conflict.txt")
	if err := os.WriteFile(conflictFile, []byte("master content"), 0644); err != nil {
		t.Fatalf("Failed to create conflict file: %v", err)
	}
	if _, err := gitRepo.Commit("Add conflict file on master", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create and checkout a feature branch
	featureBranch := "feature/abort-test"
	if err := gitRepo.CreateBranch(featureBranch); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout branch: %v", err)
	}

	// Modify the same file differently
	if err := os.WriteFile(conflictFile, []byte("feature content"), 0644); err != nil {
		t.Fatalf("Failed to modify conflict file: %v", err)
	}

	// Create a feature
	feature := fogit.NewFeature("Abort Test Feature")
	feature.Metadata["branch"] = featureBranch
	if err := repo.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Commit changes
	if _, err := gitRepo.Commit("Feature changes", nil); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create conflict on master
	if err := gitRepo.CheckoutBranch("master"); err != nil {
		t.Fatalf("Failed to checkout master: %v", err)
	}
	if err := os.WriteFile(conflictFile, []byte("different master content"), 0644); err != nil {
		t.Fatalf("Failed to modify on master: %v", err)
	}
	if _, err := gitRepo.Commit("Master changes", nil); err != nil {
		t.Fatalf("Failed to commit on master: %v", err)
	}

	// Go back to feature branch
	if err := gitRepo.CheckoutBranch(featureBranch); err != nil {
		t.Fatalf("Failed to checkout feature: %v", err)
	}

	// Attempt merge - should detect conflict
	opts := MergeOptions{
		FogitDir:   fogitDir,
		BaseBranch: "master",
	}

	result, err := Merge(ctx, repo, gitRepo, opts)
	if err != nil {
		t.Fatalf("Expected no error: %v", err)
	}
	if !result.ConflictDetected {
		t.Fatal("Expected conflict")
	}

	// Abort the merge
	abortOpts := MergeOptions{
		FogitDir: fogitDir,
		Abort:    true,
	}

	abortResult, err := Merge(ctx, repo, gitRepo, abortOpts)
	if err != nil {
		t.Fatalf("Abort failed: %v", err)
	}

	if !abortResult.Aborted {
		t.Error("Expected Aborted to be true")
	}

	// Verify merge state was cleared
	if HasMergeState(fogitDir) {
		t.Error("Expected merge state to be cleared after abort")
	}

	// Verify we're back on feature branch
	currentBranch, _ := gitRepo.GetCurrentBranch()
	if currentBranch != featureBranch {
		t.Errorf("Expected to be on %s after abort, got %s", featureBranch, currentBranch)
	}

	// Verify feature was reopened
	reopenedFeature, _ := repo.Get(ctx, feature.ID)
	if reopenedFeature.DeriveState() == fogit.StateClosed {
		t.Error("Expected feature to be reopened after abort")
	}
}

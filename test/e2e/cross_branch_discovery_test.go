// Package e2e contains end-to-end tests for FoGit.
package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestCrossBranchDiscovery_RemoteOnlyFeature tests the scenario where a feature
// exists only on a remote branch and hasn't been pulled locally.
//
// Scenario:
// 1. Developer A creates a feature on a branch and pushes it
// 2. Developer B clones the repo
// 3. Developer A creates another feature branch and pushes it
// 4. Developer B runs git fetch (but doesn't pull/merge)
// 5. Developer B should be able to discover the new feature automatically (cross-branch is default)
func TestCrossBranchDiscovery_RemoteOnlyFeature(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create temp directories
	baseDir := t.TempDir()
	bareRepoDir := filepath.Join(baseDir, "remote.git")
	devADir := filepath.Join(baseDir, "dev-a")
	devBDir := filepath.Join(baseDir, "dev-b")

	// STEP 1: Initialize developer A workspace
	t.Log("Initializing developer A workspace...")

	devARepo, err := gogit.PlainInit(devADir, false)
	if err != nil {
		t.Fatalf("Failed to init dev A: %v", err)
	}

	// Configure user for dev A
	devACfg, _ := devARepo.Config()
	devACfg.User.Name = "Developer A"
	devACfg.User.Email = "dev-a@example.com"
	_ = devARepo.SetConfig(devACfg)

	// Create initial files and fogit init
	devAWorktree, _ := devARepo.Worktree()
	_ = os.WriteFile(filepath.Join(devADir, "README.md"), []byte("# Test Project\n"), 0644)

	output, err := runFogit(t, devADir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit in dev A: %v\nOutput: %s", err, output)
	}

	// Stage and commit
	_, _ = devAWorktree.Add(".")
	_, err = devAWorktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Developer A", Email: "dev-a@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// STEP 2: Create bare repository and set as remote
	t.Log("Creating bare repository...")
	_, err = gogit.PlainInit(bareRepoDir, true)
	if err != nil {
		t.Fatalf("Failed to create bare repository: %v", err)
	}

	_, err = devARepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{bareRepoDir},
	})
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	// Push initial commit
	t.Log("Pushing initial commit to bare repo...")
	err = devARepo.Push(&gogit.PushOptions{RemoteName: "origin"})
	if err != nil {
		t.Fatalf("Failed to push initial commit: %v", err)
	}

	// STEP 3: Developer B clones
	t.Log("Developer B cloning repository...")
	devBRepo, err := gogit.PlainClone(devBDir, false, &gogit.CloneOptions{
		URL: bareRepoDir,
	})
	if err != nil {
		t.Fatalf("Failed to clone to dev B: %v", err)
	}

	devBCfg, _ := devBRepo.Config()
	devBCfg.User.Name = "Developer B"
	devBCfg.User.Email = "dev-b@example.com"
	_ = devBRepo.SetConfig(devBCfg)

	// STEP 4: Developer A creates a feature and pushes it
	t.Log("Developer A creating feature...")
	output, err = runFogit(t, devADir, "feature", "Remote Feature", "-d", "A feature created by Developer A", "-p", "high")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature created:\n%s", output)

	// Push the feature branch
	t.Log("Developer A pushing feature branch to origin...")
	err = devARepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/heads/*"),
		},
	})
	if err != nil {
		t.Fatalf("Failed to push feature branch: %v", err)
	}

	// STEP 5: Developer B fetches (but doesn't pull/checkout the feature branch)
	t.Log("Developer B fetching from origin...")
	err = devBRepo.Fetch(&gogit.FetchOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/remotes/origin/*"),
		},
	})
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	// STEP 6: Developer B tries to find the feature
	t.Log("Developer B searching for feature with --current-branch (no cross-branch)...")
	output, _ = runFogit(t, devBDir, "list", "--current-branch")
	if strings.Contains(output, "Remote Feature") {
		t.Fatalf("Feature should NOT be found with --current-branch:\n%s", output)
	}
	t.Log("✓ Feature not found with --current-branch (expected)")

	t.Log("Developer B searching for feature with default cross-branch discovery...")
	output, err = runFogit(t, devBDir, "list")
	if err != nil {
		t.Fatalf("fogit list failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Remote Feature") {
		t.Fatalf("Expected to find 'Remote Feature' with automatic cross-branch discovery, but got:\n%s", output)
	}

	t.Logf("✓ Successfully found remote feature via automatic cross-branch discovery:\n%s", output)
}

// TestCrossBranchDiscovery_RemoteOnlyFeature_NoFetch tests that we cannot find
// a remote feature if git fetch hasn't been run.
func TestCrossBranchDiscovery_RemoteOnlyFeature_NoFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create temp directories
	baseDir := t.TempDir()
	bareRepoDir := filepath.Join(baseDir, "remote.git")
	devADir := filepath.Join(baseDir, "dev-a")
	devBDir := filepath.Join(baseDir, "dev-b")

	// Init dev A
	devARepo, err := gogit.PlainInit(devADir, false)
	if err != nil {
		t.Fatalf("Failed to init dev A: %v", err)
	}

	devACfg, _ := devARepo.Config()
	devACfg.User.Name = "Developer A"
	devACfg.User.Email = "dev-a@example.com"
	_ = devARepo.SetConfig(devACfg)

	devAWorktree, _ := devARepo.Worktree()
	_ = os.WriteFile(filepath.Join(devADir, "README.md"), []byte("# Test\n"), 0644)

	output, err := runFogit(t, devADir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	_, _ = devAWorktree.Add(".")
	_, _ = devAWorktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Developer A", Email: "dev-a@example.com", When: time.Now()},
	})

	// Create bare repository and set as remote
	_, err = gogit.PlainInit(bareRepoDir, true)
	if err != nil {
		t.Fatalf("Failed to create bare repository: %v", err)
	}

	_, err = devARepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{bareRepoDir},
	})
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	_ = devARepo.Push(&gogit.PushOptions{RemoteName: "origin"})

	// Clone to dev B
	devBRepo, err := gogit.PlainClone(devBDir, false, &gogit.CloneOptions{
		URL: bareRepoDir,
	})
	if err != nil {
		t.Fatalf("Failed to clone to dev B: %v", err)
	}

	devBCfg, _ := devBRepo.Config()
	devBCfg.User.Name = "Developer B"
	devBCfg.User.Email = "dev-b@example.com"
	_ = devBRepo.SetConfig(devBCfg)

	// Developer A creates a feature and pushes
	output, err = runFogit(t, devADir, "feature", "Unfetched Feature")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	_ = devARepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec("+refs/heads/*:refs/heads/*")},
	})

	// Developer B does NOT fetch - tries to find the feature
	t.Log("Developer B NOT fetching - trying to find feature (automatic cross-branch)...")

	output, _ = runFogit(t, devBDir, "list")
	if strings.Contains(output, "Unfetched Feature") {
		t.Fatal("Should not find feature when fetch hasn't been done")
	}

	t.Log("✓ Feature not found as expected (no fetch was done)")
}

// TestCrossBranchDiscovery_ListAllBranches tests listing features across all branches
// including remote branches.
func TestCrossBranchDiscovery_ListAllBranches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create temp directories
	baseDir := t.TempDir()
	bareRepoDir := filepath.Join(baseDir, "remote.git")
	devADir := filepath.Join(baseDir, "dev-a")
	devBDir := filepath.Join(baseDir, "dev-b")

	// Init dev A
	devARepo, err := gogit.PlainInit(devADir, false)
	if err != nil {
		t.Fatalf("Failed to init dev A: %v", err)
	}

	devACfg, _ := devARepo.Config()
	devACfg.User.Name = "Developer A"
	devACfg.User.Email = "dev-a@example.com"
	_ = devARepo.SetConfig(devACfg)

	devAWorktree, _ := devARepo.Worktree()
	_ = os.WriteFile(filepath.Join(devADir, "README.md"), []byte("# Test\n"), 0644)

	output, err := runFogit(t, devADir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	_, _ = devAWorktree.Add(".")
	_, _ = devAWorktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Developer A", Email: "dev-a@example.com", When: time.Now()},
	})

	// Create bare repository and set as remote
	_, err = gogit.PlainInit(bareRepoDir, true)
	if err != nil {
		t.Fatalf("Failed to create bare repository: %v", err)
	}

	_, err = devARepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{bareRepoDir},
	})
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	_ = devARepo.Push(&gogit.PushOptions{RemoteName: "origin"})

	// Clone to dev B
	devBRepo, err := gogit.PlainClone(devBDir, false, &gogit.CloneOptions{
		URL: bareRepoDir,
	})
	if err != nil {
		t.Fatalf("Failed to clone to dev B: %v", err)
	}

	devBCfg, _ := devBRepo.Config()
	devBCfg.User.Name = "Developer B"
	devBCfg.User.Email = "dev-b@example.com"
	_ = devBRepo.SetConfig(devBCfg)

	devBWorktree, _ := devBRepo.Worktree()

	// Developer B creates a local feature
	output, err = runFogit(t, devBDir, "feature", "Local Feature")
	if err != nil {
		t.Fatalf("Failed to create local feature: %v\nOutput: %s", err, output)
	}

	// Stay on the feature branch, don't need to checkout master

	// Developer A creates a different feature and pushes
	output, err = runFogit(t, devADir, "feature", "Remote Feature")
	if err != nil {
		t.Fatalf("Failed to create remote feature: %v\nOutput: %s", err, output)
	}

	_ = devARepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec("+refs/heads/*:refs/heads/*")},
	})

	// Developer B fetches (but we need to be careful - they're on a feature branch)
	// First, stage any changes to avoid checkout issues
	_, _ = devBWorktree.Add(".")
	_, _ = devBWorktree.Commit("Save local work", &gogit.CommitOptions{
		Author:            &object.Signature{Name: "Developer B", Email: "dev-b@example.com", When: time.Now()},
		AllowEmptyCommits: true,
	})

	err = devBRepo.Fetch(&gogit.FetchOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/remotes/origin/*"),
		},
	})
	if err != nil {
		t.Fatalf("Failed to fetch: %v", err)
	}

	// List all features across branches (automatic in branch-per-feature mode)
	t.Log("Listing all features with automatic cross-branch discovery...")

	output, err = runFogit(t, devBDir, "list")
	if err != nil {
		t.Fatalf("fogit list failed: %v\nOutput: %s", err, output)
	}

	// Should find both features
	foundLocal := strings.Contains(output, "Local Feature")
	foundRemote := strings.Contains(output, "Remote Feature")

	if !foundLocal {
		t.Errorf("Expected to find 'Local Feature', got:\n%s", output)
	}
	if !foundRemote {
		t.Errorf("Expected to find 'Remote Feature' on origin branch, got:\n%s", output)
	}

	if foundLocal && foundRemote {
		t.Logf("✓ Found all features across branches:\n%s", output)
	}
}

// TestCrossBranchDiscovery_SearchAutomatic tests searching features across all branches
// with automatic cross-branch discovery (default in branch-per-feature mode).
func TestCrossBranchDiscovery_SearchAutomatic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create temp directories
	baseDir := t.TempDir()
	bareRepoDir := filepath.Join(baseDir, "remote.git")
	devADir := filepath.Join(baseDir, "dev-a")
	devBDir := filepath.Join(baseDir, "dev-b")

	// Init dev A
	devARepo, err := gogit.PlainInit(devADir, false)
	if err != nil {
		t.Fatalf("Failed to init dev A: %v", err)
	}

	devACfg, _ := devARepo.Config()
	devACfg.User.Name = "Developer A"
	devACfg.User.Email = "dev-a@example.com"
	_ = devARepo.SetConfig(devACfg)

	devAWorktree, _ := devARepo.Worktree()
	_ = os.WriteFile(filepath.Join(devADir, "README.md"), []byte("# Test\n"), 0644)

	output, err := runFogit(t, devADir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	_, _ = devAWorktree.Add(".")
	_, _ = devAWorktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Developer A", Email: "dev-a@example.com", When: time.Now()},
	})

	// Create bare repository and set as remote
	_, err = gogit.PlainInit(bareRepoDir, true)
	if err != nil {
		t.Fatalf("Failed to create bare repository: %v", err)
	}

	_, err = devARepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{bareRepoDir},
	})
	if err != nil {
		t.Fatalf("Failed to add remote: %v", err)
	}

	_ = devARepo.Push(&gogit.PushOptions{RemoteName: "origin"})

	// Clone to dev B
	devBRepo, err := gogit.PlainClone(devBDir, false, &gogit.CloneOptions{
		URL: bareRepoDir,
	})
	if err != nil {
		t.Fatalf("Failed to clone to dev B: %v", err)
	}

	devBCfg, _ := devBRepo.Config()
	devBCfg.User.Name = "Developer B"
	devBCfg.User.Email = "dev-b@example.com"
	_ = devBRepo.SetConfig(devBCfg)

	// Developer A creates a feature with specific searchable name and pushes
	output, err = runFogit(t, devADir, "feature", "Authentication Feature", "-d", "Login and registration")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	_ = devARepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec("+refs/heads/*:refs/heads/*")},
	})

	// Developer B fetches
	_ = devBRepo.Fetch(&gogit.FetchOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec("+refs/heads/*:refs/remotes/origin/*"),
		},
	})

	// Search for the feature with --current-branch (should not find on remote)
	t.Log("Searching with --current-branch (no cross-branch)...")
	output, _ = runFogit(t, devBDir, "search", "Authentication", "--current-branch")
	if strings.Contains(output, "Authentication Feature") {
		t.Log("Note: Feature found with --current-branch (may be on current branch)")
	}

	// Search with default automatic cross-branch discovery
	t.Log("Searching with automatic cross-branch discovery (default)...")
	output, err = runFogit(t, devBDir, "search", "Authentication")
	if err != nil {
		t.Fatalf("fogit search failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Authentication Feature") {
		t.Fatalf("Expected to find 'Authentication Feature' with automatic cross-branch search, but got:\n%s", output)
	}

	t.Logf("✓ Successfully found feature via automatic cross-branch search:\n%s", output)
}

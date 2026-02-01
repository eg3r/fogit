package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestCrossBranch_LinkSourceNotOnCurrentBranch tests the scenario from bug report:
// BUG: `fogit link` fails when source feature is not on current branch
//
// Steps to Reproduce:
// 1. Initialize git + fogit repo
// 2. Create feature "user-auth" (creates branch feature/user-auth)
// 3. Return to master
// 4. Create feature "api-gateway" (creates branch feature/api-gateway)
// 5. Return to master
// 6. Create feature "logging" (creates branch feature/logging)
// 7. Switch to user-auth feature
// 8. fogit list - shows all 3 features
// 9. fogit link api-gateway logging depends-on - FAILS because api-gateway's YAML is NOT on current branch
//
// Expected: Link should succeed since fogit list shows all features exist.
// Actual: Error: failed to save feature: not found
//
// Root Cause: link needs to save the relationship to the source feature's YAML file,
// but the source feature's YAML file only exists on its own branch, not the current branch.
func TestCrossBranch_LinkSourceNotOnCurrentBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir := t.TempDir()

	// Step 1: Initialize Git repository
	t.Log("Step 1: Initializing Git repository...")
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cfg, _ := gitRepo.Config()
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	_ = gitRepo.SetConfig(cfg)

	worktree, _ := gitRepo.Worktree()
	_ = os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Test Project\n"), 0644)
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Initialize fogit
	t.Log("Initializing fogit...")
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	// Disable fuzzy matching and set create_branch_from to current for test stability
	_, _ = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	_, _ = runFogit(t, projectDir, "config", "set", "workflow.create_branch_from", "current")

	// Commit config changes so they persist across branch switches
	_, _ = worktree.Add(".fogit/config.yml")
	_, _ = worktree.Commit("Configure fogit for testing", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Get the main branch name
	mainBranch := "master"
	if _, err := gitRepo.Reference(plumbing.NewBranchReferenceName("main"), false); err == nil {
		mainBranch = "main"
	}

	// Step 2: Create "user-auth" feature (creates its own branch)
	t.Log("Step 2: Creating 'user-auth' feature...")
	output, err = runFogit(t, projectDir, "create", "user-auth", "-d", "User authentication")
	if err != nil {
		t.Fatalf("Failed to create user-auth: %v\nOutput: %s", err, output)
	}
	t.Logf("user-auth created:\n%s", output)

	// Step 3: Return to main branch
	t.Log("Step 3: Returning to main branch...")
	err = worktree.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(mainBranch),
	})
	if err != nil {
		t.Fatalf("Failed to checkout %s: %v", mainBranch, err)
	}

	// Step 4: Create "api-gateway" feature (creates its own branch)
	t.Log("Step 4: Creating 'api-gateway' feature...")
	output, err = runFogit(t, projectDir, "create", "api-gateway", "-d", "API gateway")
	if err != nil {
		t.Fatalf("Failed to create api-gateway: %v\nOutput: %s", err, output)
	}
	t.Logf("api-gateway created:\n%s", output)

	// Step 5: Return to main branch
	t.Log("Step 5: Returning to main branch...")
	err = worktree.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(mainBranch),
	})
	if err != nil {
		t.Fatalf("Failed to checkout %s: %v", mainBranch, err)
	}

	// Step 6: Create "logging" feature (creates its own branch)
	t.Log("Step 6: Creating 'logging' feature...")
	output, err = runFogit(t, projectDir, "create", "logging", "-d", "Logging system")
	if err != nil {
		t.Fatalf("Failed to create logging: %v\nOutput: %s", err, output)
	}
	t.Logf("logging created:\n%s", output)

	// Step 7: Switch to user-auth feature
	t.Log("Step 7: Switching to user-auth feature...")
	output, err = runFogit(t, projectDir, "switch", "user-auth")
	if err != nil {
		t.Fatalf("Failed to switch to user-auth: %v\nOutput: %s", err, output)
	}
	t.Logf("Switched:\n%s", output)

	// Step 8: Verify fogit list shows all 3 features
	t.Log("Step 8: Verifying fogit list shows all features...")
	output, err = runFogit(t, projectDir, "list")
	if err != nil {
		t.Fatalf("fogit list failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "user-auth") {
		t.Fatalf("Expected 'user-auth' in list output, got:\n%s", output)
	}
	if !strings.Contains(output, "api-gateway") {
		t.Fatalf("Expected 'api-gateway' in list output, got:\n%s", output)
	}
	if !strings.Contains(output, "logging") {
		t.Fatalf("Expected 'logging' in list output, got:\n%s", output)
	}
	t.Logf("✓ All 3 features visible in list:\n%s", output)

	// Step 9a: Test link where source IS on current branch (should work)
	t.Log("Step 9a: Linking user-auth -> logging (source IS on current branch - should work)...")
	output, err = runFogit(t, projectDir, "link", "user-auth", "logging", "depends-on")
	if err != nil {
		t.Fatalf("Link failed when source IS on current branch: %v\nOutput: %s", err, output)
	}
	t.Logf("✓ Link succeeded when source IS on current branch:\n%s", output)

	// Step 9b: Test link where source is NOT on current branch (THIS IS THE BUG)
	t.Log("Step 9b: Linking api-gateway -> logging (source NOT on current branch - THE BUG)...")
	output, err = runFogit(t, projectDir, "link", "api-gateway", "logging", "depends-on")
	if err != nil {
		t.Fatalf("BUG CONFIRMED: Link failed when source NOT on current branch: %v\n"+
			"Output: %s\n\n"+
			"This is the bug: fogit link should save the relationship to the source feature's branch.\n"+
			"Currently it tries to save to the current branch where the source YAML doesn't exist.\n"+
			"Suggested fix: When saving relationship, detect if source is on different branch and\n"+
			"either checkout that branch temporarily, use git plumbing to write directly, or\n"+
			"give a clear error message.", err, output)
	}
	t.Logf("✓ Link succeeded even when source NOT on current branch:\n%s", output)

	// The relationship was saved on the api-gateway branch via cross-branch commit.
	// Verify the output contains the expected relationship info.
	if !strings.Contains(output, "api-gateway") || !strings.Contains(output, "logging") || !strings.Contains(output, "depends-on") {
		t.Fatalf("Expected relationship output to contain source, target, and type, got:\n%s", output)
	}
	t.Log("✓ Cross-branch link bug fix verified - source feature can be on a different branch")
}

// TestCrossBranch_LinkFeatureFromDifferentBranch tests the scenario where:
// 1. Developer creates feature A from main branch (creates branch feature/A, stores YAML there)
// 2. Developer returns to main branch
// 3. Developer creates feature B from main branch (creates branch feature/B, stores YAML there)
// 4. fogit list shows both features (cross-branch discovery works)
// 5. Developer tries to link feature B -> feature A while on feature/B branch
//
// Expected: Link should succeed since fogit list shows both features exist.
// Bug: Link fails because it only looks at current branch's .fogit/features/ directory.
//
// Per spec/specification/07-git-integration.md#cross-branch-feature-discovery:
// "Creating relationships between features on different branches" MUST use cross-branch discovery
func TestCrossBranch_LinkFeatureFromDifferentBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create temp project directory
	projectDir := t.TempDir()

	// Initialize Git repository
	t.Log("Step 1: Initializing Git repository...")
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user
	cfg, err := gitRepo.Config()
	if err != nil {
		t.Fatalf("Failed to get git config: %v", err)
	}
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	if err := gitRepo.SetConfig(cfg); err != nil {
		t.Fatalf("Failed to set git config: %v", err)
	}

	// Create initial files and commit
	worktree, err := gitRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	_ = os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Test Project\n"), 0644)
	_, _ = worktree.Add(".")
	_, err = worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Initialize fogit
	t.Log("Step 2: Initializing fogit...")
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	// Disable fuzzy matching and set create_branch_from to current for test stability
	_, _ = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	_, _ = runFogit(t, projectDir, "config", "set", "workflow.create_branch_from", "current")

	// Commit config changes so they persist across branch switches
	_, _ = worktree.Add(".fogit/config.yml")
	_, _ = worktree.Commit("Configure fogit for testing", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Step 3: Create Feature A (User Authentication) - creates its own branch from main
	// This creates branch feature/user-authentication and stores YAML there
	t.Log("Step 3: Creating Feature A (User Authentication) from main branch...")
	output, err = runFogit(t, projectDir, "feature", "User Authentication",
		"--description", "Implement login and registration")
	if err != nil {
		t.Fatalf("Failed to create Feature A: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature A created:\n%s", output)

	// Step 4: Return to main branch before creating Feature B
	// This is the key difference - we go back to main so Feature B's branch
	// does NOT inherit Feature A's YAML file
	t.Log("Step 4: Returning to main branch...")
	err = worktree.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("master"),
	})
	if err != nil {
		// Try "main" if "master" doesn't exist
		err = worktree.Checkout(&gogit.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName("main"),
		})
		if err != nil {
			t.Fatalf("Failed to checkout main/master branch: %v", err)
		}
	}

	// Step 5: Create Feature B (Database Layer) - creates its own branch from main
	// Now Feature B's branch will NOT have Feature A's YAML (isolated branches)
	t.Log("Step 5: Creating Feature B (Database Layer) from main branch...")
	output, err = runFogit(t, projectDir, "feature", "Database Layer",
		"--description", "Core database functionality")
	if err != nil {
		t.Fatalf("Failed to create Feature B: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature B created:\n%s", output)

	// Step 6: Verify fogit list shows both features (cross-branch discovery)
	t.Log("Step 6: Verifying fogit list shows both features...")
	output, err = runFogit(t, projectDir, "list")
	if err != nil {
		t.Fatalf("fogit list failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "User Authentication") {
		t.Fatalf("Expected 'User Authentication' in list output, got:\n%s", output)
	}
	if !strings.Contains(output, "Database Layer") {
		t.Fatalf("Expected 'Database Layer' in list output, got:\n%s", output)
	}
	t.Logf("✓ Both features visible in list:\n%s", output)

	// Step 7: Try to link features while on feature/database-layer branch
	// This is the bug: link should use cross-branch discovery like list does
	t.Log("Step 7: Attempting to link Database Layer -> User Authentication (cross-branch)...")
	output, err = runFogit(t, projectDir, "link", "Database Layer", "User Authentication", "depends-on",
		"--description", "Database needs auth for user sessions")
	if err != nil {
		// BUG: This currently fails because link doesn't use cross-branch discovery
		t.Fatalf("BUG CONFIRMED: Failed to link features across branches: %v\nOutput: %s\n"+
			"This is the bug: fogit link should use cross-branch discovery like fogit list does.\n"+
			"Per spec: 'Creating relationships between features on different branches' MUST work.", err, output)
	}

	t.Logf("✓ Link created successfully:\n%s", output)

	// Step 8: Verify the relationship exists
	t.Log("Step 8: Verifying relationship exists...")
	output, err = runFogit(t, projectDir, "relationships", "Database Layer")
	if err != nil {
		t.Fatalf("Failed to list relationships: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "User Authentication") || !strings.Contains(output, "depends-on") {
		t.Fatalf("Expected relationship to User Authentication in output, got:\n%s", output)
	}
	t.Logf("✓ Relationship verified:\n%s", output)
}

// TestCrossBranch_LinkBothDirections tests linking in both directions across branches
func TestCrossBranch_LinkBothDirections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir := t.TempDir()

	// Initialize Git repository
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cfg, _ := gitRepo.Config()
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	_ = gitRepo.SetConfig(cfg)

	worktree, _ := gitRepo.Worktree()
	_ = os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Test\n"), 0644)
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Initialize fogit
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	// Disable fuzzy matching and set create_branch_from to current for test stability
	_, _ = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	_, _ = runFogit(t, projectDir, "config", "set", "workflow.create_branch_from", "current")

	// Commit config changes so they persist across branch switches
	_, _ = worktree.Add(".fogit/config.yml")
	_, _ = worktree.Commit("Configure fogit for testing", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Create three features on separate branches
	t.Log("Creating Authentication Feature...")
	output, err = runFogit(t, projectDir, "feature", "Authentication Feature", "-d", "First feature")
	if err != nil {
		t.Fatalf("Failed to create Authentication Feature: %v\nOutput: %s", err, output)
	}

	t.Log("Creating Database Feature...")
	output, err = runFogit(t, projectDir, "feature", "Database Feature", "-d", "Second feature")
	if err != nil {
		t.Fatalf("Failed to create Database Feature: %v\nOutput: %s", err, output)
	}

	t.Log("Creating Cache Feature...")
	output, err = runFogit(t, projectDir, "feature", "Cache Feature", "-d", "Third feature")
	if err != nil {
		t.Fatalf("Failed to create Cache Feature: %v\nOutput: %s", err, output)
	}

	// We're now on feature/cache-feature branch

	// Link Cache Feature -> Authentication Feature (source exists locally, target on different branch)
	t.Log("Linking Cache -> Auth (target on different branch)...")
	output, err = runFogit(t, projectDir, "link", "Cache Feature", "Authentication Feature", "depends-on")
	if err != nil {
		t.Fatalf("BUG: Failed to link Cache -> Auth: %v\nOutput: %s", err, output)
	}
	t.Logf("✓ Cache -> Auth link created:\n%s", output)

	// Link Authentication Feature -> Database Feature (source on different branch, target also on different branch)
	// This requires finding Authentication Feature across branches, then creating link on its branch
	t.Log("Linking Auth -> Database (source and target on different branches)...")
	output, err = runFogit(t, projectDir, "link", "Authentication Feature", "Database Feature", "depends-on")
	if err != nil {
		t.Fatalf("BUG: Failed to link Auth -> Database: %v\nOutput: %s", err, output)
	}
	t.Logf("✓ Auth -> Database link created:\n%s", output)

	// Verify all relationships
	t.Log("Verifying relationships...")
	output, err = runFogit(t, projectDir, "relationships", "Cache Feature")
	if err != nil {
		t.Fatalf("Failed to list relationships for Cache: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Authentication Feature") {
		t.Errorf("Expected Cache -> Auth relationship, got:\n%s", output)
	}

	output, err = runFogit(t, projectDir, "relationships", "Authentication Feature")
	if err != nil {
		t.Fatalf("Failed to list relationships for Auth: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Database Feature") {
		t.Errorf("Expected Auth -> Database relationship, got:\n%s", output)
	}

	t.Log("✓ All cross-branch relationships verified")
}

// TestCrossBranch_LinkWithRemoteBranch tests linking to a feature that only exists on a remote branch
func TestCrossBranch_LinkWithRemoteBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	baseDir := t.TempDir()
	bareRepoDir := filepath.Join(baseDir, "remote.git")
	devADir := filepath.Join(baseDir, "dev-a")
	devBDir := filepath.Join(baseDir, "dev-b")

	// Initialize developer A workspace
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

	// Create bare repo
	_, err = gogit.PlainInit(bareRepoDir, true)
	if err != nil {
		t.Fatalf("Failed to create bare repo: %v", err)
	}

	_, _ = devARepo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{bareRepoDir},
	})
	_ = devARepo.Push(&gogit.PushOptions{RemoteName: "origin"})

	// Clone to dev B
	devBRepo, err := gogit.PlainClone(devBDir, false, &gogit.CloneOptions{URL: bareRepoDir})
	if err != nil {
		t.Fatalf("Failed to clone: %v", err)
	}

	devBCfg, _ := devBRepo.Config()
	devBCfg.User.Name = "Developer B"
	devBCfg.User.Email = "dev-b@example.com"
	_ = devBRepo.SetConfig(devBCfg)

	// Developer A creates a feature and pushes
	t.Log("Developer A creating Remote Feature...")
	output, err = runFogit(t, devADir, "feature", "Remote Feature", "-d", "Feature from dev A")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	_ = devARepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{"+refs/heads/*:refs/heads/*"},
	})

	// Developer B creates local feature
	t.Log("Developer B creating Local Feature...")
	output, err = runFogit(t, devBDir, "feature", "Local Feature", "-d", "Feature from dev B")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	// Developer B fetches to see remote feature
	_ = devBRepo.Fetch(&gogit.FetchOptions{
		RefSpecs: []config.RefSpec{"+refs/heads/*:refs/remotes/origin/*"},
	})

	// Verify both features visible
	output, err = runFogit(t, devBDir, "list")
	if err != nil {
		t.Fatalf("fogit list failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Remote Feature") || !strings.Contains(output, "Local Feature") {
		t.Fatalf("Expected both features in list, got:\n%s", output)
	}

	// Try to link local feature to remote-only feature
	t.Log("Linking Local Feature -> Remote Feature (target only on remote)...")
	output, err = runFogit(t, devBDir, "link", "Local Feature", "Remote Feature", "depends-on")
	if err != nil {
		t.Fatalf("BUG: Failed to link to remote feature: %v\nOutput: %s", err, output)
	}

	t.Logf("✓ Successfully linked to remote feature:\n%s", output)
}

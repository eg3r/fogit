package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestCrossBranch_CommandsDiscovery tests that analytical commands (tree, impacts, stats, export, show)
// use cross-branch discovery like `fogit list` does.
//
// Bug Report: Multiple commands fail to find features on other branches
// - fogit list shows all 5 features (cross-branch discovery works)
// - fogit tree, impacts, stats, export, show only see current branch features
//
// This creates inconsistent UX where users see features in list but can't analyze them.
func TestCrossBranch_CommandsDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir := t.TempDir()

	// Initialize Git repository
	t.Log("Setting up repository with features on separate branches...")
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

	// Get main branch name
	mainBranch := "master"
	if _, err := gitRepo.Reference(plumbing.NewBranchReferenceName("main"), false); err == nil {
		mainBranch = "main"
	}

	// Create 5 features on separate branches (matching bug report scenario)
	featuresData := []struct {
		name        string
		description string
		priority    string
	}{
		{"user-auth", "User authentication", "medium"},
		{"database-layer", "Database layer", "high"},
		{"api-gateway", "API gateway", "medium"},
		{"logging", "Logging system", "high"},
		{"caching", "Caching layer", "low"},
	}

	for _, f := range featuresData {
		// Return to main before creating each feature
		_ = worktree.Checkout(&gogit.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(mainBranch),
		})

		output, err = runFogit(t, projectDir, "create", f.name, "-d", f.description, "--priority", f.priority)
		if err != nil {
			t.Fatalf("Failed to create %s: %v\nOutput: %s", f.name, err, output)
		}
	}

	// Switch to user-auth and create relationships
	t.Log("Setting up relationships between features...")
	output, err = runFogit(t, projectDir, "switch", "user-auth")
	if err != nil {
		t.Fatalf("Failed to switch to user-auth: %v\nOutput: %s", err, output)
	}

	// Create relationships (user-auth depends on database-layer)
	output, err = runFogit(t, projectDir, "link", "user-auth", "database-layer", "depends-on")
	if err != nil {
		t.Fatalf("Failed to create link: %v\nOutput: %s", err, output)
	}

	// api-gateway depends on user-auth
	output, err = runFogit(t, projectDir, "link", "api-gateway", "user-auth", "depends-on")
	if err != nil {
		t.Fatalf("Failed to create link: %v\nOutput: %s", err, output)
	}

	// api-gateway depends on logging
	output, err = runFogit(t, projectDir, "link", "api-gateway", "logging", "depends-on")
	if err != nil {
		t.Fatalf("Failed to create link: %v\nOutput: %s", err, output)
	}

	// caching depends on database-layer
	output, err = runFogit(t, projectDir, "link", "caching", "database-layer", "depends-on")
	if err != nil {
		t.Fatalf("Failed to create link: %v\nOutput: %s", err, output)
	}

	// Verify fogit list shows all 5 features
	t.Log("Verifying fogit list shows all 5 features...")
	output, err = runFogit(t, projectDir, "list")
	if err != nil {
		t.Fatalf("fogit list failed: %v\nOutput: %s", err, output)
	}

	for _, f := range featuresData {
		if !strings.Contains(output, f.name) {
			t.Fatalf("Expected '%s' in list output, got:\n%s", f.name, output)
		}
	}
	t.Logf("✓ fogit list shows all 5 features:\n%s", output)

	// Now test each command that should use cross-branch discovery
	t.Run("stats", func(t *testing.T) {
		testCrossBranchStats(t, projectDir)
	})

	t.Run("tree", func(t *testing.T) {
		testCrossBranchTree(t, projectDir)
	})

	t.Run("impacts", func(t *testing.T) {
		testCrossBranchImpacts(t, projectDir)
	})

	t.Run("export", func(t *testing.T) {
		testCrossBranchExport(t, projectDir)
	})

	t.Run("show", func(t *testing.T) {
		testCrossBranchShow(t, projectDir)
	})

	t.Run("status", func(t *testing.T) {
		testCrossBranchStatus(t, projectDir)
	})
}

func testCrossBranchStats(t *testing.T, projectDir string) {
	t.Log("Testing fogit stats...")
	output, err := runFogit(t, projectDir, "stats")
	if err != nil {
		t.Fatalf("fogit stats failed: %v\nOutput: %s", err, output)
	}

	// Stats should show 5 features (not 1)
	if strings.Contains(output, "Total Features: 1") || strings.Contains(output, "Total: 1") {
		t.Fatalf("BUG: fogit stats only sees 1 feature (current branch only)\n"+
			"Expected: 5 features\nOutput:\n%s", output)
	}

	// Should contain "5" somewhere for total count
	if !strings.Contains(output, "5") {
		t.Fatalf("BUG: fogit stats should show 5 total features\nOutput:\n%s", output)
	}

	t.Logf("✓ fogit stats correctly shows all features:\n%s", output)
}

func testCrossBranchTree(t *testing.T, projectDir string) {
	t.Log("Testing fogit tree...")

	// Test tree with no argument (should find root features)
	output, err := runFogit(t, projectDir, "tree")
	if err != nil {
		// If it errors with "no root features found", that's the bug
		if strings.Contains(output, "No root features") || strings.Contains(output, "No features found") {
			t.Fatalf("BUG: fogit tree can't find features on other branches\n"+
				"Output: %s\nExpected: Tree showing logging as root (no depends-on)", output)
		}
		t.Fatalf("fogit tree failed: %v\nOutput: %s", err, output)
	}

	// Tree should show features
	if !strings.Contains(output, "logging") && !strings.Contains(output, "database") {
		t.Fatalf("BUG: fogit tree output missing expected features\nOutput:\n%s", output)
	}

	t.Logf("✓ fogit tree shows features:\n%s", output)

	// Test tree with specific feature from another branch
	t.Log("Testing fogit tree logging (feature on different branch)...")
	output, err = runFogit(t, projectDir, "tree", "logging")
	if err != nil {
		if strings.Contains(output, "not found") || strings.Contains(output, "Error") {
			t.Fatalf("BUG: fogit tree can't find 'logging' feature on other branch\n"+
				"Output: %s", output)
		}
		t.Fatalf("fogit tree logging failed: %v\nOutput: %s", err, output)
	}

	t.Logf("✓ fogit tree logging works:\n%s", output)
}

func testCrossBranchImpacts(t *testing.T, projectDir string) {
	t.Log("Testing fogit impacts database-layer (feature on different branch)...")
	output, err := runFogit(t, projectDir, "impacts", "database-layer")
	if err != nil {
		if strings.Contains(output, "not found") || strings.Contains(output, "Error") {
			t.Fatalf("BUG: fogit impacts can't find 'database-layer' feature on other branch\n"+
				"Output: %s\nExpected: Shows user-auth and caching as impacted", output)
		}
		t.Fatalf("fogit impacts failed: %v\nOutput: %s", err, output)
	}

	// database-layer is depended on by user-auth and caching
	// So impacts should show those features
	if !strings.Contains(output, "user-auth") && !strings.Contains(output, "caching") {
		t.Fatalf("BUG: fogit impacts should show dependent features\n"+
			"Expected: user-auth and caching depend on database-layer\nOutput:\n%s", output)
	}

	t.Logf("✓ fogit impacts database-layer works:\n%s", output)
}

func testCrossBranchExport(t *testing.T, projectDir string) {
	t.Log("Testing fogit export json...")
	output, err := runFogit(t, projectDir, "export", "json")
	if err != nil {
		t.Fatalf("fogit export failed: %v\nOutput: %s", err, output)
	}

	// Export should include all 5 features
	features := []string{"user-auth", "database-layer", "api-gateway", "logging", "caching"}
	missingFeatures := []string{}
	for _, f := range features {
		if !strings.Contains(output, f) {
			missingFeatures = append(missingFeatures, f)
		}
	}

	if len(missingFeatures) > 0 {
		t.Fatalf("BUG: fogit export missing features from other branches: %v\n"+
			"Expected all 5 features in export\nOutput:\n%s", missingFeatures, output)
	}

	// All relationships should have target_exists: true (since all features exist)
	if strings.Contains(output, `"target_exists": false`) || strings.Contains(output, `"target_exists":false`) {
		t.Logf("WARNING: Export shows target_exists: false but all features exist")
	}

	t.Logf("✓ fogit export json includes all features")
}

func testCrossBranchShow(t *testing.T, projectDir string) {
	t.Log("Testing fogit show api-gateway (feature on different branch)...")
	output, err := runFogit(t, projectDir, "show", "api-gateway")
	if err != nil {
		if strings.Contains(output, "not found") || strings.Contains(output, "Error") {
			t.Fatalf("BUG: fogit show can't find 'api-gateway' feature on other branch\n"+
				"Output: %s", output)
		}
		t.Fatalf("fogit show failed: %v\nOutput: %s", err, output)
	}

	// Should show the feature details
	if !strings.Contains(output, "api-gateway") || !strings.Contains(output, "API gateway") {
		t.Fatalf("BUG: fogit show output doesn't contain expected feature info\nOutput:\n%s", output)
	}

	t.Logf("✓ fogit show api-gateway works:\n%s", output)
}

func testCrossBranchStatus(t *testing.T, projectDir string) {
	t.Log("Testing fogit status...")
	output, err := runFogit(t, projectDir, "status")
	if err != nil {
		t.Fatalf("fogit status failed: %v\nOutput: %s", err, output)
	}

	// Status should show 5 total features (not 1)
	// Bug: status only counts features on current branch
	if strings.Contains(output, "Features: 1 total") {
		t.Fatalf("BUG: fogit status only sees 1 feature (current branch only)\n"+
			"Expected: 5 features\nOutput:\n%s", output)
	}

	// Should show "Features: 5 total"
	if !strings.Contains(output, "Features: 5 total") {
		t.Fatalf("BUG: fogit status should show 5 total features\nOutput:\n%s", output)
	}

	t.Logf("✓ fogit status correctly shows all features:\n%s", output)
}

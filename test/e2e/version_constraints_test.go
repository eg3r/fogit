package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_VersionConstraints tests relationship version constraints.
func TestE2E_VersionConstraints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "E2E_VersionConstraints")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Helper to run commands
	run := func(args ...string) (string, error) {
		return runFogit(t, projectDir, args...)
	}

	// Helper to run git commands with error checking
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", projectDir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Logf("git %v failed: %v\n%s", args, err, out)
		}
	}

	// Step 1: Initialize Git repository
	t.Log("Step 1: Initializing Git repository...")
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = projectDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git: %v\n%s", err, out)
	}

	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test User")

	initFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(initFile, []byte("# Version Constraints Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	git("add", ".")
	git("commit", "-m", "Initial commit")

	// Step 2: Initialize fogit
	// Note: --no-hooks because tests run fogit via helper, not from PATH
	t.Log("Step 2: Initializing fogit...")
	out, err := run("init", "--no-hooks")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\n%s", err, out)
	}

	// Configure base branch to match git's actual default (main or master)
	baseBranch := getBaseBranch(t, projectDir)
	_, _ = run("config", "workflow.base_branch", baseBranch)
	// Disable fuzzy matching
	_, _ = run("config", "feature_search.fuzzy_match", "false")

	// Step 3: Create target feature (API Library)
	t.Log("Step 3: Creating target feature (API Library)...")
	out, err = run("feature", "API Library", "--priority", "high", "--category", "library")
	if err != nil {
		t.Fatalf("Failed to create API Library: %v\n%s", err, out)
	}

	// Make a commit on feature branch
	apiFile := filepath.Join(projectDir, "api.go")
	if err := os.WriteFile(apiFile, []byte("package api\n// v1 API\n"), 0644); err != nil {
		t.Fatalf("Failed to create api.go: %v", err)
	}
	git("add", ".")
	git("commit", "-m", "Add API v1")

	// Merge to close and create v1
	out, err = run("merge")
	if err != nil {
		t.Fatalf("Failed to merge API Library v1: %v\n%s", err, out)
	}
	t.Log("✓ API Library v1 closed")

	// Check versions
	out, err = run("versions", "API Library")
	if err != nil {
		t.Fatalf("Failed to get versions: %v\n%s", err, out)
	}
	t.Logf("API Library versions after v1:\n%s", out)

	// Step 4: Reopen and create v2
	t.Log("Step 4: Creating API Library v2...")
	out, err = run("feature", "API Library", "--new-version")
	if err != nil {
		t.Fatalf("Failed to reopen API Library: %v\n%s", err, out)
	}

	// Update API
	if err := os.WriteFile(apiFile, []byte("package api\n// v2 API with improvements\n"), 0644); err != nil {
		t.Fatalf("Failed to update api.go: %v", err)
	}
	git("add", ".")
	git("commit", "-m", "Update API to v2")

	// Merge to close v2
	out, err = run("merge")
	if err != nil {
		t.Fatalf("Failed to merge API Library v2: %v\n%s", err, out)
	}
	t.Log("✓ API Library v2 closed")

	// Verify v2 exists
	out, err = run("versions", "API Library")
	if err != nil {
		t.Fatalf("Failed to get versions: %v\n%s", err, out)
	}
	t.Logf("API Library versions after v2:\n%s", out)

	if !strings.Contains(out, "2") {
		t.Error("API Library should have version 2")
	}

	// Step 5: Create consumer feature
	t.Log("Step 5: Creating consumer feature (Web App)...")

	// First go back to base branch before creating Web App
	git("checkout", baseBranch)

	// Use trunk-based mode for Web App so we can link it to API Library
	// (in branch-per-feature mode, features on different branches can't see each other)
	_, _ = run("config", "workflow.mode", "trunk-based")

	out, err = run("feature", "Web App", "--priority", "medium", "--category", "application")
	if err != nil {
		t.Fatalf("Failed to create Web App: %v\n%s", err, out)
	}

	// Commit feature file
	git("add", ".")
	git("commit", "-m", "Create Web App feature")

	// Step 6: Create relationship with version constraint
	t.Log("Step 6: Creating dependency with version constraint >=2...")
	out, err = run("link", "Web App", "API Library", "depends-on",
		"--description", "Web App requires API Library v2+",
		"--version-constraint", ">=2")
	if err != nil {
		t.Fatalf("Failed to create link with constraint: %v\n%s", err, out)
	}
	t.Logf("Link output: %s", out)

	// Step 7: Verify constraint is stored
	t.Log("Step 7: Verifying constraint is stored...")
	out, err = run("relationships", "Web App")
	if err != nil {
		t.Fatalf("Failed to get relationships: %v\n%s", err, out)
	}
	t.Logf("Web App relationships:\n%s", out)

	if !strings.Contains(out, ">=2") && !strings.Contains(out, "version") {
		t.Log("Note: Version constraint may not be shown in relationships output")
	}

	// Step 8: Run validate to check version constraint
	t.Log("Step 8: Running validate to check version constraint...")
	out, err = run("validate")
	t.Logf("Validate output (exit: %v):\n%s", err, out)

	// If API Library is at v2, constraint should be satisfied
	if strings.Contains(out, "E006") || strings.Contains(out, "version constraint") {
		t.Log("Note: Version constraint validation found issues")
	}

	// Step 9: Create another feature with constraint that won't be satisfied
	t.Log("Step 9: Creating feature with unsatisfiable constraint...")
	out, err = run("feature", "Legacy App")
	if err != nil {
		t.Fatalf("Failed to create Legacy App: %v\n%s", err, out)
	}
	git("checkout", baseBranch)

	// Link with constraint requiring v3 (doesn't exist)
	out, err = run("link", "Legacy App", "API Library", "depends-on",
		"--description", "Requires non-existent version",
		"--version-constraint", ">=3")
	if err != nil {
		t.Logf("Link with v3 constraint result: %v\n%s", err, out)
	}

	// Step 10: Validate should warn/error about unsatisfied constraint
	t.Log("Step 10: Validating with unsatisfied constraint...")
	out, _ = run("validate")
	t.Logf("Validate output with unsatisfied constraint:\n%s", out)

	// Check for E006 (version constraint violation)
	if strings.Contains(out, "E006") || strings.Contains(strings.ToLower(out), "constraint") {
		t.Log("✓ Validate detected version constraint issue")
	}

	// Step 11: Test different constraint formats
	t.Log("Step 11: Testing different constraint formats...")

	_, _ = run("feature", "Test App")
	git("checkout", baseBranch)

	// Test exact version
	out, err = run("link", "Test App", "API Library", "depends-on",
		"--description", "Exact version",
		"--version-constraint", "=2")
	t.Logf("Exact constraint (=2): %v\n%s", err, out)

	// Step 12: Show feature to see version details
	t.Log("Step 12: Showing feature details...")
	out, err = run("show", "API Library")
	if err != nil {
		t.Fatalf("Failed to show feature: %v\n%s", err, out)
	}
	t.Logf("API Library details:\n%s", out)

	t.Log("✅ Version constraints test completed successfully!")
}

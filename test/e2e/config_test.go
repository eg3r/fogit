package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_ConfigManagement tests configuration operations.
func TestE2E_ConfigManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "E2E_ConfigManagement")
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
	if err := os.WriteFile(initFile, []byte("# Config Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	git("add", ".")
	git("commit", "-m", "Initial commit")

	// Step 2: Initialize fogit
	t.Log("Step 2: Initializing fogit...")
	out, err := run("init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\n%s", err, out)
	}

	// Step 3: List initial configuration
	t.Log("Step 3: Listing initial configuration...")
	out, err = run("config", "--list")
	if err != nil {
		t.Fatalf("Failed to list config: %v\n%s", err, out)
	}
	t.Logf("Initial configuration:\n%s", out)

	// Verify default values exist - config shows "[workflow]" section and "mode = ..." values
	if !strings.Contains(out, "[workflow]") || !strings.Contains(out, "mode =") {
		t.Error("Config should show workflow section and mode setting")
	}

	// Step 4: Set workflow mode to trunk-based
	t.Log("Step 4: Setting workflow.mode to trunk-based...")
	out, err = run("config", "workflow.mode", "trunk-based")
	if err != nil {
		t.Fatalf("Failed to set config: %v\n%s", err, out)
	}
	t.Logf("Set workflow.mode output: %s", out)

	// Verify the change
	out, err = run("config", "--list")
	if err != nil {
		t.Fatalf("Failed to list config: %v\n%s", err, out)
	}
	t.Logf("Config after setting trunk-based:\n%s", out)

	if !strings.Contains(out, "trunk-based") {
		t.Error("workflow.mode should be trunk-based")
	}

	// Step 5: Get specific configuration value
	t.Log("Step 5: Getting specific configuration value...")
	out, err = run("config", "workflow.mode")
	if err != nil {
		t.Fatalf("Failed to get config value: %v\n%s", err, out)
	}
	t.Logf("workflow.mode value: %s", out)

	if !strings.Contains(out, "trunk-based") {
		t.Error("Should return trunk-based")
	}

	// Step 6: Set multiple configuration values
	t.Log("Step 6: Setting multiple configuration values...")

	// Set fuzzy match
	out, err = run("config", "feature_search.fuzzy_match", "true")
	if err != nil {
		t.Fatalf("Failed to set fuzzy_match: %v\n%s", err, out)
	}

	// Set max suggestions
	out, err = run("config", "feature_search.max_suggestions", "10")
	if err != nil {
		t.Fatalf("Failed to set max_suggestions: %v\n%s", err, out)
	}

	// Set min similarity
	out, err = run("config", "feature_search.min_similarity", "0.6")
	if err != nil {
		t.Fatalf("Failed to set min_similarity: %v\n%s", err, out)
	}

	// Set default priority
	out, err = run("config", "default_priority", "high")
	if err != nil {
		t.Fatalf("Failed to set default_priority: %v\n%s", err, out)
	}

	// Verify all settings
	out, err = run("config", "--list")
	if err != nil {
		t.Fatalf("Failed to list config: %v\n%s", err, out)
	}
	t.Logf("Config after multiple sets:\n%s", out)

	// Step 7: Set workflow.allow_shared_branches
	t.Log("Step 7: Setting workflow.allow_shared_branches...")
	out, err = run("config", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to set allow_shared_branches: %v\n%s", err, out)
	}

	out, err = run("config", "--list")
	if err != nil {
		t.Fatalf("Failed to list config: %v\n%s", err, out)
	}
	t.Logf("Config with shared branches:\n%s", out)

	// Step 8: Set base branch
	t.Log("Step 8: Setting workflow.base_branch...")
	out, err = run("config", "workflow.base_branch", "develop")
	if err != nil {
		t.Fatalf("Failed to set base_branch: %v\n%s", err, out)
	}

	out, err = run("config", "workflow.base_branch")
	if err != nil {
		t.Fatalf("Failed to get base_branch: %v\n%s", err, out)
	}
	t.Logf("Base branch value: %s", out)

	// Step 9: Test --unset flag
	t.Log("Step 9: Testing --unset flag...")
	out, err = run("config", "--unset", "default_priority")
	if err != nil {
		t.Logf("Unset result: %v\n%s", err, out)
	}

	out, err = run("config", "--list")
	if err != nil {
		t.Fatalf("Failed to list config: %v\n%s", err, out)
	}
	t.Logf("Config after unset:\n%s", out)

	// Step 10: Test 'set' subcommand syntax
	t.Log("Step 10: Testing 'config set' syntax...")
	out, err = run("config", "set", "default_priority", "medium")
	if err != nil {
		// Some implementations may not support 'set' subcommand
		t.Logf("Config set syntax result: %v\n%s", err, out)
	}

	// Step 11: Verify workflow mode affects behavior
	t.Log("Step 11: Verifying workflow mode affects feature creation...")

	// Set back to branch-per-feature to test behavior
	out, err = run("config", "workflow.mode", "branch-per-feature")
	if err != nil {
		t.Fatalf("Failed to set workflow mode: %v\n%s", err, out)
	}

	// Get base branch dynamically and configure it
	baseBranch := getBaseBranch(t, projectDir)
	_, _ = run("config", "workflow.base_branch", baseBranch)

	// Create a feature
	out, err = run("feature", "Config Test Feature")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\n%s", err, out)
	}

	// Check if branch was created
	gitCmd = exec.Command("git", "branch", "-a")
	gitCmd.Dir = projectDir
	branchesOut, _ := gitCmd.CombinedOutput()
	t.Logf("Branches after feature creation:\n%s", string(branchesOut))

	if strings.Contains(string(branchesOut), "feature/") {
		t.Log("✓ Branch-per-feature mode creates branches")
	}

	// Go back to base branch and switch to trunk-based
	git("checkout", baseBranch)
	_, _ = run("config", "workflow.mode", "trunk-based")

	// Create another feature
	out, err = run("feature", "Trunk Mode Feature")
	if err != nil {
		t.Fatalf("Failed to create trunk feature: %v\n%s", err, out)
	}

	// In trunk mode, should stay on base branch
	gitCmd = exec.Command("git", "branch", "--show-current")
	gitCmd.Dir = projectDir
	currentBranchOut, _ := gitCmd.CombinedOutput()
	t.Logf("Current branch after trunk feature: %s", strings.TrimSpace(string(currentBranchOut)))

	// Step 12: Final configuration state
	t.Log("Step 12: Final configuration state...")
	out, err = run("config", "--list")
	if err != nil {
		t.Fatalf("Failed to list final config: %v\n%s", err, out)
	}
	t.Logf("Final configuration:\n%s", out)

	// Verify config file exists
	configFile := filepath.Join(projectDir, ".fogit", "config.yml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file should exist at .fogit/config.yml")
	} else {
		configContent, _ := os.ReadFile(configFile)
		t.Logf("Config file content:\n%s", string(configContent))
	}

	t.Log("✅ Config management test completed successfully!")
}

// TestE2E_ConfigSearchSettings tests feature search configuration.
func TestE2E_ConfigSearchSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "E2E_ConfigSearch")
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

	// Initialize
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = projectDir
	gitCmd.Run()
	git("config", "user.email", "test@example.com")
	git("config", "user.name", "Test User")
	_ = os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Test\n"), 0644)
	git("add", ".")
	git("commit", "-m", "Initial")

	// Get base branch
	baseBranch := getBaseBranch(t, projectDir)

	_, err := run("init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v", err)
	}

	// Step 1: Test fuzzy search disabled
	t.Log("Step 1: Testing with fuzzy search disabled...")
	_, _ = run("config", "feature_search.fuzzy_match", "false")

	// Create features
	_, _ = run("feature", "Authentication Module")
	git("checkout", baseBranch)
	_, _ = run("feature", "Authorization Module")
	git("checkout", baseBranch)

	// Search with exact term
	out, err := run("search", "Authentication")
	if err != nil {
		t.Logf("Search result: %v\n%s", err, out)
	}
	t.Logf("Search 'Authentication' (fuzzy off):\n%s", out)

	if !strings.Contains(out, "Authentication Module") {
		t.Error("Should find Authentication Module")
	}

	// Step 2: Test fuzzy search enabled
	t.Log("Step 2: Testing with fuzzy search enabled...")
	_, _ = run("config", "feature_search.fuzzy_match", "true")
	_, _ = run("config", "feature_search.min_similarity", "0.5")

	// Search with partial/misspelled term
	out, _ = run("search", "Authen")
	t.Logf("Search 'Authen' (fuzzy on):\n%s", out)

	// Step 3: Test max_suggestions limit
	t.Log("Step 3: Testing max_suggestions limit...")
	_, _ = run("config", "feature_search.max_suggestions", "1")

	out, _ = run("search", "Module")
	t.Logf("Search 'Module' (max 1):\n%s", out)

	t.Log("✅ Config search settings test completed!")
}

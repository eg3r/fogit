package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_TrunkBasedMultiFeature tests multiple features in trunk-based mode
// where all features are created and worked on without branches.
func TestE2E_TrunkBasedMultiFeature(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "E2E_TrunkBasedMultiFeature")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Helper to run commands
	run := func(args ...string) (string, error) {
		return runFogit(t, projectDir, args...)
	}

	// Step 1: Initialize Git repository
	t.Log("Step 1: Initializing Git repository...")
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = projectDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git: %v\n%s", err, out)
	}

	// Configure git
	exec.Command("git", "-C", projectDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", projectDir, "config", "user.name", "Test User").Run()

	// Create initial commit
	initFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(initFile, []byte("# Trunk Based Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Initial commit").Run()

	// Step 2: Initialize fogit
	// Note: --no-hooks because tests run fogit via helper, not from PATH
	t.Log("Step 2: Initializing fogit...")
	out, err := run("init", "--no-hooks")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\n%s", err, out)
	}

	// Step 3: Set trunk-based workflow mode
	t.Log("Step 3: Setting trunk-based workflow mode...")
	out, err = run("config", "workflow.mode", "trunk-based")
	if err != nil {
		t.Fatalf("Failed to set workflow mode: %v\n%s", err, out)
	}

	// Disable fuzzy matching for reliable feature lookup
	_, _ = run("config", "feature_search.fuzzy_match", "false")

	// Verify config
	out, err = run("config", "--list")
	if err != nil {
		t.Fatalf("Failed to list config: %v\n%s", err, out)
	}
	t.Logf("Config after setting trunk-based:\n%s", out)
	if !strings.Contains(out, "trunk-based") {
		t.Error("Config should show trunk-based mode")
	}

	// Step 4: Create multiple features (no branches should be created)
	t.Log("Step 4: Creating multiple features in trunk-based mode...")

	features := []struct {
		name     string
		priority string
		category string
	}{
		{"API Endpoint", "high", "backend"},
		{"Frontend Widget", "medium", "frontend"},
		{"Database Schema", "critical", "database"},
	}

	for _, f := range features {
		out, err = run("feature", f.name, "--priority", f.priority, "--category", f.category)
		if err != nil {
			t.Fatalf("Failed to create feature '%s': %v\n%s", f.name, err, out)
		}
		t.Logf("Created: %s", f.name)
	}

	// Step 5: Verify all features are on main branch
	t.Log("Step 5: Verifying features are on main branch...")

	// Check current branch
	gitCmd = exec.Command("git", "branch", "--show-current")
	gitCmd.Dir = projectDir
	branchOut, _ := gitCmd.CombinedOutput()
	currentBranch := strings.TrimSpace(string(branchOut))
	t.Logf("Current branch: %s", currentBranch)

	// List all branches
	gitCmd = exec.Command("git", "branch", "-a")
	gitCmd.Dir = projectDir
	branchesOut, _ := gitCmd.CombinedOutput()
	t.Logf("All branches:\n%s", branchesOut)

	// In trunk-based mode, no feature branches should be created
	branchList := string(branchesOut)
	if strings.Contains(branchList, "feature/") {
		t.Error("In trunk-based mode, no feature branches should be created")
	}

	// Step 6: List features - all should be active
	t.Log("Step 6: Listing features...")
	out, err = run("list")
	if err != nil {
		t.Fatalf("Failed to list features: %v\n%s", err, out)
	}
	t.Logf("Features:\n%s", out)

	for _, f := range features {
		if !strings.Contains(out, f.name) {
			t.Errorf("Feature '%s' should be in list", f.name)
		}
	}

	// Step 7: Make changes while working on features
	t.Log("Step 7: Making changes while features are active...")

	// Create some files
	apiFile := filepath.Join(projectDir, "api.go")
	if err := os.WriteFile(apiFile, []byte("package main\n// API endpoint\n"), 0644); err != nil {
		t.Fatalf("Failed to create api.go: %v", err)
	}

	frontendFile := filepath.Join(projectDir, "widget.js")
	if err := os.WriteFile(frontendFile, []byte("// Frontend widget\n"), 0644); err != nil {
		t.Fatalf("Failed to create widget.js: %v", err)
	}

	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Add API and widget files").Run()

	// Step 8: Close features independently (use merge/finish in trunk-based mode)
	t.Log("Step 8: Closing features independently...")

	// Close "API Endpoint" feature
	out, err = run("merge", "API Endpoint")
	if err != nil {
		t.Fatalf("Failed to close API Endpoint feature: %v\n%s", err, out)
	}
	t.Logf("Closed API Endpoint: %s", out)

	// Commit the feature closure
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Close API Endpoint feature").Run()

	// Step 9: Verify feature states after close
	t.Log("Step 9: Verifying feature states...")
	out, err = run("list", "--state", "closed")
	if err != nil {
		t.Fatalf("Failed to list closed features: %v\n%s", err, out)
	}
	t.Logf("Closed features:\n%s", out)

	if !strings.Contains(out, "API Endpoint") {
		t.Error("API Endpoint should be closed")
	}

	out, err = run("list", "--state", "open")
	if err != nil {
		t.Fatalf("Failed to list open features: %v\n%s", err, out)
	}
	t.Logf("Open features:\n%s", out)

	// Frontend Widget and Database Schema should still be open
	if !strings.Contains(out, "Frontend Widget") {
		t.Error("Frontend Widget should still be open")
	}
	if !strings.Contains(out, "Database Schema") {
		t.Error("Database Schema should still be open")
	}

	// Step 10: Close remaining features
	t.Log("Step 10: Closing remaining features...")
	out, err = run("merge", "Frontend Widget")
	if err != nil {
		t.Fatalf("Failed to close Frontend Widget: %v\n%s", err, out)
	}
	t.Logf("Closed Frontend Widget: %s", out)

	// Commit the feature closure
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Close Frontend Widget feature").Run()

	out, err = run("merge", "Database Schema")
	if err != nil {
		t.Fatalf("Failed to close Database Schema: %v\n%s", err, out)
	}
	t.Logf("Closed Database Schema: %s", out)

	// Commit the feature closure
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Close Database Schema feature").Run()

	// Verify all closed
	out, err = run("list", "--state", "open")
	if err != nil {
		t.Fatalf("Failed to list open features: %v\n%s", err, out)
	}
	t.Logf("Open features after closing all:\n%s", out)

	// Should be no features or empty list message
	if strings.Contains(out, "API Endpoint") || strings.Contains(out, "Frontend Widget") || strings.Contains(out, "Database Schema") {
		t.Error("All features should be closed")
	}

	// Step 11: Verify still on main branch
	t.Log("Step 11: Verifying still on main branch...")
	gitCmd = exec.Command("git", "branch", "--show-current")
	gitCmd.Dir = projectDir
	branchOut, _ = gitCmd.CombinedOutput()
	finalBranch := strings.TrimSpace(string(branchOut))
	t.Logf("Final branch: %s", finalBranch)

	if finalBranch != currentBranch {
		t.Errorf("Should still be on %s branch, got %s", currentBranch, finalBranch)
	}

	t.Log("✅ Trunk-based multi-feature test completed successfully!")
}

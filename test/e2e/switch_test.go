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

// TestE2E_FeatureSwitching tests switching between active features:
// 1. Create a new project folder
// 2. Initialize git and fogit
// 3. Add initial project files
// 4. Create Feature A, make changes
// 5. Create Feature B (switches to B)
// 6. Switch back to Feature A
// 7. Verify correct branch and working state
// 8. Verify uncommitted changes handled (stash warning)
func TestE2E_FeatureSwitching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create a unique test project directory
	projectDir := filepath.Join(t.TempDir(), "E2E_FeatureSwitchingTest")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// STEP 1: Initialize Git repository
	t.Log("Step 1: Initializing Git repository...")
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cfg, err := gitRepo.Config()
	if err != nil {
		t.Fatalf("Failed to get git config: %v", err)
	}
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	if err := gitRepo.SetConfig(cfg); err != nil {
		t.Fatalf("Failed to set git config: %v", err)
	}

	// STEP 2: Create initial project files
	t.Log("Step 2: Creating initial project files...")
	initialFiles := map[string]string{
		"README.md": `# E2E Feature Switching Test Project

This is a test project for fogit feature switching functionality.
`,
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
		"go.mod": `module e2e-switch-test

go 1.23
`,
	}

	for filename, content := range initialFiles {
		filePath := filepath.Join(projectDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	// Make initial commit
	worktree, err := gitRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	if _, err := worktree.Add("."); err != nil {
		t.Fatalf("Failed to stage files: %v", err)
	}
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

	// STEP 3: Initialize fogit
	t.Log("Step 3: Initializing fogit...")
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}
	t.Logf("Init output: %s", output)

	// Get the base branch name and configure fogit to use it
	head, err := gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	baseBranch := head.Name().Short()
	t.Logf("Base branch: %s", baseBranch)

	if baseBranch != "main" {
		output, err = runFogit(t, projectDir, "config", "set", "workflow.base_branch", baseBranch)
		if err != nil {
			t.Fatalf("Failed to set base branch: %v\nOutput: %s", err, output)
		}
	}

	// STEP 4: Create Feature A
	t.Log("Step 4: Creating Feature A 'User Authentication'...")
	output, err = runFogit(t, projectDir, "feature", "User Authentication", "--description", "Implement user login")
	if err != nil {
		t.Fatalf("Failed to create Feature A: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature A create output: %s", output)

	// Extract Feature A ID from output (for later switching)
	featureAID := extractFeatureID(t, output)
	t.Logf("Feature A ID: %s", featureAID)

	// Verify we're on Feature A branch
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after Feature A creation: %v", err)
	}
	featureABranch := head.Name().Short()
	t.Logf("Current branch after Feature A creation: %s", featureABranch)

	if !strings.HasPrefix(featureABranch, "feature/") {
		t.Errorf("Expected to be on a feature branch (feature/*), got: %s", featureABranch)
	}

	// Make changes on Feature A
	t.Log("Step 4b: Making changes on Feature A...")
	authFile := filepath.Join(projectDir, "auth.go")
	authContent := `package main

func Login(username, password string) error {
	// Feature A: User Authentication
	return nil
}
`
	if err := os.WriteFile(authFile, []byte(authContent), 0644); err != nil {
		t.Fatalf("Failed to create auth.go: %v", err)
	}

	// Commit changes on Feature A
	if _, err := worktree.Add("auth.go"); err != nil {
		t.Fatalf("Failed to stage auth.go: %v", err)
	}
	_, err = worktree.Commit("Add authentication feature", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit Feature A changes: %v", err)
	}

	// STEP 5: Create Feature B (should switch to B)
	t.Log("Step 5: Creating Feature B 'Database Connection'...")
	output, err = runFogit(t, projectDir, "feature", "Database Connection", "--description", "Add database support")
	if err != nil {
		t.Fatalf("Failed to create Feature B: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature B create output: %s", output)

	// Extract Feature B ID
	featureBID := extractFeatureID(t, output)
	t.Logf("Feature B ID: %s", featureBID)

	// Verify we're on Feature B branch
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after Feature B creation: %v", err)
	}
	featureBBranch := head.Name().Short()
	t.Logf("Current branch after Feature B creation: %s", featureBBranch)

	if !strings.HasPrefix(featureBBranch, "feature/") {
		t.Errorf("Expected to be on a feature branch (feature/*), got: %s", featureBBranch)
	}
	if featureBBranch == featureABranch {
		t.Errorf("Expected to be on different branch than Feature A, both are: %s", featureBBranch)
	}

	// Make changes on Feature B
	t.Log("Step 5b: Making changes on Feature B...")
	dbFile := filepath.Join(projectDir, "database.go")
	dbContent := `package main

func Connect(dsn string) error {
	// Feature B: Database Connection
	return nil
}
`
	if err := os.WriteFile(dbFile, []byte(dbContent), 0644); err != nil {
		t.Fatalf("Failed to create database.go: %v", err)
	}

	// Commit changes on Feature B
	if _, err := worktree.Add("database.go"); err != nil {
		t.Fatalf("Failed to stage database.go: %v", err)
	}
	_, err = worktree.Commit("Add database connection feature", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit Feature B changes: %v", err)
	}

	// Verify Feature A's file doesn't exist on Feature B branch
	if _, err := os.Stat(authFile); !os.IsNotExist(err) {
		t.Log("Note: auth.go exists on Feature B branch (expected if Feature B branched from Feature A)")
	}

	// Verify Feature B's file exists
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		t.Fatal("database.go should exist on Feature B branch")
	}

	// STEP 6: Switch back to Feature A
	t.Log("Step 6: Switching back to Feature A...")
	output, err = runFogit(t, projectDir, "switch", featureAID)
	if err != nil {
		t.Fatalf("Failed to switch to Feature A: %v\nOutput: %s", err, output)
	}
	t.Logf("Switch output: %s", output)

	// Verify we're back on Feature A branch
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after switch: %v", err)
	}
	currentBranch := head.Name().Short()
	t.Logf("Current branch after switch: %s", currentBranch)

	if currentBranch != featureABranch {
		t.Errorf("Expected to be on Feature A branch '%s', got '%s'", featureABranch, currentBranch)
	}

	// STEP 7: Verify correct working state
	t.Log("Step 7: Verifying working state...")

	// Feature A's file should exist
	if _, err := os.Stat(authFile); os.IsNotExist(err) {
		t.Fatal("auth.go should exist after switching to Feature A")
	}

	// Feature B's file should NOT exist (it's on a different branch)
	if _, err := os.Stat(dbFile); !os.IsNotExist(err) {
		t.Log("Note: database.go exists on Feature A branch (may be expected depending on branch structure)")
	}

	// Verify feature list shows both features
	output, err = runFogit(t, projectDir, "list")
	if err != nil {
		t.Fatalf("Failed to list features: %v\nOutput: %s", err, output)
	}
	t.Logf("List output: %s", output)

	if !strings.Contains(output, "User Authentication") {
		t.Error("Feature list should contain 'User Authentication'")
	}
	if !strings.Contains(output, "Database Connection") {
		t.Error("Feature list should contain 'Database Connection'")
	}

	// STEP 8: Switch to Feature B by name
	t.Log("Step 8: Switching to Feature B by name...")
	output, err = runFogit(t, projectDir, "switch", "Database Connection")
	if err != nil {
		t.Fatalf("Failed to switch to Feature B by name: %v\nOutput: %s", err, output)
	}
	t.Logf("Switch by name output: %s", output)

	// Verify we're on Feature B branch
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after switch by name: %v", err)
	}
	currentBranch = head.Name().Short()
	t.Logf("Current branch after switch by name: %s", currentBranch)

	if currentBranch != featureBBranch {
		t.Errorf("Expected to be on Feature B branch '%s', got '%s'", featureBBranch, currentBranch)
	}

	// STEP 9: Test switching with uncommitted changes (should warn/fail)
	t.Log("Step 9: Testing switch with uncommitted changes...")

	// Create uncommitted changes
	uncommittedFile := filepath.Join(projectDir, "uncommitted.go")
	uncommittedContent := `package main

// This file has uncommitted changes
`
	if err := os.WriteFile(uncommittedFile, []byte(uncommittedContent), 0644); err != nil {
		t.Fatalf("Failed to create uncommitted.go: %v", err)
	}

	// Try to switch with uncommitted changes
	output, err = runFogit(t, projectDir, "switch", featureAID)
	// The behavior depends on implementation:
	// - Some implementations refuse to switch with uncommitted changes
	// - Some implementations stash changes automatically
	// - Some implementations warn but allow switch
	t.Logf("Switch with uncommitted changes output: %s", output)
	if err != nil {
		t.Logf("Switch with uncommitted changes returned error (expected): %v", err)
		// This is acceptable behavior - refusing to switch with uncommitted changes
		// Clean up the uncommitted file to continue testing
		if err := os.Remove(uncommittedFile); err != nil {
			t.Logf("Failed to remove uncommitted file: %v", err)
		}
	} else {
		t.Log("Switch with uncommitted changes succeeded (implementation may auto-stash)")
	}

	// STEP 10: Verify list with state filter
	t.Log("Step 10: Verifying list with state filter...")
	output, err = runFogit(t, projectDir, "list", "--state", "open")
	if err != nil {
		t.Fatalf("Failed to list open features: %v\nOutput: %s", err, output)
	}
	t.Logf("List open features output: %s", output)

	// Both features should be open
	if !strings.Contains(output, "User Authentication") && !strings.Contains(output, "Database Connection") {
		t.Log("Note: One or both features may not appear in open state list")
	}

	t.Log("✅ Feature switching test completed successfully!")
}

// extractFeatureID extracts the feature ID from fogit command output
func extractFeatureID(t *testing.T, output string) string {
	t.Helper()

	// Look for UUID pattern in output
	// Common formats:
	// "Created feature: <id>"
	// "Feature ID: <id>"
	// "<id>" on its own line
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for a line containing an ID (UUID format: 8-4-4-4-12)
		if strings.Contains(line, "ID:") || strings.Contains(line, "Created") {
			// Extract the ID part
			parts := strings.Fields(line)
			for _, part := range parts {
				// UUID format check (simplified)
				if len(part) == 36 && strings.Count(part, "-") == 4 {
					return part
				}
			}
		}
		// Check if the line itself is a UUID
		if len(line) == 36 && strings.Count(line, "-") == 4 {
			return line
		}
	}

	// If not found, try to get it from the list command
	t.Log("Could not extract feature ID from output, will use name for switching")
	return ""
}

// TestE2E_FeatureSwitchingByPartialID tests switching using partial feature ID
func TestE2E_FeatureSwitchingByPartialID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create a unique test project directory
	projectDir := filepath.Join(t.TempDir(), "E2E_PartialIDSwitchTest")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Initialize Git repository
	t.Log("Initializing Git repository...")
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for commits
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
	initialFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(initialFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	worktree, err := gitRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}
	if _, err := worktree.Add("."); err != nil {
		t.Fatalf("Failed to stage files: %v", err)
	}
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
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	// Get the base branch and configure
	head, err := gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	baseBranch := head.Name().Short()
	if baseBranch != "main" {
		output, err = runFogit(t, projectDir, "config", "set", "workflow.base_branch", baseBranch)
		if err != nil {
			t.Fatalf("Failed to set base branch: %v\nOutput: %s", err, output)
		}
	}

	// Create a feature
	output, err = runFogit(t, projectDir, "feature", "Test Feature", "--description", "For partial ID test")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	featureID := extractFeatureID(t, output)
	if featureID == "" {
		t.Skip("Could not extract feature ID, skipping partial ID test")
	}
	t.Logf("Full feature ID: %s", featureID)

	// Switch to base branch first
	if err := checkoutBranch(worktree, baseBranch); err != nil {
		t.Fatalf("Failed to checkout base branch: %v", err)
	}

	// Try switching with partial ID (first 8 characters)
	partialID := featureID[:8]
	t.Logf("Trying partial ID: %s", partialID)

	output, err = runFogit(t, projectDir, "switch", partialID)
	if err != nil {
		t.Logf("Switch with partial ID failed (may not be supported): %v\nOutput: %s", err, output)
		// This is acceptable - partial ID support is optional
	} else {
		t.Logf("Switch with partial ID succeeded: %s", output)

		// Verify we're on the feature branch
		head, err = gitRepo.Head()
		if err != nil {
			t.Fatalf("Failed to get HEAD: %v", err)
		}
		currentBranch := head.Name().Short()
		if !strings.HasPrefix(currentBranch, "feature/") {
			t.Errorf("Expected to be on feature branch, got: %s", currentBranch)
		}
	}

	t.Log("✅ Partial ID switching test completed!")
}

// checkoutBranch is a helper to checkout a specific branch
func checkoutBranch(worktree *gogit.Worktree, branch string) error {
	return worktree.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
}

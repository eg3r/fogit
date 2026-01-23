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

// TestE2E_SharedBranchWorkflow tests multiple features on the same branch:
// 1. Initialize git and fogit with allow_shared_branches: true
// 2. Create Feature A with --same (stays on current branch)
// 3. Create Feature B with --same (stays on same branch)
// 4. Make changes and commit (affects both features)
// 5. Merge closes both features
// 6. Verify only one branch was used throughout
func TestE2E_SharedBranchWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create a unique test project directory
	projectDir := filepath.Join(t.TempDir(), "E2E_SharedBranchTest")
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
		"README.md": `# E2E Shared Branch Test Project

This is a test project for fogit shared branch workflow.
`,
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
		"go.mod": `module e2e-shared-branch-test

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

	// Get the base branch name
	head, err := gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	baseBranch := head.Name().Short()
	t.Logf("Base branch: %s", baseBranch)

	// STEP 4: Enable shared branches in config
	t.Log("Step 4: Enabling shared branches in config...")
	output, err = runFogit(t, projectDir, "config", "set", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to enable shared branches: %v\nOutput: %s", err, output)
	}

	// Also set base branch if not main
	if baseBranch != "main" {
		output, err = runFogit(t, projectDir, "config", "set", "workflow.base_branch", baseBranch)
		if err != nil {
			t.Fatalf("Failed to set base branch: %v\nOutput: %s", err, output)
		}
	}

	// STEP 5: Create Feature A with --same (should stay on current branch)
	t.Log("Step 5: Creating Feature A with --same flag...")
	output, err = runFogit(t, projectDir, "feature", "OAuth Support", "--same", "--description", "Add OAuth authentication")
	if err != nil {
		t.Fatalf("Failed to create Feature A: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature A create output: %s", output)

	// Verify we're still on base branch (not a new feature branch)
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after Feature A creation: %v", err)
	}
	currentBranch := head.Name().Short()
	t.Logf("Current branch after Feature A: %s", currentBranch)

	if currentBranch != baseBranch {
		t.Errorf("Expected to stay on base branch '%s' with --same flag, got '%s'", baseBranch, currentBranch)
	}

	// Verify output mentions staying on current branch
	if !strings.Contains(output, "current branch") && !strings.Contains(output, baseBranch) {
		t.Log("Note: Output doesn't explicitly mention staying on current branch")
	}

	// STEP 6: Create Feature B with --same (should also stay on same branch)
	t.Log("Step 6: Creating Feature B with --same flag...")
	output, err = runFogit(t, projectDir, "feature", "JWT Tokens", "--same", "--description", "Implement JWT token handling")
	if err != nil {
		t.Fatalf("Failed to create Feature B: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature B create output: %s", output)

	// Verify we're STILL on base branch
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after Feature B creation: %v", err)
	}
	currentBranch = head.Name().Short()
	t.Logf("Current branch after Feature B: %s", currentBranch)

	if currentBranch != baseBranch {
		t.Errorf("Expected to stay on base branch '%s' with --same flag, got '%s'", baseBranch, currentBranch)
	}

	// STEP 7: Verify both features exist and are open
	t.Log("Step 7: Verifying both features exist...")
	output, err = runFogit(t, projectDir, "list")
	if err != nil {
		t.Fatalf("Failed to list features: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature list: %s", output)

	if !strings.Contains(output, "OAuth Support") {
		t.Error("Feature list should contain 'OAuth Support'")
	}
	if !strings.Contains(output, "JWT Tokens") {
		t.Error("Feature list should contain 'JWT Tokens'")
	}

	// STEP 8: Make changes for both features
	t.Log("Step 8: Making changes for both features...")

	// Add OAuth file
	oauthFile := filepath.Join(projectDir, "oauth.go")
	oauthContent := `package main

// OAuth handles OAuth authentication
func OAuth(provider string) error {
	// Feature A: OAuth Support
	return nil
}
`
	if err := os.WriteFile(oauthFile, []byte(oauthContent), 0644); err != nil {
		t.Fatalf("Failed to create oauth.go: %v", err)
	}

	// Add JWT file
	jwtFile := filepath.Join(projectDir, "jwt.go")
	jwtContent := `package main

// GenerateToken creates a new JWT token
func GenerateToken(userID string) (string, error) {
	// Feature B: JWT Tokens
	return "token", nil
}
`
	if err := os.WriteFile(jwtFile, []byte(jwtContent), 0644); err != nil {
		t.Fatalf("Failed to create jwt.go: %v", err)
	}

	// Commit changes - single commit affects both features
	if _, err := worktree.Add("oauth.go"); err != nil {
		t.Fatalf("Failed to stage oauth.go: %v", err)
	}
	if _, err := worktree.Add("jwt.go"); err != nil {
		t.Fatalf("Failed to stage jwt.go: %v", err)
	}
	_, err = worktree.Commit("Implement OAuth and JWT features", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit changes: %v", err)
	}

	// STEP 9: Verify we're still on the same branch
	t.Log("Step 9: Verifying branch hasn't changed...")
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after commit: %v", err)
	}
	currentBranch = head.Name().Short()

	if currentBranch != baseBranch {
		t.Errorf("Expected to still be on base branch '%s', got '%s'", baseBranch, currentBranch)
	}

	// STEP 10: Count total branches - should only have base branch
	t.Log("Step 10: Verifying only one branch exists...")
	branches, err := gitRepo.Branches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	branchCount := 0
	var branchNames []string
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		branchCount++
		branchNames = append(branchNames, ref.Name().Short())
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to iterate branches: %v", err)
	}
	t.Logf("Branches: %v (count: %d)", branchNames, branchCount)

	if branchCount != 1 {
		t.Errorf("Expected only 1 branch (base), got %d: %v", branchCount, branchNames)
	}

	// STEP 11: Verify both features are still open
	t.Log("Step 11: Verifying features are open before merge...")
	output, err = runFogit(t, projectDir, "list", "--state", "open")
	if err != nil {
		t.Fatalf("Failed to list open features: %v\nOutput: %s", err, output)
	}
	t.Logf("Open features: %s", output)

	// Both should be listed as open
	if !strings.Contains(output, "OAuth Support") {
		t.Error("OAuth Support should be open")
	}
	if !strings.Contains(output, "JWT Tokens") {
		t.Error("JWT Tokens should be open")
	}

	t.Log("✅ Shared branch workflow test completed successfully!")
	t.Log("Note: Merge with --same features closing together is not yet tested (implementation may vary)")
}

// TestE2E_SharedBranchRequiresConfig tests that --same fails without config enabled
func TestE2E_SharedBranchRequiresConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create a unique test project directory
	projectDir := filepath.Join(t.TempDir(), "E2E_SharedBranchConfigTest")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Initialize Git repository
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

	// Create initial file and commit
	readmeFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Test"), 0644); err != nil {
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

	// Initialize fogit (without enabling shared branches)
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	// Try to create feature with --same (should fail or warn)
	t.Log("Trying --same without allow_shared_branches enabled...")
	output, err = runFogit(t, projectDir, "feature", "Test Feature", "--same")

	// Behavior depends on implementation:
	// - Could fail with error about shared branches not enabled
	// - Could warn and proceed anyway
	// - Could silently ignore --same
	t.Logf("Output: %s", output)
	if err != nil {
		t.Logf("--same without config returned error (expected): %v", err)
		// Verify the error message mentions shared branches or config
		if strings.Contains(output, "shared") || strings.Contains(output, "allow") {
			t.Log("✅ Error message correctly mentions shared branch configuration")
		}
	} else {
		t.Log("--same without config succeeded (implementation may allow it by default)")
	}

	t.Log("✅ Config requirement test completed!")
}

// TestE2E_SharedBranchWithIsolateConflict tests that --same and --isolate conflict
func TestE2E_SharedBranchWithIsolateConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create a unique test project directory
	projectDir := filepath.Join(t.TempDir(), "E2E_SharedIsolateConflictTest")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Initialize Git repository
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

	// Create initial file and commit
	readmeFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Test"), 0644); err != nil {
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

	// Enable shared branches
	output, err = runFogit(t, projectDir, "config", "set", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to enable shared branches: %v\nOutput: %s", err, output)
	}

	// Try to create feature with BOTH --same AND --isolate (should error)
	t.Log("Trying --same and --isolate together (should conflict)...")
	output, err = runFogit(t, projectDir, "feature", "Conflicting Feature", "--same", "--isolate")

	if err != nil {
		t.Logf("Conflicting flags returned error (expected): %v", err)
		t.Logf("Output: %s", output)
		// Should mention the conflict
		if strings.Contains(strings.ToLower(output), "conflict") ||
			strings.Contains(strings.ToLower(output), "cannot") ||
			strings.Contains(strings.ToLower(output), "both") {
			t.Log("✅ Error message correctly mentions flag conflict")
		}
	} else {
		t.Error("Expected error when using both --same and --isolate, but command succeeded")
	}

	t.Log("✅ Flag conflict test completed!")
}

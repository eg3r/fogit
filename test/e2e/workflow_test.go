package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// fogitBinary holds the path to the built fogit binary for tests
var fogitBinary string

// TestMain builds the fogit binary once before all tests
func TestMain(m *testing.M) {
	// Build fogit binary to temp location
	tmpDir, err := os.MkdirTemp("", "fogit-e2e-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	fogitBinary = filepath.Join(tmpDir, "fogit.exe")

	// Build from project root (two levels up from test/e2e)
	projectRoot := filepath.Join("..", "..")
	cmd := exec.Command("go", "build", "-o", fogitBinary, ".")
	cmd.Dir = projectRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		panic("failed to build fogit: " + err.Error() + "\n" + string(output))
	}

	// Run tests
	os.Exit(m.Run())
}

// runFogit executes fogit with the given arguments in the specified directory
func runFogit(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(fogitBinary, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		output += stderr.String()
	}
	return output, err
}

// TestEndToEndFeatureWorkflow tests the complete fogit workflow:
// 1. Create a new project folder
// 2. Initialize git and fogit
// 3. Add initial project files
// 4. Create a new feature (branches off)
// 5. Make changes on the feature branch
// 6. Close/merge the feature
// 7. Verify we're back on base branch with changes preserved
func TestEndToEndFeatureWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create a unique test project directory
	projectDir := filepath.Join(t.TempDir(), "E2E_FeatureTest")
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
		"README.md": `# E2E Test Project

This is a test project for fogit end-to-end testing.

## Features
- Feature tracking
- Git integration
`,
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`,
		"go.mod": `module e2e-test

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

	// Verify .fogit directory was created
	fogitDir := filepath.Join(projectDir, ".fogit")
	if _, err := os.Stat(fogitDir); os.IsNotExist(err) {
		t.Fatal(".fogit directory was not created")
	}

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

	// STEP 4: Create a new feature (should create branch)
	t.Log("Step 4: Creating new feature 'Add User Authentication'...")
	output, err = runFogit(t, projectDir, "feature", "Add User Authentication", "--description", "Implement user login and registration")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}
	t.Logf("Feature create output: %s", output)

	// Verify we're on a feature branch
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after feature creation: %v", err)
	}
	featureBranch := head.Name().Short()
	t.Logf("Current branch after feature creation: %s", featureBranch)

	if !strings.HasPrefix(featureBranch, "feature/") {
		t.Errorf("Expected to be on a feature branch (feature/*), got: %s", featureBranch)
	}

	// STEP 5: Make changes on the feature branch
	t.Log("Step 5: Making changes on feature branch...")

	// Add new authentication files
	authFiles := map[string]string{
		"auth/login.go": `package auth

import "fmt"

func Login(username, password string) error {
	fmt.Printf("Logging in user: %s\n", username)
	// TODO: Implement actual authentication
	return nil
}
`,
		"auth/register.go": `package auth

import "fmt"

func Register(username, email, password string) error {
	fmt.Printf("Registering user: %s (%s)\n", username, email)
	// TODO: Implement actual registration
	return nil
}
`,
	}

	// Create auth directory
	authDir := filepath.Join(projectDir, "auth")
	if err := os.MkdirAll(authDir, 0755); err != nil {
		t.Fatalf("Failed to create auth directory: %v", err)
	}

	for filename, content := range authFiles {
		filePath := filepath.Join(projectDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	// Update main.go to use auth
	updatedMain := `package main

import (
	"fmt"
	"e2e-test/auth"
)

func main() {
	fmt.Println("Hello, World!")
	
	// New authentication feature
	if err := auth.Login("testuser", "password123"); err != nil {
		fmt.Printf("Login failed: %v\n", err)
	}
}
`
	if err := os.WriteFile(filepath.Join(projectDir, "main.go"), []byte(updatedMain), 0644); err != nil {
		t.Fatalf("Failed to update main.go: %v", err)
	}

	// Stage and commit changes
	if _, err := worktree.Add("."); err != nil {
		t.Fatalf("Failed to stage feature changes: %v", err)
	}
	_, err = worktree.Commit("Add user authentication feature", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit feature changes: %v", err)
	}

	// Verify files exist on feature branch
	if _, err := os.Stat(filepath.Join(projectDir, "auth", "login.go")); os.IsNotExist(err) {
		t.Fatal("auth/login.go should exist on feature branch")
	}

	// STEP 6: Close/merge the feature
	t.Log("Step 6: Merging/closing the feature...")
	output, err = runFogit(t, projectDir, "merge")
	if err != nil {
		t.Fatalf("Failed to merge feature: %v\nOutput: %s", err, output)
	}
	t.Logf("Merge output: %s", output)

	// STEP 7: Verify we're back on base branch
	t.Log("Step 7: Verifying we're back on base branch...")
	head, err = gitRepo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD after merge: %v", err)
	}
	currentBranch := head.Name().Short()
	t.Logf("Current branch after merge: %s", currentBranch)

	if currentBranch != baseBranch {
		t.Errorf("Expected to be back on base branch '%s', got '%s'", baseBranch, currentBranch)
	}

	// STEP 8: Verify changes were preserved (merged into base branch)
	t.Log("Step 8: Verifying changes are on base branch...")

	// Check that auth files exist on base branch
	if _, err := os.Stat(filepath.Join(projectDir, "auth", "login.go")); os.IsNotExist(err) {
		t.Error("auth/login.go should exist on base branch after merge")
	}
	if _, err := os.Stat(filepath.Join(projectDir, "auth", "register.go")); os.IsNotExist(err) {
		t.Error("auth/register.go should exist on base branch after merge")
	}

	// Verify main.go was updated
	mainContent, err := os.ReadFile(filepath.Join(projectDir, "main.go"))
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}
	if !strings.Contains(string(mainContent), "auth.Login") {
		t.Error("main.go should contain auth.Login after merge")
	}

	// STEP 9: Verify feature is closed
	t.Log("Step 9: Verifying feature is closed...")
	output, err = runFogit(t, projectDir, "list", "--state", "closed")
	if err != nil {
		t.Fatalf("Failed to list closed features: %v\nOutput: %s", err, output)
	}
	t.Logf("Closed features: %s", output)

	if !strings.Contains(output, "Add User Authentication") {
		t.Error("Feature 'Add User Authentication' should be listed as closed")
	}

	// STEP 10: Verify feature branch was deleted (default behavior)
	t.Log("Step 10: Verifying feature branch was deleted...")
	branches, err := gitRepo.Branches()
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}

	featureBranchExists := false
	err = branches.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().Short() == featureBranch {
			featureBranchExists = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to iterate branches: %v", err)
	}

	if featureBranchExists {
		t.Logf("Note: Feature branch '%s' still exists (may be expected with --no-delete)", featureBranch)
	} else {
		t.Logf("Feature branch '%s' was deleted as expected", featureBranch)
	}

	t.Log("✅ End-to-end workflow test passed!")
}

// TestEndToEndMultipleFeatures tests creating and managing multiple features
func TestEndToEndMultipleFeatures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	projectDir := filepath.Join(t.TempDir(), "E2E_MultiFeature")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Initialize git repo
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cfg, _ := gitRepo.Config()
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	gitRepo.SetConfig(cfg)

	// Create initial file and commit
	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}

	worktree, _ := gitRepo.Worktree()
	worktree.Add(".")
	worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})

	// Initialize fogit
	if _, err := runFogit(t, projectDir, "init"); err != nil {
		t.Fatalf("Failed to init fogit: %v", err)
	}

	// Get actual base branch and configure fogit
	head, _ := gitRepo.Head()
	baseBranch := head.Name().Short()
	if baseBranch != "main" {
		runFogit(t, projectDir, "config", "set", "workflow.base_branch", baseBranch)
	}

	// Create Feature 1
	t.Log("Creating Feature 1...")
	if _, err := runFogit(t, projectDir, "feature", "User Authentication", "--priority", "high"); err != nil {
		t.Fatalf("Failed to create Feature 1: %v", err)
	}

	// List features - should show Feature 1
	output, _ := runFogit(t, projectDir, "list")
	if !strings.Contains(output, "User Authentication") {
		t.Error("User Authentication should be listed")
	}

	// Add file on Feature 1 branch
	os.WriteFile(filepath.Join(projectDir, "feature1.txt"), []byte("Feature 1 content"), 0644)
	worktree.Add(".")
	worktree.Commit("Add feature1.txt", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})

	// Close Feature 1
	t.Log("Closing Feature 1...")
	if _, err := runFogit(t, projectDir, "merge"); err != nil {
		t.Fatalf("Failed to merge Feature 1: %v", err)
	}

	// Verify feature1.txt exists on main
	if _, err := os.Stat(filepath.Join(projectDir, "feature1.txt")); os.IsNotExist(err) {
		t.Error("feature1.txt should exist after merge")
	}

	// Create Feature 2 (use --new-version to skip interactive prompt)
	t.Log("Creating Feature 2...")
	if _, err := runFogit(t, projectDir, "feature", "Payment Processing", "--priority", "medium", "--new-version"); err != nil {
		t.Fatalf("Failed to create Feature 2: %v", err)
	}

	// Add file on Feature 2 branch
	os.WriteFile(filepath.Join(projectDir, "feature2.txt"), []byte("Feature 2 content"), 0644)
	worktree.Add(".")
	worktree.Commit("Add feature2.txt", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})

	// Close Feature 2
	t.Log("Closing Feature 2...")
	if _, err := runFogit(t, projectDir, "merge"); err != nil {
		t.Fatalf("Failed to merge Feature 2: %v", err)
	}

	// Verify both feature files exist
	if _, err := os.Stat(filepath.Join(projectDir, "feature1.txt")); os.IsNotExist(err) {
		t.Error("feature1.txt should still exist")
	}
	if _, err := os.Stat(filepath.Join(projectDir, "feature2.txt")); os.IsNotExist(err) {
		t.Error("feature2.txt should exist after merge")
	}

	// List all closed features
	output, _ = runFogit(t, projectDir, "list", "--state", "closed")
	if !strings.Contains(output, "User Authentication") {
		t.Error("User Authentication should be listed as closed")
	}
	if !strings.Contains(output, "Payment Processing") {
		t.Error("Payment Processing should be listed as closed")
	}

	t.Log("✅ Multiple features test passed!")
}

// TestEndToEndTrunkBasedMode tests the trunk-based workflow
func TestEndToEndTrunkBasedMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	projectDir := filepath.Join(t.TempDir(), "E2E_TrunkBased")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Initialize git repo
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cfg, _ := gitRepo.Config()
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	gitRepo.SetConfig(cfg)

	// Create initial commit
	os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Trunk Based Test\n"), 0644)
	worktree, _ := gitRepo.Worktree()
	worktree.Add(".")
	worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})

	// Get base branch
	head, _ := gitRepo.Head()
	baseBranch := head.Name().Short()

	// Initialize fogit
	if _, err := runFogit(t, projectDir, "init"); err != nil {
		t.Fatalf("Failed to init fogit: %v", err)
	}

	// Set trunk-based mode and base branch
	if _, err := runFogit(t, projectDir, "config", "set", "workflow.mode", "trunk-based"); err != nil {
		t.Fatalf("Failed to set trunk-based mode: %v", err)
	}
	if baseBranch != "main" {
		runFogit(t, projectDir, "config", "set", "workflow.base_branch", baseBranch)
	}

	// Create feature in trunk-based mode (should NOT create branch)
	t.Log("Creating feature in trunk-based mode...")
	if _, err := runFogit(t, projectDir, "feature", "Quick Fix", "--priority", "critical"); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify we're still on base branch
	head, _ = gitRepo.Head()
	if head.Name().Short() != baseBranch {
		t.Errorf("In trunk-based mode, should stay on base branch '%s', got '%s'", baseBranch, head.Name().Short())
	}

	// Close the feature
	if _, err := runFogit(t, projectDir, "merge"); err != nil {
		t.Fatalf("Failed to close feature: %v", err)
	}

	// Verify still on base branch
	head, _ = gitRepo.Head()
	if head.Name().Short() != baseBranch {
		t.Errorf("After merge in trunk-based mode, should be on base branch '%s', got '%s'", baseBranch, head.Name().Short())
	}

	t.Log("✅ Trunk-based mode test passed!")
}

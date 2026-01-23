package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_MergeSquash tests squash merge functionality.
func TestE2E_MergeSquash(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "E2E_MergeSquash")
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
	if err := os.WriteFile(initFile, []byte("# Squash Merge Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	git("add", ".")
	git("commit", "-m", "Initial commit")

	// Get initial commit count on master
	gitCmd = exec.Command("git", "rev-list", "--count", "HEAD")
	gitCmd.Dir = projectDir
	initialCountOut, _ := gitCmd.CombinedOutput()
	t.Logf("Initial commit count: %s", strings.TrimSpace(string(initialCountOut)))

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

	// Step 3: Create feature
	t.Log("Step 3: Creating feature with multiple commits...")
	out, err = run("feature", "Multi Commit Feature", "--priority", "high")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\n%s", err, out)
	}

	// Step 4: Make multiple commits on feature branch
	t.Log("Step 4: Making multiple commits...")

	// Commit 1
	file1 := filepath.Join(projectDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("Content 1\n"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	git("add", ".")
	git("commit", "-m", "Add file 1")

	// Commit 2
	file2 := filepath.Join(projectDir, "file2.txt")
	if err := os.WriteFile(file2, []byte("Content 2\n"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	git("add", ".")
	git("commit", "-m", "Add file 2")

	// Commit 3
	file3 := filepath.Join(projectDir, "file3.txt")
	if err := os.WriteFile(file3, []byte("Content 3\n"), 0644); err != nil {
		t.Fatalf("Failed to create file3: %v", err)
	}
	git("add", ".")
	git("commit", "-m", "Add file 3")

	// Count commits on feature branch
	gitCmd = exec.Command("git", "rev-list", "--count", "HEAD")
	gitCmd.Dir = projectDir
	featureCountOut, _ := gitCmd.CombinedOutput()
	t.Logf("Commits on feature branch: %s", strings.TrimSpace(string(featureCountOut)))

	// Step 5: Squash merge
	t.Log("Step 5: Performing squash merge...")
	out, err = run("merge", "--squash")
	if err != nil {
		t.Fatalf("Failed to squash merge: %v\n%s", err, out)
	}
	t.Logf("Squash merge output: %s", out)

	// Step 6: Verify single commit added
	t.Log("Step 6: Verifying single commit was added...")
	gitCmd = exec.Command("git", "rev-list", "--count", "HEAD")
	gitCmd.Dir = projectDir
	finalCountOut, _ := gitCmd.CombinedOutput()
	t.Logf("Final commit count on master: %s", strings.TrimSpace(string(finalCountOut)))

	// Check git log
	gitCmd = exec.Command("git", "log", "--oneline", "-5")
	gitCmd.Dir = projectDir
	logOut, _ := gitCmd.CombinedOutput()
	t.Logf("Git log after squash:\n%s", string(logOut))

	// Step 7: Verify all changes are present
	t.Log("Step 7: Verifying all changes are preserved...")
	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Error("file1.txt should exist after squash merge")
	}
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Error("file2.txt should exist after squash merge")
	}
	if _, err := os.Stat(file3); os.IsNotExist(err) {
		t.Error("file3.txt should exist after squash merge")
	}
	t.Log("✓ All files present after squash merge")

	// Step 8: Verify feature is closed
	t.Log("Step 8: Verifying feature is closed...")
	out, err = run("show", "Multi Commit Feature")
	if err != nil {
		t.Fatalf("Failed to show feature: %v\n%s", err, out)
	}
	t.Logf("Feature after squash:\n%s", out)

	if !strings.Contains(out, "closed") {
		t.Error("Feature should be in closed state after merge")
	}

	t.Log("✅ Squash merge test completed successfully!")
}

// TestE2E_MergeKeepBranch tests merge without deleting branch.
func TestE2E_MergeKeepBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "E2E_MergeKeepBranch")
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
	if err := os.WriteFile(initFile, []byte("# Keep Branch Test\n"), 0644); err != nil {
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

	// Step 3: Create feature
	t.Log("Step 3: Creating feature...")
	out, err = run("feature", "Preserved Branch Feature", "--priority", "medium")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\n%s", err, out)
	}

	// Get the feature branch name
	gitCmd = exec.Command("git", "branch", "--show-current")
	gitCmd.Dir = projectDir
	branchOut, _ := gitCmd.CombinedOutput()
	featureBranch := strings.TrimSpace(string(branchOut))
	t.Logf("Feature branch: %s", featureBranch)

	// Step 4: Make changes
	t.Log("Step 4: Making changes on feature branch...")
	testFile := filepath.Join(projectDir, "preserved.txt")
	if err := os.WriteFile(testFile, []byte("Preserved content\n"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	git("add", ".")
	git("commit", "-m", "Add preserved file")

	// Step 5: Merge with --no-delete
	t.Log("Step 5: Merging with --no-delete flag...")
	out, err = run("merge", "--no-delete")
	if err != nil {
		t.Fatalf("Failed to merge with --no-delete: %v\n%s", err, out)
	}
	t.Logf("Merge output: %s", out)

	// Step 6: Verify branch still exists
	t.Log("Step 6: Verifying branch still exists...")
	gitCmd = exec.Command("git", "branch", "-a")
	gitCmd.Dir = projectDir
	branchesOut, _ := gitCmd.CombinedOutput()
	t.Logf("Branches after merge:\n%s", string(branchesOut))

	if !strings.Contains(string(branchesOut), featureBranch) && featureBranch != "" {
		t.Errorf("Branch '%s' should still exist after --no-delete merge", featureBranch)
	} else {
		t.Log("✓ Feature branch preserved after merge")
	}

	// Step 7: Verify feature is closed
	t.Log("Step 7: Verifying feature is closed...")
	out, err = run("show", "Preserved Branch Feature")
	if err != nil {
		t.Fatalf("Failed to show feature: %v\n%s", err, out)
	}
	t.Logf("Feature after merge:\n%s", out)

	if !strings.Contains(out, "closed") {
		t.Error("Feature should be in closed state after merge")
	}

	// Step 8: Verify we're on master/main
	t.Log("Step 8: Verifying current branch...")
	gitCmd = exec.Command("git", "branch", "--show-current")
	gitCmd.Dir = projectDir
	currentOut, _ := gitCmd.CombinedOutput()
	currentBranch := strings.TrimSpace(string(currentOut))
	t.Logf("Current branch: %s", currentBranch)

	if currentBranch != "master" && currentBranch != "main" {
		t.Errorf("Should be on master/main after merge, got %s", currentBranch)
	}

	t.Log("✅ Merge keep branch test completed successfully!")
}

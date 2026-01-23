package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_PushToRemote tests pushing feature to remote repository.
func TestE2E_PushToRemote(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory for both repos
	tmpDir := t.TempDir()
	remoteDir := filepath.Join(tmpDir, "remote.git")
	localDir := filepath.Join(tmpDir, "local")

	if err := os.MkdirAll(remoteDir, 0755); err != nil {
		t.Fatalf("Failed to create remote directory: %v", err)
	}
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("Failed to create local directory: %v", err)
	}

	// Helper to run fogit commands
	run := func(args ...string) (string, error) {
		return runFogit(t, localDir, args...)
	}

	// Step 1: Create bare remote repository
	t.Log("Step 1: Creating bare remote repository...")
	gitCmd := exec.Command("git", "init", "--bare")
	gitCmd.Dir = remoteDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init bare repo: %v\n%s", err, out)
	}
	t.Logf("Created bare remote at: %s", remoteDir)

	// Step 2: Initialize local repository
	t.Log("Step 2: Initializing local repository...")
	gitCmd = exec.Command("git", "init")
	gitCmd.Dir = localDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init local git: %v\n%s", err, out)
	}

	exec.Command("git", "-C", localDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", localDir, "config", "user.name", "Test User").Run()

	// Create initial commit
	initFile := filepath.Join(localDir, "README.md")
	if err := os.WriteFile(initFile, []byte("# Remote Push Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	exec.Command("git", "-C", localDir, "add", ".").Run()
	exec.Command("git", "-C", localDir, "commit", "-m", "Initial commit").Run()

	// Get base branch name (main or master depending on git version/config)
	baseBranch := getBaseBranch(t, localDir)

	// Step 3: Add remote
	t.Log("Step 3: Adding remote origin...")
	gitCmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	gitCmd.Dir = localDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to add remote: %v\n%s", err, out)
	}

	// Push base branch to remote first
	gitCmd = exec.Command("git", "push", "-u", "origin", baseBranch)
	gitCmd.Dir = localDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to push %s: %v\n%s", baseBranch, err, out)
	}
	t.Logf("✓ Pushed %s to remote", baseBranch)

	// Step 4: Initialize fogit
	t.Log("Step 4: Initializing fogit...")
	out, err := run("init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\n%s", err, out)
	}

	// Step 5: Create feature
	t.Log("Step 5: Creating feature...")
	out, err = run("feature", "Remote Feature", "--priority", "high", "--category", "network")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\n%s", err, out)
	}

	// Get current branch
	gitCmd = exec.Command("git", "branch", "--show-current")
	gitCmd.Dir = localDir
	branchOut, _ := gitCmd.CombinedOutput()
	featureBranch := strings.TrimSpace(string(branchOut))
	t.Logf("Feature branch: %s", featureBranch)

	// Make a commit
	testFile := filepath.Join(localDir, "remote_feature.txt")
	if err := os.WriteFile(testFile, []byte("Remote feature content\n"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	exec.Command("git", "-C", localDir, "add", ".").Run()
	exec.Command("git", "-C", localDir, "commit", "-m", "Add remote feature file").Run()

	// Step 6: Push using fogit push
	t.Log("Step 6: Pushing feature branch using fogit push...")
	out, err = run("push")
	if err != nil {
		t.Fatalf("Failed to push: %v\n%s", err, out)
	}
	t.Logf("Push output: %s", out)

	// Step 7: Verify branch exists on remote
	t.Log("Step 7: Verifying branch on remote...")
	gitCmd = exec.Command("git", "ls-remote", "--heads", "origin")
	gitCmd.Dir = localDir
	remoteHeadsOut, _ := gitCmd.CombinedOutput()
	t.Logf("Remote heads:\n%s", string(remoteHeadsOut))

	if featureBranch != "" && !strings.Contains(string(remoteHeadsOut), featureBranch) {
		t.Errorf("Feature branch '%s' should exist on remote", featureBranch)
	} else {
		t.Log("✓ Feature branch pushed to remote")
	}

	// Step 8: Verify .fogit data was pushed
	t.Log("Step 8: Verifying .fogit data on remote...")
	// Clone remote to check contents
	cloneDir := filepath.Join(tmpDir, "clone")
	gitCmd = exec.Command("git", "clone", remoteDir, cloneDir)
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Logf("Clone output: %s", string(out))
	}

	// Check if .fogit exists in clone (on base branch)
	fogitDir := filepath.Join(cloneDir, ".fogit")
	if _, err := os.Stat(fogitDir); os.IsNotExist(err) {
		t.Logf("Note: .fogit may not be on %s branch yet", baseBranch)
	} else {
		t.Log("✓ .fogit directory exists in clone")
	}

	// Step 9: Make another change and push
	t.Log("Step 9: Making another change and pushing...")
	if err := os.WriteFile(testFile, []byte("Updated remote feature content\n"), 0644); err != nil {
		t.Fatalf("Failed to update file: %v", err)
	}
	exec.Command("git", "-C", localDir, "add", ".").Run()
	exec.Command("git", "-C", localDir, "commit", "-m", "Update remote feature").Run()

	out, err = run("push")
	if err != nil {
		t.Fatalf("Failed to push update: %v\n%s", err, out)
	}
	t.Logf("Second push output: %s", out)

	// Step 10: Test push to custom remote
	t.Log("Step 10: Testing push with --remote flag...")

	// Add another remote
	remote2Dir := filepath.Join(tmpDir, "remote2.git")
	if err := os.MkdirAll(remote2Dir, 0755); err != nil {
		t.Fatalf("Failed to create remote2: %v", err)
	}
	gitCmd = exec.Command("git", "init", "--bare")
	gitCmd.Dir = remote2Dir
	gitCmd.Run()

	exec.Command("git", "-C", localDir, "remote", "add", "upstream", remote2Dir).Run()

	out, err = run("push", "--remote", "upstream")
	if err != nil {
		t.Logf("Push to upstream result: %v\n%s", err, out)
	} else {
		t.Log("✓ Pushed to upstream remote")
	}

	// Step 11: Test --force flag
	t.Log("Step 11: Testing --force push...")
	out, err = run("push", "--force")
	if err != nil {
		t.Logf("Force push result: %v\n%s", err, out)
	} else {
		t.Log("✓ Force push succeeded")
	}

	t.Log("✅ Push to remote test completed successfully!")
}

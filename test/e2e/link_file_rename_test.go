package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestLink_FileRenameOnUpdate tests the bug where fogit link renames feature files
// from <name>.yml to <name>-<uuid-prefix>.yml and doesn't commit the changes.
//
// Bug Description:
// - The `fogit link` command renames the feature YAML file
// - The original file is deleted (shows as "deleted" in git)
// - The new file is created but untracked (shows as "untracked" in git)
// - The id_index.json is modified with the new filename
// - None of these changes are committed or staged
//
// This causes `fogit switch` to fail with "uncommitted changes detected"
func TestLink_FileRenameOnUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create a test repository with branch-per-feature mode
	projectDir := t.TempDir()

	// Initialize git repo using go-git
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cfg, _ := gitRepo.Config()
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	_ = gitRepo.SetConfig(cfg)

	worktree, _ := gitRepo.Worktree()

	// Create initial file and commit
	readmePath := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Project\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Initialize fogit (branch-per-feature mode is the default)
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("fogit init failed: %v\nOutput: %s", err, output)
	}

	// Disable fuzzy matching for deterministic results
	_, _ = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Initialize fogit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Create first feature (user-auth)
	output, err = runFogit(t, projectDir, "create", "user-auth", "-d", "User authentication feature")
	if err != nil {
		t.Fatalf("Failed to create user-auth: %v\nOutput: %s", err, output)
	}
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Add user-auth feature", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Verify the initial file name
	featuresDir := filepath.Join(projectDir, ".fogit", "features")
	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		t.Fatalf("Failed to read features dir: %v", err)
	}

	var userAuthFile string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "user-auth") && strings.HasSuffix(entry.Name(), ".yml") {
			userAuthFile = entry.Name()
			break
		}
	}

	if userAuthFile == "" {
		t.Fatal("Could not find user-auth feature file")
	}
	t.Logf("Initial user-auth file: %s", userAuthFile)

	// Check git status is clean
	status, err := worktree.Status()
	if err != nil {
		t.Fatalf("Failed to get git status: %v", err)
	}
	if !status.IsClean() {
		t.Fatalf("Expected clean git status before link, got:\n%s", status.String())
	}
	t.Log("✓ Git status is clean before link")

	// Create second feature (logging) on master branch
	// First, checkout master
	err = worktree.Checkout(&gogit.CheckoutOptions{
		Branch: "refs/heads/master",
		Force:  true,
	})
	if err != nil {
		t.Fatalf("Failed to checkout master: %v", err)
	}

	output, err = runFogit(t, projectDir, "create", "logging", "-d", "Logging feature")
	if err != nil {
		t.Fatalf("Failed to create logging: %v\nOutput: %s", err, output)
	}
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Add logging feature", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Go back to user-auth branch
	err = worktree.Checkout(&gogit.CheckoutOptions{
		Branch: "refs/heads/feature/user-auth",
		Force:  true,
	})
	if err != nil {
		t.Fatalf("Failed to checkout feature/user-auth: %v", err)
	}

	// Check git status is clean again
	status, err = worktree.Status()
	if err != nil {
		t.Fatalf("Failed to get git status: %v", err)
	}
	if !status.IsClean() {
		t.Fatalf("Expected clean git status before link (after branch switch), got:\n%s", status.String())
	}
	t.Log("✓ Git status is clean before link (after branch switch)")

	// Now link user-auth to logging
	t.Log("Creating link: user-auth -> logging (tests)")
	output, err = runFogit(t, projectDir, "link", "user-auth", "logging", "tests")
	if err != nil {
		t.Fatalf("Failed to link features: %v\nOutput: %s", err, output)
	}
	t.Logf("Link output: %s", output)

	// BUG CHECK: Verify git status after link
	// The fix should ensure the file is NOT renamed (would show as D deleted + ?? untracked)
	// Instead it should just be M modified (relationship added)
	status, err = worktree.Status()
	if err != nil {
		t.Fatalf("Failed to get git status after link: %v", err)
	}
	t.Logf("Git status after link: %s", status.String())

	// Check the file was NOT renamed (the bug causes D + ?? entries)
	statusStr := status.String()
	if strings.Contains(statusStr, " D") || strings.Contains(statusStr, "??") ||
		strings.Contains(statusStr, "D ") || strings.Contains(statusStr, "? ") {
		t.Fatalf("BUG: fogit link renamed the feature file and left uncommitted changes:\n%s\n"+
			"This breaks 'fogit switch' because it detects uncommitted changes.", statusStr)
	}

	// Verify the file wasn't renamed
	newEntries, err := os.ReadDir(featuresDir)
	if err != nil {
		t.Fatalf("Failed to read features dir after link: %v", err)
	}

	var newUserAuthFile string
	for _, entry := range newEntries {
		if strings.HasPrefix(entry.Name(), "user-auth") && strings.HasSuffix(entry.Name(), ".yml") {
			newUserAuthFile = entry.Name()
			break
		}
	}

	if newUserAuthFile != userAuthFile {
		t.Fatalf("BUG: Feature file was renamed from %s to %s", userAuthFile, newUserAuthFile)
	}

	t.Logf("✓ Feature file name unchanged: %s", userAuthFile)

	// The file should just be modified (M), which is fine - user can commit manually
	// or auto-commit will handle it if enabled
	if strings.Contains(statusStr, " M") {
		t.Log("✓ Feature file modified (as expected - relationship was added)")
	}

	// Commit the change to test that fogit switch works
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Add relationship", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Now verify fogit switch works (this was failing due to the file rename bug)
	t.Log("Testing fogit switch works after link...")
	output, err = runFogit(t, projectDir, "switch", "logging")
	if err != nil {
		t.Fatalf("fogit switch failed after link (this was the original bug): %v\nOutput: %s", err, output)
	}

	t.Log("✓ fogit switch works correctly after link")
	t.Log("✓ fogit link does not rename feature files")
}

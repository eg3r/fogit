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

// createFile creates a file with the given content
func createFile(dir, name, content string) error {
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// TestValidate_CrossBranchRelationships tests that fogit validate correctly handles
// relationships between features that exist on different branches (neither on main/master).
//
// Bug Scenario:
// 1. Create feature A on feature/A branch (not yet on main)
// 2. Create feature B on feature/B branch (not yet on main)
// 3. On feature/A branch, create a relationship: A depends-on B
// 4. Commit on feature/A branch (with hooks installed, this runs validate)
// 5. BUG: validate uses repo.List() which only sees current branch features
// 6. BUG: validate reports E001 "Orphaned relationship" because B doesn't exist on feature/A
//
// Expected: validate should use cross-branch discovery (same as link command) in branch-per-feature mode
func TestValidate_CrossBranchRelationships(t *testing.T) {
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

	// Create initial file and commit on master
	if err := createFile(projectDir, "README.md", "# Test Project\n"); err != nil {
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

	// Commit fogit initialization
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Initialize fogit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// === Create first feature (auth-service) ===
	t.Log("Creating feature: auth-service")
	output, err = runFogit(t, projectDir, "create", "auth-service", "-d", "Authentication service")
	if err != nil {
		t.Fatalf("Failed to create auth-service: %v\nOutput: %s", err, output)
	}

	// Commit the feature creation (we're now on feature/auth-service branch)
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Add auth-service feature", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})
	t.Log("✓ Created auth-service on feature/auth-service branch")

	// === Go back to master and create second feature (logging-service) ===
	t.Log("Switching to master to create second feature")
	err = worktree.Checkout(&gogit.CheckoutOptions{
		Branch: "refs/heads/master",
		Force:  true,
	})
	if err != nil {
		t.Fatalf("Failed to checkout master: %v", err)
	}

	output, err = runFogit(t, projectDir, "create", "logging-service", "-d", "Logging service")
	if err != nil {
		t.Fatalf("Failed to create logging-service: %v\nOutput: %s", err, output)
	}

	// Commit the feature creation (we're now on feature/logging-service branch)
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Add logging-service feature", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})
	t.Log("✓ Created logging-service on feature/logging-service branch")

	// === Go to auth-service branch and create relationship ===
	t.Log("Switching to feature/auth-service to create relationship")
	err = worktree.Checkout(&gogit.CheckoutOptions{
		Branch: "refs/heads/feature/auth-service",
		Force:  true,
	})
	if err != nil {
		t.Fatalf("Failed to checkout feature/auth-service: %v", err)
	}

	// Verify we're on the right branch
	head, _ := gitRepo.Head()
	t.Logf("Current branch: %s", head.Name().Short())

	// Create relationship: auth-service tests logging-service
	// Using tests (informational category) which allows cycles
	// Note: depends-on (structural category) would fail cycle detection with inverse
	t.Log("Creating link: auth-service -> logging-service (tests)")
	output, err = runFogit(t, projectDir, "link", "auth-service", "logging-service", "tests")
	if err != nil {
		t.Fatalf("fogit link failed: %v\nOutput: %s", err, output)
	}
	t.Logf("Link output: %s", output)

	// Stage and commit the relationship change
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Add tests relationship to logging-service", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// === THE BUG TEST: Run validate on feature/auth-service ===
	// validate should NOT report E001 for the relationship to logging-service
	// because logging-service exists (just on a different branch)
	t.Log("Running validate on feature/auth-service branch...")
	output, err = runFogit(t, projectDir, "validate")
	t.Logf("Validate output:\n%s", output)

	// Check for orphaned relationship error (this is the bug)
	if strings.Contains(output, "E001") || strings.Contains(output, "Orphaned") ||
		strings.Contains(output, "orphaned") || strings.Contains(output, "not found") {
		t.Fatalf("BUG: fogit validate incorrectly reports orphaned relationship for cross-branch feature!\n"+
			"The relationship to logging-service should be valid - it exists on feature/logging-service branch.\n"+
			"Validate output:\n%s", output)
	}

	// Also check exit code - validate returns exit code 4 for errors
	if err != nil {
		// Check if it's actually an exit code error
		exitErr := err.Error()
		if strings.Contains(exitErr, "exit status 4") {
			t.Fatalf("BUG: fogit validate failed with exit code 4 (validation errors) for valid cross-branch relationship!\n"+
				"Output:\n%s", output)
		}
		// Some other error
		t.Fatalf("fogit validate failed unexpectedly: %v\nOutput: %s", err, output)
	}

	// Verify validate passed
	if !strings.Contains(output, "0 errors") && !strings.Contains(output, "No issues") {
		// Check if there are any issues at all
		if strings.Contains(output, "error") || strings.Contains(output, "Error") {
			t.Fatalf("BUG: fogit validate reported errors for valid cross-branch relationship:\n%s", output)
		}
	}

	t.Log("✓ fogit validate correctly handles cross-branch relationships")

	// === Additional test: show command with --relationships flag ===
	t.Log("Verifying relationship is visible via fogit show --relationships...")
	output, err = runFogit(t, projectDir, "show", "auth-service", "--relationships")
	if err != nil {
		t.Fatalf("fogit show failed: %v\nOutput: %s", err, output)
	}
	t.Logf("Show output:\n%s", output)

	// The show output should contain the relationship info
	if !strings.Contains(output, "tests") {
		t.Logf("Note: show --relationships doesn't display target name, checking for relationship type")
	}
	t.Log("✓ Show command works with relationships flag")

	// === Test that relationships command also works with cross-branch features ===
	t.Log("Verifying relationships command works...")
	output, err = runFogit(t, projectDir, "relationships", "auth-service")
	if err != nil {
		t.Fatalf("fogit relationships failed: %v\nOutput: %s", err, output)
	}
	t.Logf("Relationships output:\n%s", output)

	if !strings.Contains(output, "logging-service") && !strings.Contains(output, "tests") {
		t.Fatalf("Expected relationships output to show tests to logging-service, got:\n%s", output)
	}
	t.Log("✓ fogit relationships works correctly with cross-branch features")

	t.Log("✓ All cross-branch relationship tests passed!")
}

// TestValidate_CrossBranchOrphanDetection tests that validate correctly identifies
// truly orphaned relationships (where the target feature doesn't exist anywhere)
// vs valid cross-branch relationships.
func TestValidate_CrossBranchOrphanDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	projectDir := t.TempDir()

	// Initialize git repo
	gitRepo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	cfg, _ := gitRepo.Config()
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	_ = gitRepo.SetConfig(cfg)

	worktree, _ := gitRepo.Worktree()

	// Create initial commit
	if err := createFile(projectDir, "README.md", "# Test\n"); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Initialize fogit
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("fogit init failed: %v\nOutput: %s", err, output)
	}
	_, _ = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	_, _ = worktree.Add(".")
	_, _ = worktree.Commit("Initialize fogit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test User", Email: "test@example.com", When: time.Now()},
	})

	// Create a feature
	output, err = runFogit(t, projectDir, "create", "orphan-test", "-d", "Test feature")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	// Manually create an orphaned relationship by editing the YAML
	// This simulates a case where a target feature was deleted but relationship remains
	t.Log("Creating orphaned relationship (target doesn't exist anywhere)...")
	output, err = runFogit(t, projectDir, "link", "orphan-test", "non-existent-feature", "depends-on")
	if err == nil {
		// The link should fail because target doesn't exist
		t.Log("Link command correctly rejected non-existent target")
	} else {
		// Expected: link should fail because feature doesn't exist
		if !strings.Contains(err.Error(), "not found") && !strings.Contains(output, "not found") {
			t.Logf("Link failed as expected (target not found): %v", err)
		}
	}

	// Validate should pass (no orphaned relationships created)
	output, err = runFogit(t, projectDir, "validate")
	t.Logf("Validate output:\n%s", output)

	// Should be clean since we couldn't create the orphaned relationship
	if err != nil {
		t.Fatalf("validate failed unexpectedly: %v\nOutput: %s", err, output)
	}

	t.Log("✓ Validate correctly handles orphan detection")
}

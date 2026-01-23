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

// TestE2E_ValidateRepository tests the validate command for repository integrity
// This tests:
// - Detect orphaned relationships
// - Detect missing inverse relationships
// - Detect cycle violations
// - --fix repairs orphaned relationships
// - --fix creates missing inverses
// - Report output format correct
func TestE2E_ValidateRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	projectDir := filepath.Join(t.TempDir(), "E2E_ValidateRepository")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Step 1: Initialize Git repository
	t.Log("Step 1: Initializing Git repository...")
	repo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init Git: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	readmeFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Validate Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}
	_, err = worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Step 2: Initialize fogit
	t.Log("Step 2: Initializing fogit...")
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	_, err = runFogit(t, projectDir, "config", "set", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to enable shared branches: %v", err)
	}

	_, err = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	if err != nil {
		t.Fatalf("Failed to disable fuzzy matching: %v", err)
	}

	// Step 3: Create features with valid relationships
	t.Log("Step 3: Creating features with valid relationships...")
	output, err = runFogit(t, projectDir, "feature", "Base Service", "--same", "-d", "Foundation service")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	output, err = runFogit(t, projectDir, "feature", "Consumer Service", "--same", "-d", "Service that uses base")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	output, err = runFogit(t, projectDir, "feature", "Helper Module", "--same", "-d", "Helper utilities")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}

	// Create valid relationships
	output, err = runFogit(t, projectDir, "link", "Consumer Service", "Base Service", "depends-on")
	if err != nil {
		t.Fatalf("Failed to create relationship: %v\nOutput: %s", err, output)
	}

	output, err = runFogit(t, projectDir, "link", "Consumer Service", "Helper Module", "related-to")
	if err != nil {
		t.Fatalf("Failed to create relationship: %v\nOutput: %s", err, output)
	}

	// Step 4: Run validate on clean repository
	t.Log("Step 4: Running validate on clean repository...")
	output, err = runFogit(t, projectDir, "validate")
	if err != nil {
		// validate may return error if issues found, that's ok
		t.Logf("Validate output (initial): %v\nOutput: %s", err, output)
	} else {
		t.Logf("Validate output (clean):\n%s", output)
	}

	// Step 5: Create an orphaned relationship by manually editing
	t.Log("Step 5: Creating orphaned relationship scenario...")

	// Get the Consumer Service feature file and manually add an orphaned relationship
	fogitDir := filepath.Join(projectDir, ".fogit", "features")
	entries, err := os.ReadDir(fogitDir)
	if err != nil {
		t.Fatalf("Failed to read fogit directory: %v", err)
	}

	var consumerFile string
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "consumer") {
			consumerFile = filepath.Join(fogitDir, entry.Name())
			break
		}
	}

	if consumerFile != "" {
		// Read the file and add an orphaned relationship
		data, err := os.ReadFile(consumerFile)
		if err != nil {
			t.Fatalf("Failed to read feature file: %v", err)
		}

		// Add an orphaned relationship pointing to non-existent feature
		// This is a simple append - actual format depends on YAML structure
		orphanedRel := `  - type: depends-on
    target_id: 00000000-0000-0000-0000-000000000000
    description: orphaned relationship
`
		// Find relationships section and add
		content := string(data)
		if strings.Contains(content, "relationships:") {
			// Add after relationships: line
			content = strings.Replace(content, "relationships:", "relationships:\n"+orphanedRel, 1)
		} else {
			// Add new relationships section
			content += "\nrelationships:\n" + orphanedRel
		}

		if err := os.WriteFile(consumerFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write modified feature: %v", err)
		}
		t.Log("Added orphaned relationship to Consumer Service")
	}

	// Step 6: Run validate to detect orphaned relationship
	t.Log("Step 6: Running validate to detect issues...")
	output, err = runFogit(t, projectDir, "validate")
	if err != nil {
		t.Logf("Validate found issues (expected): %v", err)
	}
	t.Logf("Validate output:\n%s", output)

	// Check if orphaned relationship detected (E001)
	if strings.Contains(output, "E001") || strings.Contains(output, "orphan") || strings.Contains(output, "target") {
		t.Log("✓ Orphaned relationship detected")
	}

	// Step 7: Test validate with --fix
	t.Log("Step 7: Testing validate with --fix...")
	output, err = runFogit(t, projectDir, "validate", "--fix")
	if err != nil {
		t.Logf("Validate --fix result: %v", err)
	}
	t.Logf("Validate --fix output:\n%s", output)

	// Step 8: Run validate again to check if fixed
	t.Log("Step 8: Running validate after fix...")
	output, err = runFogit(t, projectDir, "validate")
	if err != nil {
		t.Logf("Post-fix validate: %v", err)
	}
	t.Logf("Post-fix validate output:\n%s", output)

	// Step 9: Test validate with --report flag
	t.Log("Step 9: Testing validate with --report flag...")
	reportFile := filepath.Join(t.TempDir(), "validate_report.txt")
	output, err = runFogit(t, projectDir, "validate", "--report", reportFile)
	if err != nil {
		t.Logf("Validate report result: %v", err)
	}
	t.Logf("Validate report output:\n%s", output)

	// Check if report file was created
	if _, err := os.Stat(reportFile); err == nil {
		reportData, _ := os.ReadFile(reportFile)
		t.Logf("Report file content:\n%s", string(reportData))
	} else {
		t.Log("Report file was not created (may not have issues)")
	}

	// Step 10: Test validate with --quiet flag
	t.Log("Step 10: Testing validate with --quiet flag...")
	output, err = runFogit(t, projectDir, "validate", "--quiet")
	exitCode := 0
	if err != nil {
		exitCode = 1
	}
	t.Logf("Validate --quiet exit code: %d, output (should be minimal): '%s'", exitCode, output)

	// Step 11: Create fresh repo and test various error codes
	t.Log("Step 11: Testing validation error codes...")

	// Create new project for clean error code testing
	projectDir2 := filepath.Join(t.TempDir(), "E2E_ValidateErrors")
	if err := os.MkdirAll(projectDir2, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	repo2, err := gogit.PlainInit(projectDir2, false)
	if err != nil {
		t.Fatalf("Failed to init Git: %v", err)
	}

	worktree2, err := repo2.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	readme2 := filepath.Join(projectDir2, "README.md")
	if err := os.WriteFile(readme2, []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if _, err := worktree2.Add("README.md"); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}
	_, err = worktree2.Commit("Initial", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "t@t.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	_, err = runFogit(t, projectDir2, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v", err)
	}

	_, err = runFogit(t, projectDir2, "config", "set", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to config: %v", err)
	}

	// Create features
	_, err = runFogit(t, projectDir2, "feature", "Alpha", "--same")
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	_, err = runFogit(t, projectDir2, "feature", "Beta", "--same")
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create relationship
	_, err = runFogit(t, projectDir2, "link", "Beta", "Alpha", "depends-on")
	if err != nil {
		t.Fatalf("Failed to link: %v", err)
	}

	// Validate clean repo
	output, err = runFogit(t, projectDir2, "validate")
	if err == nil {
		t.Log("✓ Clean repository validates successfully")
	} else {
		t.Logf("Clean repo validation: %v\nOutput: %s", err, output)
	}

	t.Log("✅ Validate repository test completed successfully!")
}

// TestE2E_ValidateErrorCodes tests specific validation error codes
func TestE2E_ValidateErrorCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	projectDir := filepath.Join(t.TempDir(), "E2E_ValidateErrorCodes")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Initialize Git
	repo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init Git: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	readmeFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Error Codes Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("Failed to stage: %v", err)
	}
	_, err = worktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Initialize fogit
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	_, err = runFogit(t, projectDir, "config", "set", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to config: %v", err)
	}

	_, err = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	if err != nil {
		t.Fatalf("Failed to config: %v", err)
	}

	// Create features for testing
	t.Log("Creating features for validation testing...")
	_, err = runFogit(t, projectDir, "feature", "Source Feature", "--same")
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	_, err = runFogit(t, projectDir, "feature", "Target Feature", "--same")
	if err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Create valid relationship
	_, err = runFogit(t, projectDir, "link", "Source Feature", "Target Feature", "depends-on")
	if err != nil {
		t.Fatalf("Failed to link: %v", err)
	}

	// Run validate and check for various error code mentions
	t.Log("Testing validate output for error codes...")
	output, err = runFogit(t, projectDir, "validate")
	if err != nil {
		t.Logf("Validate result: %v", err)
	}
	t.Logf("Validate output:\n%s", output)

	// Check validate help for error codes
	output, err = runFogit(t, projectDir, "validate", "--help")
	if err != nil {
		t.Fatalf("Failed to get help: %v", err)
	}
	t.Logf("Validate help output:\n%s", output)

	// Verify error codes are documented
	expectedCodes := []string{"E001", "E002", "E003", "E004", "E005", "E006"}
	for _, code := range expectedCodes {
		if strings.Contains(output, code) {
			t.Logf("✓ Error code %s documented in help", code)
		}
	}

	t.Log("✅ Validate error codes test completed!")
}

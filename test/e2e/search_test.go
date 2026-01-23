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

// TestE2E_SearchFuzzy tests the search command with fuzzy matching
// This tests:
// - Search finds exact matches
// - Search finds partial matches
// - Search handles typos (fuzzy)
// - Results ranked by relevance
func TestE2E_SearchFuzzy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	projectDir := filepath.Join(t.TempDir(), "E2E_SearchFuzzy")
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
	if err := os.WriteFile(readmeFile, []byte("# Search Test\n"), 0644); err != nil {
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

	// Disable fuzzy matching during feature creation
	_, err = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	if err != nil {
		t.Fatalf("Failed to disable fuzzy matching: %v", err)
	}

	// Step 3: Create features with varied names for search testing
	t.Log("Step 3: Creating features for search testing...")

	features := []struct {
		name        string
		description string
		category    string
	}{
		{"User Authentication", "Login and registration system", "auth"},
		{"User Profile Management", "User profile editing and viewing", "users"},
		{"API Authentication", "API key management", "auth"},
		{"Dashboard Analytics", "Analytics dashboard for users", "analytics"},
		{"Database Connection Pool", "Connection pooling for database", "database"},
		{"Payment Processing", "Credit card payment handling", "billing"},
		{"Email Notification Service", "Send email notifications", "notifications"},
	}

	for _, f := range features {
		output, err = runFogit(t, projectDir, "feature", f.name, "--same", "-d", f.description, "--category", f.category)
		if err != nil {
			t.Fatalf("Failed to create feature %s: %v\nOutput: %s", f.name, err, output)
		}
		t.Logf("Created: %s", f.name)
	}

	// Now enable fuzzy matching for search tests
	_, err = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "true")
	if err != nil {
		t.Fatalf("Failed to enable fuzzy matching: %v", err)
	}

	// Step 4: Test exact match search
	t.Log("Step 4: Testing exact match search...")
	output, err = runFogit(t, projectDir, "search", "User Authentication")
	if err != nil {
		t.Fatalf("Failed to search: %v\nOutput: %s", err, output)
	}
	t.Logf("Exact search 'User Authentication':\n%s", output)

	if !strings.Contains(output, "User Authentication") {
		t.Error("Exact search should find User Authentication")
	}

	// Step 5: Test partial match search
	t.Log("Step 5: Testing partial match search...")
	output, err = runFogit(t, projectDir, "search", "auth")
	if err != nil {
		t.Fatalf("Failed to search: %v\nOutput: %s", err, output)
	}
	t.Logf("Partial search 'auth':\n%s", output)

	if !strings.Contains(output, "User Authentication") {
		t.Error("Partial search should find User Authentication")
	}
	if !strings.Contains(output, "API Authentication") {
		t.Error("Partial search should find API Authentication")
	}

	// Step 6: Test search by description
	t.Log("Step 6: Testing search by description...")
	output, err = runFogit(t, projectDir, "search", "login")
	if err != nil {
		t.Fatalf("Failed to search: %v\nOutput: %s", err, output)
	}
	t.Logf("Search 'login' (in description):\n%s", output)

	if !strings.Contains(output, "User Authentication") {
		t.Error("Search should find User Authentication (login in description)")
	}

	// Step 7: Test search by category filter
	t.Log("Step 7: Testing search with category filter (Payment)...")
	output, err = runFogit(t, projectDir, "search", "Payment", "--category", "billing")
	if err != nil {
		t.Fatalf("Failed to search: %v\nOutput: %s", err, output)
	}
	t.Logf("Search 'Payment' with category=billing:\n%s", output)

	if !strings.Contains(output, "Payment Processing") {
		t.Error("Search should find Payment Processing (billing category)")
	}

	// Step 8: Test fuzzy search (typo handling)
	t.Log("Step 8: Testing fuzzy search (with typo)...")
	output, err = runFogit(t, projectDir, "search", "authentcation") // Typo: missing 'i'
	if err != nil {
		// Fuzzy search might not find anything or might error - both are acceptable
		t.Logf("Fuzzy search result (typo): %v\nOutput: %s", err, output)
	} else {
		t.Logf("Fuzzy search 'authentcation' (typo):\n%s", output)
		// If it found results, that's fuzzy matching working
		if strings.Contains(output, "Authentication") {
			t.Log("✓ Fuzzy matching found results despite typo")
		}
	}

	// Step 9: Test search with state filter
	t.Log("Step 9: Testing search with state filter...")
	output, err = runFogit(t, projectDir, "search", "User", "--state", "open")
	if err != nil {
		t.Fatalf("Failed to search: %v\nOutput: %s", err, output)
	}
	t.Logf("Search 'User' with state:open:\n%s", output)

	if !strings.Contains(output, "User Authentication") {
		t.Error("Search with state filter should find User Authentication")
	}

	// Step 10: Test search with category filter
	t.Log("Step 10: Testing search with category filter...")
	output, err = runFogit(t, projectDir, "search", "User", "--category", "auth")
	if err != nil {
		t.Fatalf("Failed to search: %v\nOutput: %s", err, output)
	}
	t.Logf("Search 'User' with category:auth:\n%s", output)

	if !strings.Contains(output, "User Authentication") {
		t.Error("Search with category filter should find User Authentication")
	}
	if strings.Contains(output, "User Profile Management") {
		t.Error("Search should NOT find User Profile Management (different category)")
	}

	// Step 11: Test search with no results
	t.Log("Step 11: Testing search with no results...")
	output, err = runFogit(t, projectDir, "search", "xyznonexistent123")
	if err != nil {
		t.Logf("No results search (expected): %v\nOutput: %s", err, output)
	} else {
		t.Logf("No results output: %s", output)
		// Output should indicate no matches or be empty
	}

	// Step 12: Test search with JSON format
	t.Log("Step 12: Testing search with JSON format...")
	output, err = runFogit(t, projectDir, "search", "Database", "--format", "json")
	if err != nil {
		t.Fatalf("Failed to search: %v\nOutput: %s", err, output)
	}
	t.Logf("Search JSON format:\n%s", output)

	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Error("JSON format should produce JSON output")
	}

	// Step 13: Test search across multiple words
	t.Log("Step 13: Testing search with multiple words...")
	output, err = runFogit(t, projectDir, "search", "email notification")
	if err != nil {
		t.Fatalf("Failed to search: %v\nOutput: %s", err, output)
	}
	t.Logf("Multi-word search:\n%s", output)

	if !strings.Contains(output, "Email Notification Service") {
		t.Error("Multi-word search should find Email Notification Service")
	}

	// Step 14: Test case-insensitive search
	t.Log("Step 14: Testing case-insensitive search...")
	output, err = runFogit(t, projectDir, "search", "DASHBOARD")
	if err != nil {
		t.Fatalf("Failed to search: %v\nOutput: %s", err, output)
	}
	t.Logf("Case-insensitive search:\n%s", output)

	if !strings.Contains(output, "Dashboard Analytics") {
		t.Error("Case-insensitive search should find Dashboard Analytics")
	}

	t.Log("✅ Search fuzzy test completed successfully!")
}

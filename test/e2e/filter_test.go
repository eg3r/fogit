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

// TestE2E_FilterExpressions tests the filter command with complex expressions
// This tests:
// - Simple filter: priority:high
// - AND expression: state:open AND priority:high
// - OR expression: category:auth OR category:security
// - NOT expression: NOT state:closed
// - Combined: (priority:high OR priority:critical) AND state:open
// - Wildcard: name:*auth*
func TestE2E_FilterExpressions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	projectDir := filepath.Join(t.TempDir(), "E2E_FilterExpressions")
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
	if err := os.WriteFile(readmeFile, []byte("# Filter Test\n"), 0644); err != nil {
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

	// Step 3: Create diverse features for filtering
	t.Log("Step 3: Creating diverse features...")

	features := []struct {
		name     string
		priority string
		category string
		tags     []string
	}{
		{"Auth Login", "high", "authentication", []string{"security", "backend"}},
		{"Auth Registration", "medium", "authentication", []string{"security", "frontend"}},
		{"User Dashboard", "low", "ui", []string{"frontend"}},
		{"Admin Panel", "critical", "admin", []string{"security", "backend"}},
		{"API Gateway", "high", "infrastructure", []string{"backend", "core"}},
		{"Database Migration", "medium", "database", []string{"backend"}},
		{"Security Audit", "critical", "security", []string{"security", "compliance"}},
	}

	for _, f := range features {
		args := []string{"feature", f.name, "--same", "-p", f.priority, "--category", f.category}
		if len(f.tags) > 0 {
			args = append(args, "--tags", strings.Join(f.tags, ","))
		}
		output, err = runFogit(t, projectDir, args...)
		if err != nil {
			t.Fatalf("Failed to create feature %s: %v\nOutput: %s", f.name, err, output)
		}
		t.Logf("Created: %s (priority=%s, category=%s)", f.name, f.priority, f.category)
	}

	// Close one feature to test state filtering
	output, err = runFogit(t, projectDir, "update", "User Dashboard", "--state", "closed")
	if err != nil {
		t.Fatalf("Failed to close feature: %v\nOutput: %s", err, output)
	}

	// Step 4: Test simple priority filter
	t.Log("Step 4: Testing simple filter (priority:high)...")
	output, err = runFogit(t, projectDir, "filter", "priority:high")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("priority:high results:\n%s", output)

	if !strings.Contains(output, "Auth Login") {
		t.Error("priority:high should include Auth Login")
	}
	if !strings.Contains(output, "API Gateway") {
		t.Error("priority:high should include API Gateway")
	}
	if strings.Contains(output, "User Dashboard") {
		t.Error("priority:high should NOT include User Dashboard (low priority)")
	}

	// Step 5: Test AND expression
	t.Log("Step 5: Testing AND expression (state:open AND priority:high)...")
	output, err = runFogit(t, projectDir, "filter", "state:open AND priority:high")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("state:open AND priority:high results:\n%s", output)

	if !strings.Contains(output, "Auth Login") {
		t.Error("AND filter should include Auth Login (open + high)")
	}
	if !strings.Contains(output, "API Gateway") {
		t.Error("AND filter should include API Gateway (open + high)")
	}

	// Step 6: Test OR expression
	t.Log("Step 6: Testing OR expression (category:authentication OR category:security)...")
	output, err = runFogit(t, projectDir, "filter", "category:authentication OR category:security")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("category OR results:\n%s", output)

	if !strings.Contains(output, "Auth Login") {
		t.Error("OR filter should include Auth Login (authentication)")
	}
	if !strings.Contains(output, "Security Audit") {
		t.Error("OR filter should include Security Audit (security)")
	}

	// Step 7: Test NOT expression
	t.Log("Step 7: Testing NOT expression (NOT state:closed)...")
	output, err = runFogit(t, projectDir, "filter", "NOT state:closed")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("NOT state:closed results:\n%s", output)

	if strings.Contains(output, "User Dashboard") {
		t.Error("NOT state:closed should NOT include User Dashboard (closed)")
	}
	if !strings.Contains(output, "Auth Login") {
		t.Error("NOT state:closed should include Auth Login (open)")
	}

	// Step 8: Test combined expression with grouping
	t.Log("Step 8: Testing combined expression ((priority:high OR priority:critical) AND state:open)...")
	output, err = runFogit(t, projectDir, "filter", "(priority:high OR priority:critical) AND state:open")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("Combined expression results:\n%s", output)

	if !strings.Contains(output, "Auth Login") {
		t.Error("Combined filter should include Auth Login (high + open)")
	}
	if !strings.Contains(output, "Admin Panel") {
		t.Error("Combined filter should include Admin Panel (critical + open)")
	}
	if !strings.Contains(output, "Security Audit") {
		t.Error("Combined filter should include Security Audit (critical + open)")
	}

	// Step 9: Test wildcard filter
	t.Log("Step 9: Testing wildcard filter (name:*Auth*)...")
	output, err = runFogit(t, projectDir, "filter", "name:*Auth*")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("name:*Auth* results:\n%s", output)

	if !strings.Contains(output, "Auth Login") {
		t.Error("Wildcard filter should include Auth Login")
	}
	if !strings.Contains(output, "Auth Registration") {
		t.Error("Wildcard filter should include Auth Registration")
	}
	if strings.Contains(output, "API Gateway") {
		t.Error("Wildcard filter should NOT include API Gateway")
	}

	// Step 10: Test tags filter
	t.Log("Step 10: Testing tags filter (tags:security)...")
	output, err = runFogit(t, projectDir, "filter", "tags:security")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("tags:security results:\n%s", output)

	if !strings.Contains(output, "Auth Login") {
		t.Error("Tags filter should include Auth Login (has security tag)")
	}
	if !strings.Contains(output, "Admin Panel") {
		t.Error("Tags filter should include Admin Panel (has security tag)")
	}
	if strings.Contains(output, "Database Migration") {
		t.Error("Tags filter should NOT include Database Migration (no security tag)")
	}

	// Step 11: Test complex nested expression
	t.Log("Step 11: Testing complex nested expression...")
	output, err = runFogit(t, projectDir, "filter", "tags:backend AND (priority:high OR priority:critical)")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("Complex nested results:\n%s", output)

	if !strings.Contains(output, "Auth Login") {
		t.Error("Complex filter should include Auth Login (backend + high)")
	}
	if !strings.Contains(output, "Admin Panel") {
		t.Error("Complex filter should include Admin Panel (backend + critical)")
	}
	if strings.Contains(output, "Database Migration") {
		t.Error("Complex filter should NOT include Database Migration (backend but medium)")
	}

	// Step 12: Test filter with JSON output format
	t.Log("Step 12: Testing filter with JSON format...")
	output, err = runFogit(t, projectDir, "filter", "priority:critical", "--format", "json")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("JSON format output:\n%s", output)

	if !strings.Contains(output, "{") || !strings.Contains(output, "}") {
		t.Error("JSON format should produce JSON output")
	}

	// Step 13: Test filter with sort
	t.Log("Step 13: Testing filter with sort...")
	output, err = runFogit(t, projectDir, "filter", "state:open", "--sort", "priority")
	if err != nil {
		t.Fatalf("Failed to filter: %v\nOutput: %s", err, output)
	}
	t.Logf("Sorted results:\n%s", output)

	t.Log("✅ Filter expressions test completed successfully!")
}

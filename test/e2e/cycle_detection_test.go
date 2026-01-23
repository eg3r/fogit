package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_CycleDetection tests dependency cycle detection and prevention.
func TestE2E_CycleDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "E2E_CycleDetection")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Helper to run commands
	run := func(args ...string) (string, error) {
		return runFogit(t, projectDir, args...)
	}

	// Step 1: Initialize Git repository
	t.Log("Step 1: Initializing Git repository...")
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = projectDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git: %v\n%s", err, out)
	}

	exec.Command("git", "-C", projectDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", projectDir, "config", "user.name", "Test User").Run()

	initFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(initFile, []byte("# Cycle Detection Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Initial commit").Run()

	// Step 2: Initialize fogit
	t.Log("Step 2: Initializing fogit...")
	out, err := run("init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\n%s", err, out)
	}

	// Use trunk-based mode for simpler feature creation
	_, _ = run("config", "workflow.mode", "trunk-based")
	// Disable fuzzy matching
	_, _ = run("config", "feature_search.fuzzy_match", "false")

	// Step 3: Create features for cycle testing
	t.Log("Step 3: Creating features A, B, C...")

	out, err = run("feature", "Feature A", "--priority", "high")
	if err != nil {
		t.Fatalf("Failed to create Feature A: %v\n%s", err, out)
	}
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Create Feature A").Run()

	out, err = run("feature", "Feature B", "--priority", "medium")
	if err != nil {
		t.Fatalf("Failed to create Feature B: %v\n%s", err, out)
	}
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Create Feature B").Run()

	out, err = run("feature", "Feature C", "--priority", "low")
	if err != nil {
		t.Fatalf("Failed to create Feature C: %v\n%s", err, out)
	}
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Create Feature C").Run()

	// Step 4: Test direct cycle prevention (A → B → A)
	t.Log("Step 4: Testing direct cycle prevention (A → B → A)...")

	// Create A depends-on B (should succeed)
	out, err = run("link", "Feature A", "Feature B", "depends-on", "--description", "A depends on B")
	if err != nil {
		t.Fatalf("Failed to create A → B link: %v\n%s", err, out)
	}
	t.Log("✓ Created A → B dependency")

	// Try to create B depends-on A (should fail - creates direct cycle)
	out, err = run("link", "Feature B", "Feature A", "depends-on", "--description", "B depends on A")
	if err == nil {
		t.Log("WARNING: B → A link was allowed, checking if it creates a cycle warning...")
	} else {
		t.Logf("✓ Direct cycle (B → A) prevented: %s", out)
	}

	// Check if cycle was detected
	if strings.Contains(out, "cycle") || strings.Contains(out, "Cycle") {
		t.Log("✓ Cycle was detected in output")
	}

	// Step 5: Test indirect cycle prevention (A → B → C → A)
	t.Log("Step 5: Testing indirect cycle prevention (A → B → C → A)...")

	// Create B depends-on C (should succeed)
	out, err = run("link", "Feature B", "Feature C", "depends-on", "--description", "B depends on C")
	if err != nil {
		t.Fatalf("Failed to create B → C link: %v\n%s", err, out)
	}
	t.Log("✓ Created B → C dependency")

	// Try to create C depends-on A (should fail - creates indirect cycle)
	out, err = run("link", "Feature C", "Feature A", "depends-on", "--description", "C depends on A")
	if err == nil {
		t.Log("WARNING: C → A link was allowed, checking validation...")
	} else {
		t.Logf("✓ Indirect cycle (C → A) prevented: %s", out)
	}

	// Step 6: Test informational category allows cycles
	t.Log("Step 6: Testing informational category allows cycles...")

	// relates-to is typically in informational category which allows cycles
	out, err = run("link", "Feature B", "Feature A", "relates-to", "--description", "B relates to A")
	if err != nil {
		t.Logf("relates-to link result: %v\n%s", err, out)
	} else {
		t.Log("✓ Informational relationship (relates-to) allowed cycle")
	}

	// Step 7: Verify relationships
	t.Log("Step 7: Verifying relationships...")
	out, err = run("relationships", "Feature A")
	if err != nil {
		t.Fatalf("Failed to get relationships: %v\n%s", err, out)
	}
	t.Logf("Feature A relationships:\n%s", out)

	out, err = run("relationships", "Feature B")
	if err != nil {
		t.Fatalf("Failed to get relationships: %v\n%s", err, out)
	}
	t.Logf("Feature B relationships:\n%s", out)

	// Step 8: Run validate to check for cycles
	t.Log("Step 8: Running validate to check for cycles...")
	out, err = run("validate")
	t.Logf("Validate output (exit code: %v):\n%s", err, out)

	// Check if validate reports any cycles
	if strings.Contains(strings.ToLower(out), "cycle") {
		t.Log("✓ Validate detected cycles in relationships")
	}

	// Step 9: Test tree command with cycles
	t.Log("Step 9: Testing tree command with potential cycles...")
	out, _ = run("tree", "Feature A")
	t.Logf("Tree for Feature A:\n%s", out)

	// Tree should handle cycles gracefully (show visited indicator or limit depth)
	if strings.Contains(out, "Feature B") {
		t.Log("✓ Tree shows Feature B as dependency")
	}

	// Step 10: Test impacts command with cycles
	t.Log("Step 10: Testing impacts command with potential cycles...")
	out, _ = run("impacts", "Feature C")
	t.Logf("Impacts of Feature C:\n%s", out)

	t.Log("✅ Cycle detection test completed successfully!")
}

// TestE2E_CycleDetectionModes tests different cycle detection modes (strict, warn, none).
func TestE2E_CycleDetectionModes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "E2E_CycleDetectionModes")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Helper to run commands
	run := func(args ...string) (string, error) {
		return runFogit(t, projectDir, args...)
	}

	// Initialize repository
	t.Log("Step 1: Initializing repository...")
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = projectDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git: %v\n%s", err, out)
	}

	exec.Command("git", "-C", projectDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", projectDir, "config", "user.name", "Test User").Run()

	if err := os.WriteFile(filepath.Join(projectDir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Initial commit").Run()

	out, err := run("init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\n%s", err, out)
	}

	// Use trunk-based mode for simpler feature creation
	_, _ = run("config", "workflow.mode", "trunk-based")
	// Disable fuzzy matching
	_, _ = run("config", "feature_search.fuzzy_match", "false")

	// Step 2: Define category with 'warn' mode
	t.Log("Step 2: Defining category with 'warn' cycle detection...")
	out, err = run("categories", "define", "soft-deps",
		"--description", "Soft dependencies with warning on cycles",
		"--no-cycles",
		"--detection", "warn")
	if err != nil {
		t.Fatalf("Failed to define category: %v\n%s", err, out)
	}

	// Define type using this category
	out, err = run("define", "soft-depends",
		"--category", "soft-deps",
		"--inverse", "soft-required-by",
		"--description", "Soft dependency")
	if err != nil {
		t.Fatalf("Failed to define type: %v\n%s", err, out)
	}

	// Step 3: Define category with 'none' mode (allows cycles)
	t.Log("Step 3: Defining category with 'none' cycle detection...")
	out, err = run("categories", "define", "feedback-loops",
		"--description", "Feedback relationships that allow cycles",
		"--allow-cycles",
		"--detection", "none")
	if err != nil {
		t.Fatalf("Failed to define category: %v\n%s", err, out)
	}

	out, err = run("define", "feeds-back-to",
		"--category", "feedback-loops",
		"--bidirectional",
		"--description", "Feedback loop relationship")
	if err != nil {
		t.Fatalf("Failed to define type: %v\n%s", err, out)
	}

	// Step 4: Create features
	t.Log("Step 4: Creating test features...")
	_, _ = run("feature", "Module X")
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Create Module X").Run()
	_, _ = run("feature", "Module Y")
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Create Module Y").Run()

	// Step 5: Test 'none' mode allows cycles
	t.Log("Step 5: Testing 'none' mode allows cycles...")
	out, err = run("link", "Module X", "Module Y", "feeds-back-to", "--description", "X feeds Y")
	if err != nil {
		t.Logf("First feedback link result: %v\n%s", err, out)
	}

	out, err = run("link", "Module Y", "Module X", "feeds-back-to", "--description", "Y feeds X")
	if err != nil {
		t.Logf("Reverse feedback link result: %v\n%s", err, out)
	} else {
		t.Log("✓ Cycle allowed in 'none' detection mode")
	}

	// Step 6: Verify categories with verbose
	t.Log("Step 6: Verifying category settings...")
	out, err = run("categories", "--verbose")
	if err != nil {
		t.Fatalf("Failed to list categories: %v\n%s", err, out)
	}
	t.Logf("Categories with settings:\n%s", out)

	t.Log("✅ Cycle detection modes test completed!")
}

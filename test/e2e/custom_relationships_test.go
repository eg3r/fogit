package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestE2E_CustomRelationshipTypes tests defining and using custom relationship types.
func TestE2E_CustomRelationshipTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "E2E_CustomRelationshipTypes")
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

	// Configure git
	exec.Command("git", "-C", projectDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", projectDir, "config", "user.name", "Test User").Run()

	// Create initial commit
	initFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(initFile, []byte("# Custom Relationships Test\n"), 0644); err != nil {
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

	// Step 3: List default categories
	t.Log("Step 3: Listing default categories...")
	out, err = run("categories", "--verbose")
	if err != nil {
		t.Fatalf("Failed to list categories: %v\n%s", err, out)
	}
	t.Logf("Default categories:\n%s", out)

	// Verify default categories exist
	if !strings.Contains(out, "structural") {
		t.Error("Default category 'structural' should exist")
	}
	if !strings.Contains(out, "informational") {
		t.Error("Default category 'informational' should exist")
	}

	// Step 4: List default relationship types
	t.Log("Step 4: Listing default relationship types...")
	out, err = run("types", "--verbose")
	if err != nil {
		t.Fatalf("Failed to list types: %v\n%s", err, out)
	}
	t.Logf("Default types:\n%s", out)

	// Verify default types exist
	if !strings.Contains(out, "depends-on") {
		t.Error("Default type 'depends-on' should exist")
	}
	if !strings.Contains(out, "blocks") {
		t.Error("Default type 'blocks' should exist")
	}

	// Step 5: Define a custom category
	t.Log("Step 5: Defining custom category 'approval'...")
	out, err = run("categories", "define", "approval",
		"--description", "Approval workflow relationships",
		"--no-cycles",
		"--detection", "strict")
	if err != nil {
		t.Fatalf("Failed to define category: %v\n%s", err, out)
	}
	t.Logf("Category define output: %s", out)

	// Verify category was created
	out, err = run("categories")
	if err != nil {
		t.Fatalf("Failed to list categories: %v\n%s", err, out)
	}
	t.Logf("Categories after define:\n%s", out)

	if !strings.Contains(out, "approval") {
		t.Error("Custom category 'approval' should exist")
	}

	// Step 6: Define a custom relationship type with inverse
	t.Log("Step 6: Defining custom type 'approves' with inverse...")
	out, err = run("define", "approves",
		"--category", "approval",
		"--inverse", "approved-by",
		"--description", "Feature approval relationship")
	if err != nil {
		t.Fatalf("Failed to define type: %v\n%s", err, out)
	}
	t.Logf("Type define output: %s", out)

	// Verify type was created
	out, err = run("types", "--verbose")
	if err != nil {
		t.Fatalf("Failed to list types: %v\n%s", err, out)
	}
	t.Logf("Types after define:\n%s", out)

	if !strings.Contains(out, "approves") {
		t.Error("Custom type 'approves' should exist")
	}
	if !strings.Contains(out, "approved-by") {
		t.Error("Inverse type 'approved-by' should exist")
	}

	// Step 7: Define a bidirectional relationship type
	t.Log("Step 7: Defining bidirectional type 'collaborates-with'...")
	out, err = run("define", "collaborates-with",
		"--category", "informational",
		"--bidirectional",
		"--description", "Features that collaborate")
	if err != nil {
		t.Fatalf("Failed to define bidirectional type: %v\n%s", err, out)
	}
	t.Logf("Bidirectional type output: %s", out)

	// Step 8: Create features and use custom relationship
	t.Log("Step 8: Creating features and using custom relationship...")

	// Use trunk-based mode for simpler feature creation
	_, _ = run("config", "workflow.mode", "trunk-based")
	// Disable fuzzy matching
	_, _ = run("config", "feature_search.fuzzy_match", "false")

	out, err = run("feature", "Feature Request")
	if err != nil {
		t.Fatalf("Failed to create Feature Request: %v\n%s", err, out)
	}

	// Commit feature changes
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Create Feature Request").Run()

	out, err = run("feature", "Tech Review")
	if err != nil {
		t.Fatalf("Failed to create Tech Review: %v\n%s", err, out)
	}

	// Commit feature changes
	exec.Command("git", "-C", projectDir, "add", ".").Run()
	exec.Command("git", "-C", projectDir, "commit", "-m", "Create Tech Review").Run()

	// Step 9: Link features using custom type
	t.Log("Step 9: Linking with custom 'approves' type...")
	out, err = run("link", "Tech Review", "Feature Request", "approves",
		"--description", "Tech review approves feature")
	if err != nil {
		t.Fatalf("Failed to create link: %v\n%s", err, out)
	}
	t.Logf("Link output: %s", out)

	// Step 10: Verify relationship and inverse were created
	t.Log("Step 10: Verifying relationship and inverse...")
	out, err = run("relationships", "Tech Review")
	if err != nil {
		t.Fatalf("Failed to get relationships: %v\n%s", err, out)
	}
	t.Logf("Tech Review relationships:\n%s", out)

	if !strings.Contains(out, "approves") {
		t.Error("Tech Review should have 'approves' relationship")
	}

	out, err = run("relationships", "Feature Request")
	if err != nil {
		t.Fatalf("Failed to get relationships: %v\n%s", err, out)
	}
	t.Logf("Feature Request relationships:\n%s", out)

	if !strings.Contains(out, "approved-by") {
		t.Error("Feature Request should have inverse 'approved-by' relationship")
	}

	// Step 11: Test bidirectional relationship
	t.Log("Step 11: Testing bidirectional relationship...")
	out, err = run("feature", "Partner Feature")
	if err != nil {
		t.Fatalf("Failed to create Partner Feature: %v\n%s", err, out)
	}
	exec.Command("git", "-C", projectDir, "checkout", "master").Run()

	out, err = run("link", "Feature Request", "Partner Feature", "collaborates-with",
		"--description", "Features collaborate together")
	if err != nil {
		t.Fatalf("Failed to create bidirectional link: %v\n%s", err, out)
	}

	// Both sides should show the relationship
	out, err = run("relationships", "Feature Request")
	if err != nil {
		t.Fatalf("Failed to get relationships: %v\n%s", err, out)
	}
	t.Logf("Feature Request relationships (after bidirectional):\n%s", out)

	if !strings.Contains(out, "collaborates-with") {
		t.Error("Feature Request should have 'collaborates-with' relationship")
	}

	out, err = run("relationships", "Partner Feature")
	if err != nil {
		t.Fatalf("Failed to get relationships: %v\n%s", err, out)
	}
	t.Logf("Partner Feature relationships:\n%s", out)

	if !strings.Contains(out, "collaborates-with") {
		t.Error("Partner Feature should also have 'collaborates-with' relationship (bidirectional)")
	}

	// Step 12: Filter types by category
	t.Log("Step 12: Filtering types by category...")
	out, err = run("types", "--category", "approval")
	if err != nil {
		t.Fatalf("Failed to filter types: %v\n%s", err, out)
	}
	t.Logf("Approval category types:\n%s", out)

	if !strings.Contains(out, "approves") {
		t.Error("Should show 'approves' type in approval category")
	}

	t.Log("✅ Custom relationship types test completed successfully!")
}

// TestE2E_RelationshipExportImport tests exporting and importing relationship definitions.
func TestE2E_RelationshipExportImport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temp directory
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Helper to run commands
	runIn := func(dir string, args ...string) (string, error) {
		return runFogit(t, dir, args...)
	}

	// Initialize source repository
	t.Log("Step 1: Initializing source repository...")
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = sourceDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git: %v\n%s", err, out)
	}
	exec.Command("git", "-C", sourceDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", sourceDir, "config", "user.name", "Test User").Run()
	if err := os.WriteFile(filepath.Join(sourceDir, "README.md"), []byte("# Source\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	exec.Command("git", "-C", sourceDir, "add", ".").Run()
	exec.Command("git", "-C", sourceDir, "commit", "-m", "Initial commit").Run()

	out, err := runIn(sourceDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\n%s", err, out)
	}

	// Step 2: Define custom category and types in source
	t.Log("Step 2: Defining custom relationships in source...")

	out, err = runIn(sourceDir, "categories", "define", "custom-workflow",
		"--description", "Custom workflow category",
		"--no-cycles")
	if err != nil {
		t.Fatalf("Failed to define category: %v\n%s", err, out)
	}

	out, err = runIn(sourceDir, "define", "reviews",
		"--category", "custom-workflow",
		"--inverse", "reviewed-by",
		"--description", "Code review relationship")
	if err != nil {
		t.Fatalf("Failed to define type: %v\n%s", err, out)
	}

	out, err = runIn(sourceDir, "define", "validates",
		"--category", "custom-workflow",
		"--inverse", "validated-by",
		"--description", "Validation relationship")
	if err != nil {
		t.Fatalf("Failed to define type: %v\n%s", err, out)
	}

	// Step 3: Export to JSON
	t.Log("Step 3: Exporting relationship definitions to JSON...")
	exportFile := filepath.Join(sourceDir, "relationships.json")
	out, err = runIn(sourceDir, "relationship", "export", "json", "--output", exportFile)
	if err != nil {
		t.Fatalf("Failed to export: %v\n%s", err, out)
	}
	t.Logf("Export output: %s", out)

	// Read and display export file
	exportContent, err := os.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}
	t.Logf("Exported JSON:\n%s", string(exportContent))

	if !strings.Contains(string(exportContent), "custom-workflow") {
		t.Error("Export should contain custom-workflow category")
	}
	if !strings.Contains(string(exportContent), "reviews") {
		t.Error("Export should contain reviews type")
	}

	// Step 4: Export to YAML
	t.Log("Step 4: Exporting to YAML format...")
	yamlFile := filepath.Join(sourceDir, "relationships.yaml")
	out, err = runIn(sourceDir, "relationship", "export", "yaml", "--output", yamlFile)
	if err != nil {
		t.Fatalf("Failed to export YAML: %v\n%s", err, out)
	}

	yamlContent, err := os.ReadFile(yamlFile)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}
	t.Logf("Exported YAML:\n%s", string(yamlContent))

	// Step 5: Initialize target repository
	t.Log("Step 5: Initializing target repository...")
	gitCmd = exec.Command("git", "init")
	gitCmd.Dir = targetDir
	if out, err := gitCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to init git in target: %v\n%s", err, out)
	}
	exec.Command("git", "-C", targetDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", targetDir, "config", "user.name", "Test User").Run()
	if err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# Target\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	exec.Command("git", "-C", targetDir, "add", ".").Run()
	exec.Command("git", "-C", targetDir, "commit", "-m", "Initial commit").Run()

	out, err = runIn(targetDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit in target: %v\n%s", err, out)
	}

	// Verify target doesn't have custom types yet
	out, err = runIn(targetDir, "types")
	if err != nil {
		t.Fatalf("Failed to list types in target: %v\n%s", err, out)
	}
	t.Logf("Target types before import:\n%s", out)

	if strings.Contains(out, "reviews") {
		t.Error("Target should not have 'reviews' type yet")
	}

	// Step 6: Import JSON into target
	t.Log("Step 6: Importing JSON into target repository...")
	// Use --merge to merge with existing categories
	out, err = runIn(targetDir, "relationship", "import", exportFile, "--merge")
	if err != nil {
		t.Fatalf("Failed to import: %v\n%s", err, out)
	}
	t.Logf("Import output: %s", out)

	// Step 7: Verify import
	t.Log("Step 7: Verifying imported definitions...")
	out, err = runIn(targetDir, "categories")
	if err != nil {
		t.Fatalf("Failed to list categories: %v\n%s", err, out)
	}
	t.Logf("Target categories after import:\n%s", out)

	if !strings.Contains(out, "custom-workflow") {
		t.Error("Target should have 'custom-workflow' category after import")
	}

	out, err = runIn(targetDir, "types", "--verbose")
	if err != nil {
		t.Fatalf("Failed to list types: %v\n%s", err, out)
	}
	t.Logf("Target types after import:\n%s", out)

	if !strings.Contains(out, "reviews") {
		t.Error("Target should have 'reviews' type after import")
	}
	if !strings.Contains(out, "validates") {
		t.Error("Target should have 'validates' type after import")
	}

	// Step 8: Test --types-only export
	t.Log("Step 8: Testing --types-only export...")
	typesOnlyFile := filepath.Join(sourceDir, "types-only.json")
	out, err = runIn(sourceDir, "relationship", "export", "json", "--types-only", "--output", typesOnlyFile)
	if err != nil {
		t.Fatalf("Failed to export types only: %v\n%s", err, out)
	}

	typesContent, _ := os.ReadFile(typesOnlyFile)
	t.Logf("Types-only export:\n%s", string(typesContent))

	// Step 9: Test --categories-only export
	t.Log("Step 9: Testing --categories-only export...")
	catsOnlyFile := filepath.Join(sourceDir, "categories-only.yaml")
	out, err = runIn(sourceDir, "relationship", "export", "yaml", "--categories-only", "--output", catsOnlyFile)
	if err != nil {
		t.Fatalf("Failed to export categories only: %v\n%s", err, out)
	}

	catsContent, _ := os.ReadFile(catsOnlyFile)
	t.Logf("Categories-only export:\n%s", string(catsContent))

	// Step 10: Test --merge mode (reimport should skip existing)
	t.Log("Step 10: Testing --merge mode...")
	out, err = runIn(targetDir, "relationship", "import", exportFile, "--merge")
	if err != nil {
		t.Fatalf("Failed to merge import: %v\n%s", err, out)
	}
	t.Logf("Merge import output: %s", out)

	// Step 11: Test --overwrite mode
	t.Log("Step 11: Testing --overwrite mode...")
	out, err = runIn(targetDir, "relationship", "import", exportFile, "--overwrite")
	if err != nil {
		t.Fatalf("Failed to overwrite import: %v\n%s", err, out)
	}
	t.Logf("Overwrite import output: %s", out)

	t.Log("✅ Relationship export/import test completed successfully!")
}

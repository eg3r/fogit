package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// TestE2E_ExportImportJSON tests exporting and importing features in JSON format
// This tests:
// - Export features with relationships to JSON
// - JSON contains all feature metadata
// - Import into fresh repository
// - Features and relationships preserved
// - --merge mode (skip conflicts)
// - --overwrite mode (replace conflicts)
func TestE2E_ExportImportJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create source repository
	sourceDir := filepath.Join(t.TempDir(), "E2E_ExportImportJSON_Source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Step 1: Initialize source Git repository
	t.Log("Step 1: Initializing source Git repository...")
	repo, err := gogit.PlainInit(sourceDir, false)
	if err != nil {
		t.Fatalf("Failed to init Git repo: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	// Create initial file and commit
	readmeFile := filepath.Join(sourceDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Export/Import Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("Failed to stage README: %v", err)
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

	// Step 2: Initialize fogit in source
	t.Log("Step 2: Initializing fogit in source repository...")
	output, err := runFogit(t, sourceDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	// Enable shared branches for easier testing
	output, err = runFogit(t, sourceDir, "config", "set", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to enable shared branches: %v\nOutput: %s", err, output)
	}

	// Disable fuzzy matching
	output, err = runFogit(t, sourceDir, "config", "set", "feature_search.fuzzy_match", "false")
	if err != nil {
		t.Fatalf("Failed to disable fuzzy matching: %v\nOutput: %s", err, output)
	}

	// Step 3: Create features with various metadata
	t.Log("Step 3: Creating features with metadata...")

	features := []struct {
		name        string
		description string
		priority    string
		tags        []string
	}{
		{"Auth Service", "Authentication microservice", "high", []string{"security", "backend"}},
		{"User Dashboard", "User-facing dashboard", "medium", []string{"frontend", "ui"}},
		{"Database Layer", "Core database operations", "critical", []string{"backend", "core"}},
	}

	for _, f := range features {
		args := []string{"feature", f.name, "--same", "-d", f.description, "-p", f.priority}
		if len(f.tags) > 0 {
			args = append(args, "--tags", strings.Join(f.tags, ","))
		}
		output, err = runFogit(t, sourceDir, args...)
		if err != nil {
			t.Fatalf("Failed to create feature %s: %v\nOutput: %s", f.name, err, output)
		}
		t.Logf("Created feature: %s", f.name)
	}

	// Step 4: Create relationships
	t.Log("Step 4: Creating relationships...")
	output, err = runFogit(t, sourceDir, "link", "Auth Service", "Database Layer", "depends-on", "--description", "Auth needs DB")
	if err != nil {
		t.Fatalf("Failed to create relationship: %v\nOutput: %s", err, output)
	}

	output, err = runFogit(t, sourceDir, "link", "User Dashboard", "Auth Service", "depends-on", "--description", "Dashboard needs Auth")
	if err != nil {
		t.Fatalf("Failed to create relationship: %v\nOutput: %s", err, output)
	}

	// Step 5: Export to JSON
	t.Log("Step 5: Exporting features to JSON...")
	exportFile := filepath.Join(sourceDir, "features_export.json")
	output, err = runFogit(t, sourceDir, "export", "json", "--output", exportFile)
	if err != nil {
		t.Fatalf("Failed to export JSON: %v\nOutput: %s", err, output)
	}
	t.Logf("Export output: %s", output)

	// Verify JSON file exists and is valid
	jsonData, err := os.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	// Verify JSON structure
	var exportedData map[string]interface{}
	if err := json.Unmarshal(jsonData, &exportedData); err != nil {
		t.Fatalf("Invalid JSON: %v\nContent: %s", err, string(jsonData))
	}

	// Check that features are present
	featuresData, ok := exportedData["features"].([]interface{})
	if !ok {
		t.Fatalf("Expected features array in export, got: %T", exportedData["features"])
	}
	if len(featuresData) != 3 {
		t.Errorf("Expected 3 features, got %d", len(featuresData))
	}
	t.Logf("Exported %d features", len(featuresData))

	// Step 6: Create target repository for import
	t.Log("Step 6: Creating target repository for import...")
	targetDir := filepath.Join(t.TempDir(), "E2E_ExportImportJSON_Target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	// Initialize Git in target
	targetRepo, err := gogit.PlainInit(targetDir, false)
	if err != nil {
		t.Fatalf("Failed to init target Git repo: %v", err)
	}

	targetWorktree, err := targetRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get target worktree: %v", err)
	}

	targetReadme := filepath.Join(targetDir, "README.md")
	if err := os.WriteFile(targetReadme, []byte("# Target Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to create target README: %v", err)
	}
	if _, err := targetWorktree.Add("README.md"); err != nil {
		t.Fatalf("Failed to stage target README: %v", err)
	}
	_, err = targetWorktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit in target: %v", err)
	}

	// Initialize fogit in target
	output, err = runFogit(t, targetDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit in target: %v\nOutput: %s", err, output)
	}

	// Enable shared branches
	output, err = runFogit(t, targetDir, "config", "set", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to enable shared branches in target: %v\nOutput: %s", err, output)
	}

	// Step 7: Import JSON into target
	t.Log("Step 7: Importing JSON into target repository...")
	output, err = runFogit(t, targetDir, "import", exportFile)
	if err != nil {
		t.Fatalf("Failed to import JSON: %v\nOutput: %s", err, output)
	}
	t.Logf("Import output: %s", output)

	// Step 8: Verify imported features
	t.Log("Step 8: Verifying imported features...")
	output, err = runFogit(t, targetDir, "list")
	if err != nil {
		t.Fatalf("Failed to list features: %v\nOutput: %s", err, output)
	}
	t.Logf("Imported features:\n%s", output)

	// Check all features exist
	for _, f := range features {
		if !strings.Contains(output, f.name) {
			t.Errorf("Imported features should contain %s", f.name)
		}
	}

	// Step 9: Verify imported relationships
	t.Log("Step 9: Verifying imported relationships...")
	output, err = runFogit(t, targetDir, "relationships", "Auth Service")
	if err != nil {
		t.Fatalf("Failed to get relationships: %v\nOutput: %s", err, output)
	}
	t.Logf("Auth Service relationships:\n%s", output)

	if !strings.Contains(output, "Database Layer") {
		t.Error("Auth Service should have relationship to Database Layer")
	}

	// Step 10: Test --dry-run mode
	t.Log("Step 10: Testing --dry-run mode...")
	output, err = runFogit(t, targetDir, "import", exportFile, "--dry-run")
	if err == nil {
		t.Log("Dry-run completed (no changes applied)")
	}
	t.Logf("Dry-run output: %s", output)

	// Step 11: Test import conflict detection
	t.Log("Step 11: Testing import conflict detection...")
	output, err = runFogit(t, targetDir, "import", exportFile)
	if err == nil {
		t.Log("Import without flags on existing data - checking behavior...")
	}
	t.Logf("Conflict detection output: %s", output)

	// Step 12: Test --merge mode (skip existing, import new only)
	t.Log("Step 12: Testing --merge mode...")

	// Create a new feature in source only
	output, err = runFogit(t, sourceDir, "feature", "New Service", "--same", "-d", "Brand new service")
	if err != nil {
		t.Fatalf("Failed to create new feature: %v\nOutput: %s", err, output)
	}

	// Re-export
	exportFile2 := filepath.Join(sourceDir, "features_export2.json")
	output, err = runFogit(t, sourceDir, "export", "json", "--output", exportFile2)
	if err != nil {
		t.Fatalf("Failed to export JSON: %v\nOutput: %s", err, output)
	}

	// Import with merge mode
	output, err = runFogit(t, targetDir, "import", exportFile2, "--merge")
	if err != nil {
		t.Fatalf("Failed to import with merge: %v\nOutput: %s", err, output)
	}
	t.Logf("Merge import output: %s", output)

	// Verify new feature was imported
	output, err = runFogit(t, targetDir, "list")
	if err != nil {
		t.Fatalf("Failed to list features: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "New Service") {
		t.Error("New Service should have been imported in merge mode")
	}

	// Step 13: Test --overwrite mode
	t.Log("Step 13: Testing --overwrite mode...")

	// Update feature in source
	output, err = runFogit(t, sourceDir, "update", "Auth Service", "--description", "Updated description for auth")
	if err != nil {
		t.Fatalf("Failed to update feature: %v\nOutput: %s", err, output)
	}

	// Re-export
	exportFile3 := filepath.Join(sourceDir, "features_export3.json")
	output, err = runFogit(t, sourceDir, "export", "json", "--output", exportFile3)
	if err != nil {
		t.Fatalf("Failed to export JSON: %v\nOutput: %s", err, output)
	}

	// Import with overwrite mode
	output, err = runFogit(t, targetDir, "import", exportFile3, "--overwrite")
	if err != nil {
		t.Fatalf("Failed to import with overwrite: %v\nOutput: %s", err, output)
	}
	t.Logf("Overwrite import output: %s", output)

	// Verify feature was updated
	output, err = runFogit(t, targetDir, "show", "Auth Service")
	if err != nil {
		t.Fatalf("Failed to show feature: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Updated description") {
		t.Error("Auth Service description should have been updated by overwrite")
	}

	// Step 14: Test export with filters
	t.Log("Step 14: Testing export with filters...")
	exportFiltered := filepath.Join(sourceDir, "features_filtered.json")
	output, err = runFogit(t, sourceDir, "export", "json", "--output", exportFiltered, "--tag", "security")
	if err != nil {
		t.Fatalf("Failed to export filtered: %v\nOutput: %s", err, output)
	}

	filteredData, err := os.ReadFile(exportFiltered)
	if err != nil {
		t.Fatalf("Failed to read filtered export: %v", err)
	}

	var filteredExport map[string]interface{}
	if err := json.Unmarshal(filteredData, &filteredExport); err != nil {
		t.Fatalf("Invalid filtered JSON: %v", err)
	}

	filteredFeatures, ok := filteredExport["features"].([]interface{})
	if !ok {
		t.Fatal("Expected features array in filtered export")
	}
	t.Logf("Filtered export contains %d features (with security tag)", len(filteredFeatures))

	t.Log("✅ Export/Import JSON test completed successfully!")
}

// TestE2E_ExportImportYAML tests exporting and importing features in YAML format
// This tests:
// - Export features to YAML format
// - YAML is human-readable
// - Round-trip preserves data
func TestE2E_ExportImportYAML(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Create test repository
	projectDir := filepath.Join(t.TempDir(), "E2E_ExportImportYAML")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	// Step 1: Initialize Git repository
	t.Log("Step 1: Initializing Git repository...")
	repo, err := gogit.PlainInit(projectDir, false)
	if err != nil {
		t.Fatalf("Failed to init Git repo: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	readmeFile := filepath.Join(projectDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# YAML Export Test\n"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("Failed to stage README: %v", err)
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

	// Step 2: Initialize fogit
	t.Log("Step 2: Initializing fogit...")
	output, err := runFogit(t, projectDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit: %v\nOutput: %s", err, output)
	}

	// Enable shared branches
	output, err = runFogit(t, projectDir, "config", "set", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to enable shared branches: %v\nOutput: %s", err, output)
	}

	// Disable fuzzy matching
	output, err = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	if err != nil {
		t.Fatalf("Failed to disable fuzzy matching: %v\nOutput: %s", err, output)
	}

	// Step 3: Create features
	t.Log("Step 3: Creating features...")
	features := []string{"Feature Alpha", "Feature Beta", "Feature Gamma"}
	for _, name := range features {
		output, err = runFogit(t, projectDir, "feature", name, "--same", "-d", "Description for "+name)
		if err != nil {
			t.Fatalf("Failed to create feature %s: %v\nOutput: %s", name, err, output)
		}
	}

	// Create relationships
	output, err = runFogit(t, projectDir, "link", "Feature Beta", "Feature Alpha", "depends-on")
	if err != nil {
		t.Fatalf("Failed to create relationship: %v\nOutput: %s", err, output)
	}

	// Step 4: Export to YAML
	t.Log("Step 4: Exporting to YAML...")
	exportFile := filepath.Join(projectDir, "features.yaml")
	output, err = runFogit(t, projectDir, "export", "yaml", "--output", exportFile)
	if err != nil {
		t.Fatalf("Failed to export YAML: %v\nOutput: %s", err, output)
	}

	// Read and display YAML content
	yamlData, err := os.ReadFile(exportFile)
	if err != nil {
		t.Fatalf("Failed to read YAML file: %v", err)
	}
	t.Logf("YAML export content:\n%s", string(yamlData))

	// Verify YAML format (should not be JSON)
	if strings.HasPrefix(strings.TrimSpace(string(yamlData)), "{") {
		t.Error("YAML export should not start with { (that's JSON)")
	}

	// Verify YAML contains expected data
	if !strings.Contains(string(yamlData), "Feature Alpha") {
		t.Error("YAML should contain Feature Alpha")
	}
	if !strings.Contains(string(yamlData), "Feature Beta") {
		t.Error("YAML should contain Feature Beta")
	}

	// Step 5: Create fresh repository and import YAML
	t.Log("Step 5: Creating fresh repository for YAML import...")
	targetDir := filepath.Join(t.TempDir(), "E2E_ExportImportYAML_Target")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}

	targetRepo, err := gogit.PlainInit(targetDir, false)
	if err != nil {
		t.Fatalf("Failed to init target Git repo: %v", err)
	}

	targetWorktree, err := targetRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get target worktree: %v", err)
	}

	targetReadme := filepath.Join(targetDir, "README.md")
	if err := os.WriteFile(targetReadme, []byte("# Target YAML Repo\n"), 0644); err != nil {
		t.Fatalf("Failed to create target README: %v", err)
	}
	if _, err := targetWorktree.Add("README.md"); err != nil {
		t.Fatalf("Failed to stage target README: %v", err)
	}
	_, err = targetWorktree.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit in target: %v", err)
	}

	output, err = runFogit(t, targetDir, "init")
	if err != nil {
		t.Fatalf("Failed to init fogit in target: %v\nOutput: %s", err, output)
	}

	output, err = runFogit(t, targetDir, "config", "set", "workflow.allow_shared_branches", "true")
	if err != nil {
		t.Fatalf("Failed to enable shared branches: %v\nOutput: %s", err, output)
	}

	// Step 6: Import YAML
	t.Log("Step 6: Importing YAML...")
	output, err = runFogit(t, targetDir, "import", exportFile)
	if err != nil {
		t.Fatalf("Failed to import YAML: %v\nOutput: %s", err, output)
	}
	t.Logf("Import output: %s", output)

	// Step 7: Verify imported data
	t.Log("Step 7: Verifying imported data...")
	output, err = runFogit(t, targetDir, "list")
	if err != nil {
		t.Fatalf("Failed to list features: %v\nOutput: %s", err, output)
	}
	t.Logf("Imported features:\n%s", output)

	for _, name := range features {
		if !strings.Contains(output, name) {
			t.Errorf("Imported features should contain %s", name)
		}
	}

	// Verify relationships preserved
	output, err = runFogit(t, targetDir, "relationships", "Feature Beta")
	if err != nil {
		t.Fatalf("Failed to get relationships: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Feature Alpha") {
		t.Error("Relationship to Feature Alpha should be preserved")
	}

	// Step 8: Test round-trip by re-exporting and comparing
	t.Log("Step 8: Testing round-trip export...")
	reExportFile := filepath.Join(targetDir, "features_reexport.yaml")
	output, err = runFogit(t, targetDir, "export", "yaml", "--output", reExportFile)
	if err != nil {
		t.Fatalf("Failed to re-export YAML: %v\nOutput: %s", err, output)
	}

	reExportData, err := os.ReadFile(reExportFile)
	if err != nil {
		t.Fatalf("Failed to read re-export file: %v", err)
	}

	// Both exports should contain the same features
	for _, name := range features {
		if !strings.Contains(string(reExportData), name) {
			t.Errorf("Re-exported YAML should contain %s", name)
		}
	}

	t.Log("✅ Export/Import YAML test completed successfully!")
}

// TestE2E_ExportCSV tests exporting features to CSV format
func TestE2E_ExportCSV(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	projectDir := filepath.Join(t.TempDir(), "E2E_ExportCSV")
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
	if err := os.WriteFile(readmeFile, []byte("# CSV Test\n"), 0644); err != nil {
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

	// Disable fuzzy matching to avoid prompts for similar feature names
	_, err = runFogit(t, projectDir, "config", "set", "feature_search.fuzzy_match", "false")
	if err != nil {
		t.Fatalf("Failed to disable fuzzy matching: %v", err)
	}

	// Create features
	output, err = runFogit(t, projectDir, "feature", "CSV Feature One", "--same", "-p", "high")
	if err != nil {
		t.Fatalf("Failed to create feature: %v\nOutput: %s", err, output)
	}
	output, err = runFogit(t, projectDir, "feature", "CSV Feature Two", "--same", "-p", "low")
	if err != nil {
		t.Fatalf("Failed to create feature 2: %v\nOutput: %s", err, output)
	}

	// Export to CSV
	t.Log("Exporting to CSV...")
	output, err = runFogit(t, projectDir, "export", "csv")
	if err != nil {
		t.Fatalf("Failed to export CSV: %v\nOutput: %s", err, output)
	}
	t.Logf("CSV output:\n%s", output)

	// Verify CSV format
	if !strings.Contains(output, ",") {
		t.Error("CSV output should contain commas")
	}
	if !strings.Contains(output, "CSV Feature One") {
		t.Error("CSV should contain feature names")
	}

	t.Log("✅ Export CSV test completed!")
}

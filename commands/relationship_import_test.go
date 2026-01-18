package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestRelationshipImportCommand(t *testing.T) {
	// Setup test directory with fogit repo
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize with default config
	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Create import file with a new custom type and category
	importData := RelationshipExport{
		FogitVersion: "1.0",
		ExportedAt:   "2025-10-30T14:30:00Z",
		RelationshipCategories: map[string]RelationshipCategoryExport{
			"custom": {
				Description:     "Custom test category",
				AllowCycles:     false,
				CycleDetection:  "strict",
				IncludeInImpact: true,
			},
		},
		RelationshipTypes: map[string]RelationshipTypeExport{
			"custom-depends": {
				Category:    "custom",
				Description: "Custom dependency type",
				Inverse:     "custom-required-by",
			},
			"custom-required-by": {
				Category:    "custom",
				Description: "Inverse of custom dependency",
				Inverse:     "custom-depends",
			},
		},
	}

	importFile := filepath.Join(tmpDir, "import.json")
	data, _ := json.MarshalIndent(importData, "", "  ")
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "import", importFile})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// Verify the config was updated
	loadedCfg, err := config.Load(fogitDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Check category was imported
	if _, exists := loadedCfg.Relationships.Categories["custom"]; !exists {
		t.Error("expected 'custom' category to be imported")
	}

	// Check types were imported
	if _, exists := loadedCfg.Relationships.Types["custom-depends"]; !exists {
		t.Error("expected 'custom-depends' type to be imported")
	}
	if _, exists := loadedCfg.Relationships.Types["custom-required-by"]; !exists {
		t.Error("expected 'custom-required-by' type to be imported")
	}
}

func TestRelationshipImportYAML(t *testing.T) {
	// Setup test directory with fogit repo
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize with default config
	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Create YAML import file
	importData := RelationshipExport{
		FogitVersion: "1.0",
		ExportedAt:   "2025-10-30T14:30:00Z",
		RelationshipCategories: map[string]RelationshipCategoryExport{
			"yaml-test": {
				Description:    "YAML test category",
				AllowCycles:    true,
				CycleDetection: "none",
			},
		},
	}

	importFile := filepath.Join(tmpDir, "import.yaml")
	data, _ := yaml.Marshal(importData)
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "import", importFile})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// Verify the config was updated
	loadedCfg, err := config.Load(fogitDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if _, exists := loadedCfg.Relationships.Categories["yaml-test"]; !exists {
		t.Error("expected 'yaml-test' category to be imported")
	}
}

func TestRelationshipImportConflictError(t *testing.T) {
	// Setup test directory with fogit repo
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize with default config
	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Try to import existing category "structural"
	importData := RelationshipExport{
		FogitVersion: "1.0",
		RelationshipCategories: map[string]RelationshipCategoryExport{
			"structural": {
				Description: "Conflicting structural category",
			},
		},
	}

	importFile := filepath.Join(tmpDir, "conflict.json")
	data, _ := json.MarshalIndent(importData, "", "  ")
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "import", importFile})
	err := ExecuteRootCmd()

	if err == nil {
		t.Error("expected error when importing conflicting category")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRelationshipImportMerge(t *testing.T) {
	// Setup test directory with fogit repo
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize with default config
	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Import with existing category + new category
	importData := RelationshipExport{
		FogitVersion: "1.0",
		RelationshipCategories: map[string]RelationshipCategoryExport{
			"structural": {
				Description: "Conflicting structural category",
			},
			"new-merge-category": {
				Description:    "New category to be added",
				CycleDetection: "strict",
			},
		},
	}

	importFile := filepath.Join(tmpDir, "merge.json")
	data, _ := json.MarshalIndent(importData, "", "  ")
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "import", importFile, "--merge"})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatalf("import with merge failed: %v", err)
	}

	// Verify config
	loadedCfg, err := config.Load(fogitDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// New category should be added
	if _, exists := loadedCfg.Relationships.Categories["new-merge-category"]; !exists {
		t.Error("expected 'new-merge-category' to be added")
	}

	// Existing category should be unchanged (not overwritten with the import's description)
	if loadedCfg.Relationships.Categories["structural"].Description == "Conflicting structural category" {
		t.Error("expected 'structural' category to keep original description")
	}
}

func TestRelationshipImportOverwrite(t *testing.T) {
	// Setup test directory with fogit repo
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize with default config
	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Import with existing category to overwrite
	importData := RelationshipExport{
		FogitVersion: "1.0",
		RelationshipCategories: map[string]RelationshipCategoryExport{
			"structural": {
				Description:    "Overwritten structural category",
				AllowCycles:    false, // Keep the same to avoid validation issues
				CycleDetection: "warn",
			},
		},
	}

	importFile := filepath.Join(tmpDir, "overwrite.json")
	data, _ := json.MarshalIndent(importData, "", "  ")
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "import", importFile, "--overwrite"})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatalf("import with overwrite failed: %v", err)
	}

	// Verify config
	loadedCfg, err := config.Load(fogitDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Category should be overwritten
	structuralCat := loadedCfg.Relationships.Categories["structural"]
	if structuralCat.Description != "Overwritten structural category" {
		t.Errorf("expected description to be overwritten, got '%s'", structuralCat.Description)
	}
	if structuralCat.CycleDetection != "warn" {
		t.Errorf("expected CycleDetection to be 'warn' after overwrite, got '%s'", structuralCat.CycleDetection)
	}
}

func TestRelationshipImportInvalidCycleDetection(t *testing.T) {
	// Setup test directory with fogit repo
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize with default config
	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Import with invalid cycle_detection
	importData := RelationshipExport{
		FogitVersion: "1.0",
		RelationshipCategories: map[string]RelationshipCategoryExport{
			"bad-cat": {
				Description:    "Bad category",
				CycleDetection: "invalid",
			},
		},
	}

	importFile := filepath.Join(tmpDir, "invalid.json")
	data, _ := json.MarshalIndent(importData, "", "  ")
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "import", importFile})
	err := ExecuteRootCmd()

	if err == nil {
		t.Error("expected error for invalid cycle_detection")
	}
	if !strings.Contains(err.Error(), "cycle_detection") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRelationshipImportInverseConsistency(t *testing.T) {
	// Setup test directory with fogit repo
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize with default config
	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Import with inconsistent inverse relationships
	importData := RelationshipExport{
		FogitVersion: "1.0",
		RelationshipCategories: map[string]RelationshipCategoryExport{
			"test-inv": {
				Description: "Test category",
			},
		},
		RelationshipTypes: map[string]RelationshipTypeExport{
			"type-a": {
				Category: "test-inv",
				Inverse:  "type-b",
			},
			"type-b": {
				Category: "test-inv",
				Inverse:  "type-c", // Inconsistent - should be type-a
			},
		},
	}

	importFile := filepath.Join(tmpDir, "inconsistent.json")
	data, _ := json.MarshalIndent(importData, "", "  ")
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "import", importFile})
	err := ExecuteRootCmd()

	if err == nil {
		t.Error("expected error for inconsistent inverse relationships")
	}
	if !strings.Contains(err.Error(), "inconsistent") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRelationshipImportMissingCategory(t *testing.T) {
	// Setup test directory with fogit repo
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize with minimal config (no categories)
	cfg := &fogit.Config{
		Repository: fogit.RepositoryConfig{
			Version: "1.0",
		},
		Relationships: fogit.RelationshipsConfig{
			Categories: make(map[string]fogit.RelationshipCategory),
			Types:      make(map[string]fogit.RelationshipTypeConfig),
		},
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Import type that references non-existent category
	importData := RelationshipExport{
		FogitVersion: "1.0",
		RelationshipTypes: map[string]RelationshipTypeExport{
			"orphan-type": {
				Category:    "non-existent-category",
				Description: "Type with missing category",
			},
		},
	}

	importFile := filepath.Join(tmpDir, "missing-cat.json")
	data, _ := json.MarshalIndent(importData, "", "  ")
	if err := os.WriteFile(importFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "import", importFile})
	err := ExecuteRootCmd()

	if err == nil {
		t.Error("expected error for missing category reference")
	}
	if !strings.Contains(err.Error(), "unknown category") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRelationshipImportConflictingFlags(t *testing.T) {
	// Setup test directory with fogit repo
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize with default config
	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Create dummy import file
	importFile := filepath.Join(tmpDir, "dummy.json")
	if err := os.WriteFile(importFile, []byte(`{"fogit_version": "1.0"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "import", importFile, "--merge", "--overwrite"})
	err := ExecuteRootCmd()

	if err == nil {
		t.Error("expected error when both --merge and --overwrite are set")
	}
	if !strings.Contains(err.Error(), "cannot specify both") {
		t.Errorf("unexpected error message: %v", err)
	}
}

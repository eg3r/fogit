package commands

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestRelationshipExportCommand(t *testing.T) {
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

	// Create a feature to ensure repo is valid
	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		format      string
		outputFile  string
		typesOnly   bool
		catsOnly    bool
		wantErr     bool
		checkOutput func(t *testing.T, data []byte)
	}{
		{
			name:       "export json default to file",
			format:     "",
			outputFile: "export.json",
			wantErr:    false,
			checkOutput: func(t *testing.T, data []byte) {
				var export RelationshipExport
				if err := json.Unmarshal(data, &export); err != nil {
					t.Errorf("invalid JSON output: %v", err)
					return
				}
				if export.FogitVersion != "1.0" {
					t.Errorf("expected fogit_version 1.0, got %s", export.FogitVersion)
				}
				if export.ExportedAt == "" {
					t.Error("expected exported_at to be set")
				}
				if len(export.RelationshipCategories) == 0 {
					t.Error("expected relationship_categories to be present")
				}
				if len(export.RelationshipTypes) == 0 {
					t.Error("expected relationship_types to be present")
				}
			},
		},
		{
			name:       "export json explicit to file",
			format:     "json",
			outputFile: "explicit.json",
			wantErr:    false,
			checkOutput: func(t *testing.T, data []byte) {
				var export RelationshipExport
				if err := json.Unmarshal(data, &export); err != nil {
					t.Errorf("invalid JSON output: %v", err)
				}
			},
		},
		{
			name:       "export yaml to file",
			format:     "yaml",
			outputFile: "export.yaml",
			wantErr:    false,
			checkOutput: func(t *testing.T, data []byte) {
				var export RelationshipExport
				if err := yaml.Unmarshal(data, &export); err != nil {
					t.Errorf("invalid YAML output: %v", err)
					return
				}
				if export.FogitVersion != "1.0" {
					t.Errorf("expected fogit_version 1.0, got %s", export.FogitVersion)
				}
				if len(export.RelationshipCategories) == 0 {
					t.Error("expected relationship_categories to be present")
				}
				if len(export.RelationshipTypes) == 0 {
					t.Error("expected relationship_types to be present")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, tt.outputFile)

			// Build command
			cmdArgs := []string{"-C", tmpDir, "relationship", "export"}
			if tt.format != "" {
				cmdArgs = append(cmdArgs, tt.format)
			}
			cmdArgs = append(cmdArgs, "--output", outputPath)

			ResetFlags()
			rootCmd.SetArgs(cmdArgs)
			err := ExecuteRootCmd()

			if (err != nil) != tt.wantErr {
				t.Errorf("got error = %v, wantErr = %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkOutput != nil {
				data, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("failed to read output file: %v", err)
				}
				tt.checkOutput(t, data)
			}
		})
	}
}

func TestRelationshipExportInvalidFormat(t *testing.T) {
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

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "export", "xml"})
	err := ExecuteRootCmd()

	if err == nil {
		t.Error("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRelationshipExportTypesOnly(t *testing.T) {
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

	outputFile := filepath.Join(tmpDir, "types-only.json")

	// Execute with --types-only
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "export", "--types-only", "--output", outputFile})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var export RelationshipExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(export.RelationshipCategories) != 0 {
		t.Error("expected no relationship_categories when --types-only is set")
	}
	if len(export.RelationshipTypes) == 0 {
		t.Error("expected relationship_types to be present")
	}
}

func TestRelationshipExportCategoriesOnly(t *testing.T) {
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

	outputFile := filepath.Join(tmpDir, "categories-only.json")

	// Execute with --categories-only
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "export", "--categories-only", "--output", outputFile})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var export RelationshipExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if len(export.RelationshipCategories) == 0 {
		t.Error("expected relationship_categories to be present")
	}
	if len(export.RelationshipTypes) != 0 {
		t.Error("expected no relationship_types when --categories-only is set")
	}
}

func TestRelationshipExportToFile(t *testing.T) {
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

	outputFile := filepath.Join(tmpDir, "export.json")

	// Execute with --output
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "export", "--output", outputFile})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatal(err)
	}

	// Check file exists and is valid
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var export RelationshipExport
	if err := json.Unmarshal(data, &export); err != nil {
		t.Fatalf("invalid JSON in file: %v", err)
	}

	if export.FogitVersion != "1.0" {
		t.Errorf("expected fogit_version 1.0, got %s", export.FogitVersion)
	}
}

func TestRelationshipExportConflictingFlags(t *testing.T) {
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

	// Execute with both --types-only and --categories-only
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "export", "--types-only", "--categories-only"})
	err := ExecuteRootCmd()

	if err == nil {
		t.Error("expected error when both --types-only and --categories-only are set")
	}
	if !strings.Contains(err.Error(), "cannot specify both") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRelationshipExportYAMLFormat(t *testing.T) {
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

	outputFile := filepath.Join(tmpDir, "export.yaml")

	// Execute with yaml format and output
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "relationship", "export", "yaml", "--output", outputFile})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatal(err)
	}

	// Check file exists and is valid YAML
	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var export RelationshipExport
	if err := yaml.Unmarshal(data, &export); err != nil {
		t.Fatalf("invalid YAML in file: %v", err)
	}

	if export.FogitVersion != "1.0" {
		t.Errorf("expected fogit_version 1.0, got %s", export.FogitVersion)
	}
}

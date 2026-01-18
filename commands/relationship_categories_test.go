package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestRelationshipCategoriesCommand(t *testing.T) {
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
		name           string
		verbose        bool
		wantCategories []string
		wantDetails    []string // Details that should appear in verbose mode
	}{
		{
			name:           "list all categories compact",
			verbose:        false,
			wantCategories: []string{"structural", "informational", "workflow", "compliance"},
		},
		{
			name:           "verbose mode shows settings",
			verbose:        true,
			wantCategories: []string{"structural", "informational", "workflow", "compliance"},
			wantDetails: []string{
				"Types:",
				"Cycle Detection:",
				"Include in Impact:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Build args with -C flag
			args := []string{"-C", tmpDir, "categories"}
			if tt.verbose {
				args = append(args, "--verbose")
			}

			// Reset flags and run with -C flag
			ResetFlags()
			rootCmd.SetArgs(args)
			err := ExecuteRootCmd()

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("relationship-categories command error = %v", err)
			}

			// Check expected categories appear
			for _, wantCat := range tt.wantCategories {
				if !strings.Contains(output, wantCat) {
					t.Errorf("Output missing expected category %q:\n%s", wantCat, output)
				}
			}

			// Check details appear in verbose mode
			if tt.verbose {
				for _, detail := range tt.wantDetails {
					if !strings.Contains(output, detail) {
						t.Errorf("Verbose output missing detail %q:\n%s", detail, output)
					}
				}
			}
		})
	}
}

func TestRelationshipCategoriesCommand_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with no categories
	cfg := &fogit.Config{
		Relationships: fogit.RelationshipsConfig{
			Types:      make(map[string]fogit.RelationshipTypeConfig),
			Categories: make(map[string]fogit.RelationshipCategory),
		},
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Should not error with empty config: %v", err)
	}

	if !strings.Contains(output, "No relationship categories defined") {
		t.Errorf("Expected 'No relationship categories defined' message, got:\n%s", output)
	}
}

func TestRelationshipCategoriesCommand_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create .fogit directory at all

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories"})
	err := ExecuteRootCmd()

	if err == nil {
		t.Fatal("Expected error when .fogit directory doesn't exist")
	}

	if !strings.Contains(err.Error(), "failed to get .fogit directory") {
		t.Errorf("Expected 'failed to get .fogit directory' error, got: %v", err)
	}
}

func TestRelationshipCategoriesCommand_TypeCounting(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with specific type counts per category
	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("relationship-categories command error = %v", err)
	}

	// Verify type counts are shown
	// Default config has: structural (3 types), informational (3 types), workflow (2 types), compliance (2 types)
	if !strings.Contains(output, "types)") {
		t.Errorf("Expected type counts in output, got:\n%s", output)
	}
}

func TestRelationshipCategoriesCommand_Aliases(t *testing.T) {
	// Test that command aliases work
	if len(relationshipCategoriesCmd.Aliases) == 0 {
		t.Error("Expected relationshipCategoriesCmd to have aliases")
	}

	expectedAliases := []string{"category", "cats"}
	for _, expected := range expectedAliases {
		found := false
		for _, alias := range relationshipCategoriesCmd.Aliases {
			if alias == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected %q to be an alias for relationshipCategoriesCmd, got: %v", expected, relationshipCategoriesCmd.Aliases)
		}
	}
}

func TestRelationshipCategoriesCommand_CycleDetectionModes(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Reset flags and run with -C flag (verbose mode)
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories", "--verbose"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("relationship-categories command error = %v", err)
	}

	// Verify cycle detection modes are shown
	cycleDetectionModes := []string{"strict", "warn", "none"}
	foundMode := false
	for _, mode := range cycleDetectionModes {
		if strings.Contains(output, mode) {
			foundMode = true
			break
		}
	}
	if !foundMode {
		t.Errorf("Expected at least one cycle detection mode in verbose output, got:\n%s", output)
	}
}

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

func TestRelationshipTypesCommand(t *testing.T) {
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
		category       string
		verbose        bool
		wantTypes      []string // Types that should appear in output
		wantNotTypes   []string // Types that should NOT appear
		wantCategories []string // Categories that should appear
	}{
		{
			name:           "list all types compact",
			category:       "",
			verbose:        false,
			wantTypes:      []string{"depends-on", "blocks", "related-to", "contains", "implements"},
			wantCategories: []string{"structural", "informational", "workflow"},
		},
		{
			name:         "filter by structural category",
			category:     "structural",
			verbose:      false,
			wantTypes:    []string{"depends-on", "contains", "implements"},
			wantNotTypes: []string{"relates-to", "documents"},
		},
		{
			name:         "filter by informational category",
			category:     "informational",
			verbose:      false,
			wantTypes:    []string{"related-to", "references", "conflicts-with"},
			wantNotTypes: []string{"depends-on", "blocks"},
		},
		{
			name:         "filter by workflow category",
			category:     "workflow",
			verbose:      false,
			wantTypes:    []string{"blocks", "blocked-by"},
			wantNotTypes: []string{"depends-on", "relates-to"},
		},
		{
			name:         "filter by compliance category",
			category:     "compliance",
			verbose:      false,
			wantTypes:    []string{}, // Empty - compliance category exists but has no types yet
			wantNotTypes: []string{"depends-on", "blocks"},
		},
		{
			name:     "verbose mode shows details",
			category: "structural",
			verbose:  true,
			wantTypes: []string{
				"depends-on",
				"Inverse:",     // Should show inverse field
				"Aliases:",     // Should show aliases
				"Description:", // Should show description
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build args with -C flag
			args := []string{"-C", tmpDir, "types"}
			if tt.category != "" {
				args = append(args, "--category", tt.category)
			}
			if tt.verbose {
				args = append(args, "--verbose")
			}

			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

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
				t.Fatalf("relationship-types command error = %v", err)
			}

			// Check expected types appear
			for _, wantType := range tt.wantTypes {
				if !strings.Contains(output, wantType) {
					t.Errorf("Output missing expected type %q:\n%s", wantType, output)
				}
			}

			// Check unwanted types don't appear
			for _, notType := range tt.wantNotTypes {
				if strings.Contains(output, notType) {
					t.Errorf("Output contains unwanted type %q:\n%s", notType, output)
				}
			}

			// Check categories appear (for compact mode without filter)
			if !tt.verbose && tt.category == "" {
				for _, cat := range tt.wantCategories {
					if !strings.Contains(output, cat) {
						t.Errorf("Output missing category %q:\n%s", cat, output)
					}
				}
			}
		})
	}
}

func TestRelationshipTypesCommand_InvalidCategory(t *testing.T) {
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

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "types", "--category", "nonexistent"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Should not error on invalid category filter: %v", err)
	}

	// Should show "No types found" message
	if !strings.Contains(output, "No relationship types found") {
		t.Errorf("Expected 'No relationship types found' message for invalid category, got:\n%s", output)
	}
}

func TestRelationshipTypesCommand_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create .fogit directory at all

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "types"})
	err := ExecuteRootCmd()

	if err == nil {
		t.Fatal("Expected error when .fogit directory doesn't exist")
	}

	if !strings.Contains(err.Error(), "failed to get .fogit directory") {
		t.Errorf("Expected 'failed to get .fogit directory' error, got: %v", err)
	}
}

func TestRelationshipTypesCommand_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with no types
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
	rootCmd.SetArgs([]string{"-C", tmpDir, "types"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("Should not error with empty config: %v", err)
	}

	if !strings.Contains(output, "No relationship types defined") {
		t.Errorf("Expected 'No relationship types defined' message, got:\n%s", output)
	}
}

func TestRelationshipTypesCommand_Aliases(t *testing.T) {
	// Test that command aliases work
	if len(relationshipTypesCmd.Aliases) == 0 {
		t.Error("Expected relationshipTypesCmd to have aliases")
	}

	found := false
	for _, alias := range relationshipTypesCmd.Aliases {
		if alias == "type" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'type' to be an alias for relationshipTypesCmd, got: %v", relationshipTypesCmd.Aliases)
	}
}

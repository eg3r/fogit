package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// TestDiffCommandArgs tests argument validation for diff command
func TestDiffCommandArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "one argument - feature name",
			args:    []string{"Feature Name"},
			wantErr: false,
		},
		{
			name:    "two arguments - feature and version",
			args:    []string{"Feature Name", "1"},
			wantErr: false,
		},
		{
			name:    "three arguments - feature and two versions",
			args:    []string{"Feature Name", "1", "2"},
			wantErr: false,
		},
		{
			name:    "four arguments - too many",
			args:    []string{"Feature1", "1", "2", "3"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := diffCmd.Args(diffCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDiffCommandFlags tests that diff command flags are defined correctly
func TestDiffCommandFlags(t *testing.T) {
	if diffCmd.Flags().Lookup("format") == nil {
		t.Error("--format flag not defined")
	}
}

// TestDiffCommand_VersionDiff tests diff command comparing two versions
func TestDiffCommand_VersionDiff(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create a feature with multiple versions
	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Multi-Version Feature")
	feature.Description = "Initial description"
	feature.SetPriority(fogit.PriorityLow)

	// Manually add version 2
	now := time.Now()
	feature.Versions["2"] = &fogit.FeatureVersion{
		CreatedAt:  now,
		ModifiedAt: now,
		Notes:      "Version 2",
	}
	// Modify version 1 to have different timestamp
	feature.Versions["1"].Notes = "Version 1"

	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "diff", feature.ID, "1", "2"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("diff command failed: %v", err)
	}
}

// TestDiffCommand_NonExistent tests diff for non-existent feature
func TestDiffCommand_NonExistent(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "diff", "nonexistent-feature"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Error("expected error when diffing non-existent feature")
	}
}

// TestDiffCommand_InvalidVersion tests diff with invalid version
func TestDiffCommand_InvalidVersion(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create a feature
	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	feature.SetPriority(fogit.PriorityMedium)
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "diff", feature.ID, "999"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Error("expected error for invalid version")
	}
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "invalid") {
		t.Logf("error was: %v", err) // Log for debugging, don't fail if error message differs
	}
}

// Note: Business logic tests for CalculateVersionDiff are in internal/features/diff_test.go
// Domain model tests for GetSortedVersionKeys are in pkg/fogit/feature_test.go

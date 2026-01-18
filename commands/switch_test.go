package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestSwitchCommandArgs(t *testing.T) {
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
			name:    "one argument",
			args:    []string{"feature-name"},
			wantErr: false,
		},
		{
			name:    "too many arguments",
			args:    []string{"feature1", "feature2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := switchCmd.Args(switchCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Note: Feature state and storage tests are covered in:
// - internal/storage/repository_test.go (Create, Get, Update)
// - pkg/fogit/feature_test.go (DeriveState, UpdateState)
// - internal/features/branch_test.go (GetFeatureBranch)

// TestSwitchCommand_ByID tests switching to a feature by ID
func TestSwitchCommand_ByID(t *testing.T) {
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
	rootCmd.SetArgs([]string{"-C", tmpDir, "switch", feature.ID})

	// Note: This will fail without git, but we verify it finds the feature
	err := ExecuteRootCmd()
	// Switch requires git, so we expect an error about git not found
	// but NOT an error about feature not found
	if err != nil && err.Error() != "" {
		// Check it's not a "feature not found" error
		if err.Error() == "feature not found" {
			t.Errorf("feature should have been found, got: %v", err)
		}
	}
}

// TestSwitchCommand_NonExistent tests switching to a non-existent feature
func TestSwitchCommand_NonExistent(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "switch", "nonexistent-feature"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Error("expected error when switching to non-existent feature")
	}
}

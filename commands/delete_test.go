package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// TestDeleteCommandArgs tests that the delete command validates arguments correctly.
// This is a command-level test for CLI argument validation.
func TestDeleteCommandArgs(t *testing.T) {
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
			args:    []string{"Feature Name"},
			wantErr: false,
		},
		{
			name:    "too many arguments",
			args:    []string{"Feature1", "Feature2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := deleteCmd.Args(deleteCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDeleteCommandFlags tests that delete command flags are defined correctly.
func TestDeleteCommandFlags(t *testing.T) {
	if deleteCmd.Flags().Lookup("force") == nil {
		t.Error("--force flag not defined")
	}
}

// TestDeleteCommand_Force tests the delete command with --force flag
func TestDeleteCommand_Force(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create a feature to delete
	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature to Delete")
	feature.SetPriority(fogit.PriorityMedium)
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}

	// Verify feature exists
	_, err := repo.Get(context.Background(), feature.ID)
	if err != nil {
		t.Fatalf("feature should exist before delete: %v", err)
	}

	// Reset flags and execute delete command with -C flag
	ResetFlags()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"-C", tmpDir, "delete", feature.ID, "--force"})

	err = ExecuteRootCmd()
	if err != nil {
		t.Fatalf("delete command failed: %v", err)
	}

	// Verify feature is deleted
	_, err = repo.Get(context.Background(), feature.ID)
	if err != fogit.ErrNotFound {
		t.Errorf("feature should be deleted, got error: %v", err)
	}
}

// TestDeleteCommand_NonExistent tests deleting a non-existent feature
func TestDeleteCommand_NonExistent(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Reset flags and execute delete command for non-existent feature with -C flag
	ResetFlags()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetArgs([]string{"-C", tmpDir, "delete", "nonexistent-feature", "--force"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Error("expected error when deleting non-existent feature")
	}
}

// Note: Business logic tests for:
// - storage.Repository.Delete() are in internal/storage/repository_test.go
// - features.Find() (case-insensitive, by name/ID) are in internal/features/finder_test.go
// - features.FindIncomingRelationships() are in internal/features/relationships_test.go
// - features.CleanupIncomingRelationships() are in internal/features/relationships_test.go

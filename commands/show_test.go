package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// TestShowCommandArgs tests argument validation
func TestShowCommandArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no arguments",
			args:    []string{},
			wantErr: true,
			errMsg:  "accepts 1 arg",
		},
		{
			name:    "one argument",
			args:    []string{"User Authentication"},
			wantErr: false,
		},
		{
			name:    "too many arguments",
			args:    []string{"Feature1", "Feature2"},
			wantErr: true,
			errMsg:  "accepts 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := showCmd.Args(showCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Args() error = %q, want error containing %q", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestShowCommandFlags tests that show command flags are defined correctly
func TestShowCommandFlags(t *testing.T) {
	flags := []string{"format", "versions", "relationships"}
	for _, flag := range flags {
		t.Run("has "+flag+" flag", func(t *testing.T) {
			if showCmd.Flags().Lookup(flag) == nil {
				t.Errorf("--%s flag not defined", flag)
			}
		})
	}
}

// TestShowCommand_TextFormat tests show command with default text format
func TestShowCommand_TextFormat(t *testing.T) {
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
	feature.Description = "Test description"
	feature.SetPriority(fogit.PriorityHigh)
	feature.SetCategory("testing")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "show", feature.ID})

	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("show command failed: %v", err)
	}
}

// TestShowCommand_JSONFormat tests show command with JSON format
func TestShowCommand_JSONFormat(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create a feature
	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("JSON Test Feature")
	feature.SetPriority(fogit.PriorityMedium)
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "show", feature.ID, "--format", "json"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("show command failed: %v", err)
	}
}

// TestShowCommand_NonExistent tests showing a non-existent feature
func TestShowCommand_NonExistent(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "show", "nonexistent-feature"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Error("expected error when showing non-existent feature")
	}
}

// TestShowCommand_InvalidFormat tests show command with invalid format
func TestShowCommand_InvalidFormat(t *testing.T) {
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
	rootCmd.SetArgs([]string{"-C", tmpDir, "show", feature.ID, "--format", "invalid"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Error("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("error should mention invalid format, got: %v", err)
	}
}

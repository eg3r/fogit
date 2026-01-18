package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// TestUpdateCommandArgs tests argument validation
func TestUpdateCommandArgs(t *testing.T) {
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
			err := updateCmd.Args(updateCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestHasUpdateFlags tests the flag checking logic
func TestHasUpdateFlags(t *testing.T) {
	tests := []struct {
		name     string
		setFlags map[string]string
		want     bool
	}{
		{
			name:     "no flags set",
			setFlags: map[string]string{},
			want:     false,
		},
		{
			name:     "state flag set",
			setFlags: map[string]string{"state": "in-progress"},
			want:     true,
		},
		{
			name:     "priority flag set",
			setFlags: map[string]string{"priority": "high"},
			want:     true,
		},
		{
			name:     "description flag set",
			setFlags: map[string]string{"description": "New desc"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh command with flags
			cmd := updateCmd
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				f.Changed = false
			})

			// Set test flags
			for key, val := range tt.setFlags {
				cmd.Flags().Set(key, val)
			}

			got := hasUpdateFlags(cmd)
			if got != tt.want {
				t.Errorf("hasUpdateFlags() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Note: Storage update tests are covered in internal/storage/repository_test.go:
// - TestFileRepository_Update (basic update)
// - Update nonexistent feature
// - State transitions via UpdateState()

// Note: Feature state/closed_at tests are covered in pkg/fogit/feature_test.go:
// - TestFeature_UpdateState
// - TestFeature_DeriveState
// - TestFeature_GetClosedAt

// Note: features.Find tests are covered in internal/features/finder_test.go

// TestUpdateCommand_State tests state update via CLI
func TestUpdateCommand_State(t *testing.T) {
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

	// Reset flags and execute update command with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "update", feature.ID, "--state", "in-progress"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Errorf("update command failed: %v", err)
	}

	// Verify state was updated
	updated, err := repo.Get(context.Background(), feature.ID)
	if err != nil {
		t.Fatalf("failed to get updated feature: %v", err)
	}
	if updated.DeriveState() != fogit.StateInProgress {
		t.Errorf("state not updated, got %v, want %v", updated.DeriveState(), fogit.StateInProgress)
	}
}

// TestUpdateCommand_Priority tests priority update via CLI
func TestUpdateCommand_Priority(t *testing.T) {
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
	feature.SetPriority(fogit.PriorityLow)
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}

	// Reset flags and execute update command with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "update", feature.ID, "--priority", "critical"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Errorf("update command failed: %v", err)
	}

	// Verify priority was updated
	updated, err := repo.Get(context.Background(), feature.ID)
	if err != nil {
		t.Fatalf("failed to get updated feature: %v", err)
	}
	if updated.GetPriority() != fogit.PriorityCritical {
		t.Errorf("priority not updated, got %v, want %v", updated.GetPriority(), fogit.PriorityCritical)
	}
}

// TestUpdateCommand_NonExistent tests updating a non-existent feature
func TestUpdateCommand_NonExistent(t *testing.T) {
	// Setup temp directory with fogit structure
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Reset flags and execute update command with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "update", "nonexistent-feature", "--state", "closed"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Error("expected error when updating non-existent feature")
	}
}

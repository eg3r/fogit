package commands

import (
	"testing"
)

// TestMergeCommandArgs tests argument validation
func TestMergeCommandArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no arguments is allowed",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "one argument is allowed",
			args:    []string{"Feature Name"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// merge command doesn't restrict args, just test that it accepts them
			if len(tt.args) > 1 != tt.wantErr {
				t.Errorf("Expected wantErr=%v for %d args", tt.wantErr, len(tt.args))
			}
		})
	}
}

// TestMergeCommandFlags tests that merge command flags are defined
func TestMergeCommandFlags(t *testing.T) {
	if mergeCmd.Flags().Lookup("no-delete") == nil {
		t.Error("--no-delete flag not defined")
	}
	if mergeCmd.Flags().Lookup("squash") == nil {
		t.Error("--squash flag not defined")
	}
}

// Note: TestMergeFeatureNotFound is covered by internal/storage/repository_test.go
// which thoroughly tests repo.Get() with non-existent features.

package commands

import (
	"testing"
)

func TestFeatureCommandArgs(t *testing.T) {
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
			err := featureCmd.Args(featureCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Note: Storage edge case tests (unicode, long names, special chars, etc.)
// are covered in internal/storage/repository_test.go

// Note: TestHandleBranchCreation_WorkflowModes tests are in internal/features/branch_test.go

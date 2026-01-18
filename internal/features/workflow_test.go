package features

import (
	"testing"
)

func TestDetermineBranchAction(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		allowShared bool
		same        bool
		isolate     bool
		wantAction  BranchAction
		wantErr     bool
	}{
		{
			name:        "trunk-based default",
			mode:        "trunk-based",
			allowShared: true,
			same:        false,
			isolate:     false,
			wantAction:  BranchActionNone,
			wantErr:     false,
		},
		{
			name:        "trunk-based with same (error)",
			mode:        "trunk-based",
			allowShared: true,
			same:        true,
			isolate:     false,
			wantAction:  BranchActionNone,
			wantErr:     true,
		},
		{
			name:        "branch-per-feature default",
			mode:        "branch-per-feature",
			allowShared: true,
			same:        false,
			isolate:     false,
			wantAction:  BranchActionCreate,
			wantErr:     false,
		},
		{
			name:        "branch-per-feature with same",
			mode:        "branch-per-feature",
			allowShared: true,
			same:        true,
			isolate:     false,
			wantAction:  BranchActionStay,
			wantErr:     false,
		},
		{
			name:        "branch-per-feature with same but not allowed",
			mode:        "branch-per-feature",
			allowShared: false,
			same:        true,
			isolate:     false,
			wantAction:  BranchActionNone,
			wantErr:     true,
		},
		{
			name:        "branch-per-feature with isolate",
			mode:        "branch-per-feature",
			allowShared: true,
			same:        false,
			isolate:     true,
			wantAction:  BranchActionCreate,
			wantErr:     false,
		},
		{
			name:        "both flags (error)",
			mode:        "branch-per-feature",
			allowShared: true,
			same:        true,
			isolate:     true,
			wantAction:  BranchActionNone,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAction, err := DetermineBranchAction(tt.mode, tt.allowShared, tt.same, tt.isolate)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetermineBranchAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotAction != tt.wantAction {
				t.Errorf("DetermineBranchAction() = %v, want %v", gotAction, tt.wantAction)
			}
		})
	}
}

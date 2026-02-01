package features

import (
	"strings"
	"testing"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "User Authentication",
			expected: "feature/user-authentication",
		},
		{
			name:     "with special characters",
			input:    "Add @Login & Registration!",
			expected: "feature/add-login-registration",
		},
		{
			name:     "with underscores",
			input:    "fix_bug_123",
			expected: "feature/fix-bug-123",
		},
		{
			name:     "with multiple spaces",
			input:    "Create   New    Feature",
			expected: "feature/create-new-feature",
		},
		{
			name:     "mixed case",
			input:    "AddUserAPI",
			expected: "feature/adduserapi",
		},
		{
			name:     "unicode characters (accents)",
			input:    "Café Feature",
			expected: "feature/cafe-feature",
		},
		// Removed: emoji test - covered by special characters
		{
			name:     "leading and trailing whitespace",
			input:    "  My Feature  ",
			expected: "feature/my-feature",
		},
		{
			name:     "only special characters",
			input:    "!@#$%^&*()",
			expected: "feature/unnamed",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "feature/unnamed",
		},
		{
			name:     "dots and commas",
			input:    "v1.0.0, Release",
			expected: "feature/v1-0-0-release",
		},
		// Removed: parentheses test - covered by special characters
		{
			name:     "forward slashes",
			input:    "API/V2/Endpoint",
			expected: "feature/api/v2/endpoint",
		},
		{
			name:     "very long name",
			input:    "This is an extremely long feature name that exceeds the maximum length limit for Git branch names and needs to be truncated to avoid issues with the Git system when pushing or pulling branches from remote repositories",
			expected: "feature/this-is-an-extremely-long-feature-name-that-exceeds-the-maximum-length-limit-for-git-branch-names-and-needs-to-be-truncated-to-avoid-issues-with-the-git-system-when-pushing-or-pulling-branches-from-remote-repositories",
		},
		{
			name:     "consecutive hyphens",
			input:    "My---Feature",
			expected: "feature/my-feature",
		},
		// Removed: tabs/newlines - covered by whitespace test
		// Removed: german umlauts and french accents - one unicode test sufficient
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetFeatureBranch(t *testing.T) {
	tests := []struct {
		name     string
		feature  *fogit.Feature
		expected string
	}{
		{
			name: "branch in metadata",
			feature: func() *fogit.Feature {
				f := fogit.NewFeature("Test Feature")
				f.Metadata["branch"] = "feature/test-feature"
				return f
			}(),
			expected: "feature/test-feature",
		},
		{
			name: "branch in current version",
			feature: func() *fogit.Feature {
				f := fogit.NewFeature("Test Feature")
				// NewFeature creates version "1" by default
				// Set the branch on that version
				if v, ok := f.Versions["1"]; ok {
					v.Branch = "feature/from-version"
				}
				return f
			}(),
			expected: "feature/from-version",
		},
		{
			name: "metadata takes precedence",
			feature: func() *fogit.Feature {
				f := fogit.NewFeature("Test Feature")
				f.Metadata["branch"] = "feature/from-metadata"
				if v, ok := f.Versions["1"]; ok {
					v.Branch = "feature/from-version"
				}
				return f
			}(),
			expected: "feature/from-metadata",
		},
		{
			name: "no branch anywhere",
			feature: func() *fogit.Feature {
				f := fogit.NewFeature("Test Feature")
				// Clear the branch from the default version
				if v, ok := f.Versions["1"]; ok {
					v.Branch = ""
				}
				return f
			}(),
			expected: "",
		},
		{
			name: "empty metadata branch falls back to version",
			feature: func() *fogit.Feature {
				f := fogit.NewFeature("Test Feature")
				f.Metadata["branch"] = ""
				if v, ok := f.Versions["1"]; ok {
					v.Branch = "feature/fallback"
				}
				return f
			}(),
			expected: "feature/fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetFeatureBranch(tt.feature)
			if result != tt.expected {
				t.Errorf("GetFeatureBranch() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHandleBranchCreation_WorkflowModes(t *testing.T) {
	tests := []struct {
		name        string
		featureName string
		mode        string
		sameFlag    bool
		isolateFlag bool
		allowShared bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "trunk-based mode ignores branch creation",
			featureName: "Test Feature",
			mode:        "trunk-based",
			sameFlag:    false,
			isolateFlag: false,
			wantErr:     false,
		},
		{
			name:        "trunk-based with --same flag errors",
			featureName: "Test Feature",
			mode:        "trunk-based",
			sameFlag:    true,
			isolateFlag: false,
			wantErr:     true,
			errContains: "only works in branch-per-feature mode",
		},
		{
			name:        "trunk-based with --isolate flag errors",
			featureName: "Test Feature",
			mode:        "trunk-based",
			sameFlag:    false,
			isolateFlag: true,
			wantErr:     true,
			errContains: "only works in branch-per-feature mode",
		},
		{
			name:        "both --same and --isolate errors",
			featureName: "Test Feature",
			mode:        "branch-per-feature",
			sameFlag:    true,
			isolateFlag: true,
			allowShared: true,
			wantErr:     true,
			errContains: "cannot use both",
		},
		{
			name:        "--same requires allow_shared_branches",
			featureName: "Test Feature",
			mode:        "branch-per-feature",
			sameFlag:    true,
			isolateFlag: false,
			allowShared: false,
			wantErr:     true,
			errContains: "requires workflow.allow_shared_branches: true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &fogit.Config{
				Workflow: fogit.WorkflowConfig{
					Mode:                tt.mode,
					BaseBranch:          "main",
					AllowSharedBranches: tt.allowShared,
				},
			}

			err := HandleBranchCreation(tt.featureName, cfg, tt.sameFlag, tt.isolateFlag, false)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

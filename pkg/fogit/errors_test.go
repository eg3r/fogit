package fogit

import (
	"errors"
	"fmt"
	"testing"
)

func TestNotFoundError(t *testing.T) {
	tests := []struct {
		name      string
		err       *NotFoundError
		wantMsg   string
		wantIs    []error
		wantNotIs []error
	}{
		{
			name:    "with resource and identifier",
			err:     NewNotFoundError("feature", "abc123"),
			wantMsg: "feature not found: abc123",
			wantIs:  []error{ErrNotFound, ErrFeatureNotFound, ErrRelationshipNotFound},
		},
		{
			name:    "with resource only",
			err:     &NotFoundError{Resource: "feature"},
			wantMsg: "feature not found",
			wantIs:  []error{ErrNotFound},
		},
		{
			name:    "with custom message",
			err:     &NotFoundError{Resource: "feature", Identifier: "abc", Message: "custom error message"},
			wantMsg: "custom error message",
			wantIs:  []error{ErrNotFound},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}

			for _, target := range tt.wantIs {
				if !errors.Is(tt.err, target) {
					t.Errorf("errors.Is(%v, %v) = false, want true", tt.err, target)
				}
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name    string
		err     *ValidationError
		wantMsg string
	}{
		{
			name:    "with field and value",
			err:     NewValidationError("priority", "super-high", "must be one of low, medium, high, critical"),
			wantMsg: `invalid priority "super-high": must be one of low, medium, high, critical`,
		},
		{
			name:    "with field only",
			err:     &ValidationError{Field: "name", Message: "cannot be empty"},
			wantMsg: "invalid name: cannot be empty",
		},
		{
			name:    "message only",
			err:     &ValidationError{Message: "validation failed"},
			wantMsg: "validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}

			// ValidationError should implement error interface
			// Just verify it works with errors.Is for common sentinel errors
			_ = errors.Is(tt.err, ErrInvalidState)
			_ = errors.Is(tt.err, ErrInvalidPriority)
		})
	}
}

func TestDuplicateError(t *testing.T) {
	tests := []struct {
		name    string
		err     *DuplicateError
		wantMsg string
	}{
		{
			name:    "with resource and identifier",
			err:     NewDuplicateError("feature", "login-system"),
			wantMsg: "feature already exists: login-system",
		},
		{
			name:    "with resource only",
			err:     &DuplicateError{Resource: "relationship"},
			wantMsg: "relationship already exists",
		},
		{
			name:    "with custom message",
			err:     &DuplicateError{Message: "duplicate entry detected"},
			wantMsg: "duplicate entry detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}

			if !errors.Is(tt.err, ErrFeatureAlreadyExists) {
				t.Error("DuplicateError should match ErrFeatureAlreadyExists")
			}
			if !errors.Is(tt.err, ErrDuplicateRelationship) {
				t.Error("DuplicateError should match ErrDuplicateRelationship")
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"ErrNotFound", ErrNotFound, true},
		{"ErrFeatureNotFound", ErrFeatureNotFound, true},
		{"ErrRelationshipNotFound", ErrRelationshipNotFound, true},
		{"NotFoundError struct", NewNotFoundError("feature", "abc"), true},
		{"wrapped ErrNotFound", errors.Join(errors.New("context"), ErrNotFound), true},
		{"unrelated error", errors.New("some other error"), false},
		{"ErrInvalidState", ErrInvalidState, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"ErrInvalidState", ErrInvalidState, true},
		{"ErrInvalidPriority", ErrInvalidPriority, true},
		{"ErrEmptyName", ErrEmptyName, true},
		{"ValidationError struct", NewValidationError("field", "value", "msg"), true},
		{"unrelated error", errors.New("some other error"), false},
		{"ErrNotFound", ErrNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidationError(tt.err); got != tt.want {
				t.Errorf("IsValidationError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsDuplicateError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"ErrFeatureAlreadyExists", ErrFeatureAlreadyExists, true},
		{"ErrDuplicateRelationship", ErrDuplicateRelationship, true},
		{"DuplicateError struct", NewDuplicateError("feature", "abc"), true},
		{"unrelated error", errors.New("some other error"), false},
		{"ErrNotFound", ErrNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDuplicateError(tt.err); got != tt.want {
				t.Errorf("IsDuplicateError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestErrorsIsCompatibility(t *testing.T) {
	// Test that new error types work with errors.Is for backward compatibility
	notFoundErr := NewNotFoundError("feature", "test-id")

	// Should work with errors.Is
	if !errors.Is(notFoundErr, ErrNotFound) {
		t.Error("NotFoundError should match ErrNotFound via errors.Is")
	}

	// Should work in switch statements comparing to sentinel
	var err error = notFoundErr
	switch {
	case errors.Is(err, ErrNotFound):
		// Expected path
	default:
		t.Error("errors.Is should match in switch statement")
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"nil error", nil, ExitSuccess},
		{"general error", errors.New("something went wrong"), ExitGeneralError},
		{"ErrNotFound", ErrNotFound, ExitNotFound},
		{"ErrFeatureNotFound", ErrFeatureNotFound, ExitNotFound},
		{"ErrRelationshipNotFound", ErrRelationshipNotFound, ExitNotFound},
		{"NotFoundError struct", NewNotFoundError("feature", "abc"), ExitNotFound},
		{"ErrInvalidState", ErrInvalidState, ExitValidationError},
		{"ErrInvalidPriority", ErrInvalidPriority, ExitValidationError},
		{"ErrEmptyName", ErrEmptyName, ExitValidationError},
		{"ValidationError struct", NewValidationError("field", "val", "msg"), ExitValidationError},
		{"ErrFeatureAlreadyExists", ErrFeatureAlreadyExists, ExitConflict},
		{"ErrDuplicateRelationship", ErrDuplicateRelationship, ExitConflict},
		{"DuplicateError struct", NewDuplicateError("feature", "abc"), ExitConflict},
		{"ErrRepositoryNotInitialized", ErrRepositoryNotInitialized, ExitConfigError},
		{"explicit ExitCodeError", NewExitCodeError(errors.New("git failed"), ExitGitError), ExitGitError},
		{"wrapped not found", fmt.Errorf("failed: %w", ErrNotFound), ExitNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetExitCode(tt.err)
			if got != tt.wantCode {
				t.Errorf("GetExitCode(%v) = %d, want %d", tt.err, got, tt.wantCode)
			}
		})
	}
}

func TestExitCodeError(t *testing.T) {
	innerErr := errors.New("git command failed")
	exitErr := NewExitCodeError(innerErr, ExitGitError)

	// Test Error() returns inner error message
	if exitErr.Error() != "git command failed" {
		t.Errorf("Error() = %q, want %q", exitErr.Error(), "git command failed")
	}

	// Test Unwrap() returns inner error
	if exitErr.Unwrap() != innerErr {
		t.Error("Unwrap() should return inner error")
	}

	// Test errors.Is works through wrapper
	if !errors.Is(exitErr, innerErr) {
		t.Error("errors.Is should find inner error")
	}

	// Test GetExitCode extracts the code
	if GetExitCode(exitErr) != ExitGitError {
		t.Errorf("GetExitCode() = %d, want %d", GetExitCode(exitErr), ExitGitError)
	}
}

func TestExitCodeConstants(t *testing.T) {
	// Verify exit codes match spec (08-interface.md)
	expected := map[string]int{
		"ExitSuccess":         0,
		"ExitGeneralError":    1,
		"ExitInvalidArgs":     2,
		"ExitNotFound":        3,
		"ExitValidationError": 4,
		"ExitConfigError":     5,
		"ExitGitError":        6,
		"ExitConflict":        7,
		"ExitPermissionError": 8,
	}

	actual := map[string]int{
		"ExitSuccess":         ExitSuccess,
		"ExitGeneralError":    ExitGeneralError,
		"ExitInvalidArgs":     ExitInvalidArgs,
		"ExitNotFound":        ExitNotFound,
		"ExitValidationError": ExitValidationError,
		"ExitConfigError":     ExitConfigError,
		"ExitGitError":        ExitGitError,
		"ExitConflict":        ExitConflict,
		"ExitPermissionError": ExitPermissionError,
	}

	for name, want := range expected {
		got := actual[name]
		if got != want {
			t.Errorf("%s = %d, want %d (per spec)", name, got, want)
		}
	}
}

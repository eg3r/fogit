package fogit

import (
	"errors"
	"fmt"
)

// Domain errors - sentinel errors for backward compatibility
var (
	// Feature errors
	ErrEmptyName       = errors.New("name cannot be empty")
	ErrInvalidState    = errors.New("invalid state: must be one of open, in-progress, closed")
	ErrInvalidPriority = errors.New("invalid priority: must be one of low, medium, high, critical")
	ErrFeatureNotFound = errors.New("feature not found")

	// Relationship errors
	ErrInvalidRelationshipType = errors.New("invalid relationship type")
	ErrEmptyTargetID           = errors.New("target ID cannot be empty")
	ErrDuplicateRelationship   = errors.New("relationship already exists")
	ErrRelationshipNotFound    = errors.New("relationship not found")

	// Repository errors
	ErrRepositoryNotInitialized = errors.New("fogit repository not initialized")
	ErrFeatureAlreadyExists     = errors.New("feature already exists")
	ErrNotFound                 = errors.New("not found")
)

// NotFoundError provides detailed context for not found errors
type NotFoundError struct {
	Resource   string // "feature", "relationship", "version", etc.
	Identifier string // ID, name, or other identifier used in lookup
	Message    string // Optional additional context
}

func (e *NotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Identifier != "" {
		return fmt.Sprintf("%s not found: %s", e.Resource, e.Identifier)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// Is implements errors.Is for compatibility with sentinel errors
func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound || target == ErrFeatureNotFound || target == ErrRelationshipNotFound
}

// NewNotFoundError creates a new NotFoundError
func NewNotFoundError(resource, identifier string) *NotFoundError {
	return &NotFoundError{Resource: resource, Identifier: identifier}
}

// ValidationError provides context for validation failures
type ValidationError struct {
	Field   string // Field that failed validation
	Value   string // The invalid value (if safe to include)
	Message string // Description of the validation failure
}

func (e *ValidationError) Error() string {
	if e.Field != "" && e.Value != "" {
		return fmt.Sprintf("invalid %s %q: %s", e.Field, e.Value, e.Message)
	}
	if e.Field != "" {
		return fmt.Sprintf("invalid %s: %s", e.Field, e.Message)
	}
	return e.Message
}

// Is implements errors.Is for compatibility with sentinel errors
func (e *ValidationError) Is(target error) bool {
	return target == ErrInvalidState || target == ErrInvalidPriority || target == ErrEmptyName
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{Field: field, Value: value, Message: message}
}

// DuplicateError indicates an attempt to create something that already exists
type DuplicateError struct {
	Resource   string // "feature", "relationship", etc.
	Identifier string // ID or name of the duplicate
	Message    string // Optional additional context
}

func (e *DuplicateError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Identifier != "" {
		return fmt.Sprintf("%s already exists: %s", e.Resource, e.Identifier)
	}
	return fmt.Sprintf("%s already exists", e.Resource)
}

// Is implements errors.Is for compatibility with sentinel errors
func (e *DuplicateError) Is(target error) bool {
	return target == ErrFeatureAlreadyExists || target == ErrDuplicateRelationship
}

// NewDuplicateError creates a new DuplicateError
func NewDuplicateError(resource, identifier string) *DuplicateError {
	return &DuplicateError{Resource: resource, Identifier: identifier}
}

// Helper functions to check error types

// IsNotFound checks if an error is a not found error (either sentinel or structured)
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFoundErr *NotFoundError
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrFeatureNotFound) ||
		errors.Is(err, ErrRelationshipNotFound) || errors.As(err, &notFoundErr)
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	var validationErr *ValidationError
	return errors.Is(err, ErrInvalidState) || errors.Is(err, ErrInvalidPriority) ||
		errors.Is(err, ErrEmptyName) || errors.As(err, &validationErr)
}

// IsDuplicateError checks if an error is a duplicate error
func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	var dupErr *DuplicateError
	return errors.Is(err, ErrFeatureAlreadyExists) || errors.Is(err, ErrDuplicateRelationship) ||
		errors.As(err, &dupErr)
}

// Exit codes per spec (08-interface.md)
const (
	ExitSuccess           = 0  // Command completed successfully
	ExitGeneralError      = 1  // Unexpected errors, internal failures
	ExitInvalidArgs       = 2  // Unknown option, invalid syntax, missing required argument
	ExitNotFound          = 3  // Feature ID/name doesn't exist, relationship target not found
	ExitValidationError   = 4  // Invalid YAML, schema violation, broken relationship reference
	ExitConfigError       = 5  // Invalid config.yml, missing required config
	ExitGitError          = 6  // Not a Git repository, Git command failed, hook error
	ExitConflict          = 7  // Relationship would create cycle, duplicate feature name
	ExitPermissionError   = 8  // Read-only filesystem, file access denied
	ExitMergeConflict     = 9  // Git merge conflict detected, requires resolution
	ExitMergeInProgress   = 10 // A merge is already in progress (use --continue or --abort)
	ExitNoMergeInProgress = 11 // --continue or --abort used but no merge is active
)

// ExitCodeError wraps an error with a specific exit code
type ExitCodeError struct {
	Err      error
	ExitCode int
}

func (e *ExitCodeError) Error() string {
	return e.Err.Error()
}

func (e *ExitCodeError) Unwrap() error {
	return e.Err
}

// NewExitCodeError wraps an error with a specific exit code
func NewExitCodeError(err error, code int) *ExitCodeError {
	return &ExitCodeError{Err: err, ExitCode: code}
}

// MergeConflictError indicates a git merge conflict that requires manual resolution
type MergeConflictError struct {
	ConflictFiles []string // Files with conflicts
	Message       string   // Additional context
}

func (e *MergeConflictError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "merge conflict detected: resolve conflicts and run 'fogit merge --continue'"
}

// NewMergeConflictError creates a new MergeConflictError
func NewMergeConflictError(conflictFiles []string) *MergeConflictError {
	return &MergeConflictError{ConflictFiles: conflictFiles}
}

// IsMergeConflictError checks if an error is a merge conflict error
func IsMergeConflictError(err error) bool {
	if err == nil {
		return false
	}
	var mergeErr *MergeConflictError
	return errors.As(err, &mergeErr)
}

// MergeInProgressError indicates a merge is already in progress
type MergeInProgressError struct {
	Message string
}

func (e *MergeInProgressError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "merge already in progress: use --continue or --abort"
}

// NewMergeInProgressError creates a new MergeInProgressError
func NewMergeInProgressError() *MergeInProgressError {
	return &MergeInProgressError{}
}

// IsMergeInProgressError checks if an error is a merge in progress error
func IsMergeInProgressError(err error) bool {
	if err == nil {
		return false
	}
	var mergeErr *MergeInProgressError
	return errors.As(err, &mergeErr)
}

// NoMergeInProgressError indicates --continue or --abort used but no merge is active
type NoMergeInProgressError struct {
	Message string
}

func (e *NoMergeInProgressError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "no merge in progress"
}

// NewNoMergeInProgressError creates a new NoMergeInProgressError
func NewNoMergeInProgressError() *NoMergeInProgressError {
	return &NoMergeInProgressError{}
}

// IsNoMergeInProgressError checks if an error is a no merge in progress error
func IsNoMergeInProgressError(err error) bool {
	if err == nil {
		return false
	}
	var mergeErr *NoMergeInProgressError
	return errors.As(err, &mergeErr)
}

// GetExitCode determines the appropriate exit code for an error
// based on the error type. Returns ExitSuccess (0) for nil errors.
func GetExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	// Check for explicit exit code wrapper
	var exitErr *ExitCodeError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode
	}

	// Check for merge-related errors (must be before general error checks)
	if IsMergeConflictError(err) {
		return ExitMergeConflict
	}
	if IsMergeInProgressError(err) {
		return ExitMergeInProgress
	}
	if IsNoMergeInProgressError(err) {
		return ExitNoMergeInProgress
	}

	// Check for not found errors
	if IsNotFound(err) {
		return ExitNotFound
	}

	// Check for validation errors
	if IsValidationError(err) {
		return ExitValidationError
	}

	// Check for duplicate/conflict errors
	if IsDuplicateError(err) {
		return ExitConflict
	}

	// Check for repository not initialized (config error)
	if errors.Is(err, ErrRepositoryNotInitialized) {
		return ExitConfigError
	}

	// Default to general error
	return ExitGeneralError
}

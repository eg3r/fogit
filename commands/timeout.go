package commands

import (
	"context"
	"time"
)

// Default operation timeouts.
// These can be overridden via environment variables or config in the future.
const (
	// DefaultListTimeout is the timeout for listing features.
	// This operation reads all feature files and could be slow with many features.
	DefaultListTimeout = 30 * time.Second

	// DefaultSearchTimeout is the timeout for search operations.
	// This includes fuzzy matching across all features.
	DefaultSearchTimeout = 30 * time.Second

	// DefaultExportTimeout is the timeout for export operations.
	// This reads all features and converts them to the export format.
	DefaultExportTimeout = 60 * time.Second

	// DefaultImportTimeout is the timeout for import operations.
	// This reads external files and creates features.
	DefaultImportTimeout = 60 * time.Second

	// DefaultValidateTimeout is the timeout for validation operations.
	// This validates all features and their relationships.
	DefaultValidateTimeout = 60 * time.Second
)

// WithTimeout creates a child context with the specified timeout.
// It returns the context, a cancel function that should be deferred,
// and handles the case where the parent context is nil.
//
// Usage:
//
//	ctx, cancel := WithTimeout(cmd.Context(), DefaultListTimeout)
//	defer cancel()
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, timeout)
}

// WithListTimeout creates a context with the default list operation timeout.
func WithListTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, DefaultListTimeout)
}

// WithSearchTimeout creates a context with the default search operation timeout.
func WithSearchTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, DefaultSearchTimeout)
}

// WithExportTimeout creates a context with the default export operation timeout.
func WithExportTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, DefaultExportTimeout)
}

// WithImportTimeout creates a context with the default import operation timeout.
func WithImportTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, DefaultImportTimeout)
}

// WithValidateTimeout creates a context with the default validation operation timeout.
func WithValidateTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, DefaultValidateTimeout)
}

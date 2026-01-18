// Package testutil provides common test utilities and helpers for FoGit tests.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// TempDir creates a temporary directory for testing and returns a cleanup function.
// The cleanup function removes the directory and all its contents.
//
// Usage:
//
//	tempDir, cleanup := testutil.TempDir(t)
//	defer cleanup()
func TempDir(t *testing.T) (string, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "fogit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// TempDirWithFogit creates a temporary directory with a .fogit structure for testing.
// Returns the root directory path, the .fogit directory path, and a cleanup function.
//
// Usage:
//
//	rootDir, fogitDir, cleanup := testutil.TempDirWithFogit(t)
//	defer cleanup()
func TempDirWithFogit(t *testing.T) (rootDir string, fogitDir string, cleanup func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "fogit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	fogitDir = filepath.Join(tempDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")

	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create .fogit/features dir: %v", err)
	}

	cleanup = func() {
		os.RemoveAll(tempDir)
	}

	return tempDir, fogitDir, cleanup
}

// NewTestFeature creates a new feature with the given name for testing.
// It sets common defaults that are useful for tests.
func NewTestFeature(name string) *fogit.Feature {
	f := fogit.NewFeature(name)
	return f
}

// NewTestFeatureWithType creates a new feature with the given name and type for testing.
func NewTestFeatureWithType(name, featureType string) *fogit.Feature {
	f := fogit.NewFeature(name)
	f.SetType(featureType)
	return f
}

// NewTestFeatureWithPriority creates a new feature with the given name and priority for testing.
func NewTestFeatureWithPriority(name string, priority fogit.Priority) *fogit.Feature {
	f := fogit.NewFeature(name)
	f.SetPriority(priority)
	return f
}

// RequireNoError fails the test immediately if err is not nil.
// Use this for setup steps where failure means the test cannot continue.
func RequireNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError checks that an error occurred and optionally matches the expected error.
func AssertError(t *testing.T, err error, expected error) {
	t.Helper()
	if err == nil {
		t.Error("expected an error but got nil")
		return
	}
	if expected != nil && err != expected {
		t.Errorf("expected error %v, got %v", expected, err)
	}
}

// AssertNoError checks that no error occurred.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// AssertEqual checks that two values are equal.
func AssertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

// AssertContains checks that a string contains a substring.
func AssertContains(t *testing.T, got, substr string, msg string) {
	t.Helper()
	if !containsString(got, substr) {
		t.Errorf("%s: %q does not contain %q", msg, got, substr)
	}
}

func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// AssertNotContains checks that a string does not contain a substring.
func AssertNotContains(t *testing.T, got, substr string, msg string) {
	t.Helper()
	if containsString(got, substr) {
		t.Errorf("%s: %q should not contain %q", msg, got, substr)
	}
}

// AssertTrue checks that a condition is true.
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Error(msg)
	}
}

// AssertFalse checks that a condition is false.
func AssertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Error(msg)
	}
}

// AssertNil checks that a value is nil.
func AssertNil(t *testing.T, val interface{}, msg string) {
	t.Helper()
	if val != nil {
		t.Errorf("%s: expected nil, got %v", msg, val)
	}
}

// AssertNotNil checks that a value is not nil.
func AssertNotNil(t *testing.T, val interface{}, msg string) {
	t.Helper()
	if val == nil {
		t.Errorf("%s: expected non-nil value", msg)
	}
}

// GitTestEnv represents a test environment with a Git repository and FoGit setup.
type GitTestEnv struct {
	RootDir    string                  // Root directory of the test environment
	FogitDir   string                  // Path to .fogit directory
	GitRepo    *gogit.Repository       // go-git repository
	Repository *storage.FileRepository // FoGit storage repository
	cleanup    func()
}

// TempDirWithGit creates a temporary directory with an initialized Git repository.
// Returns a GitTestEnv with the repository configured for testing.
//
// Usage:
//
//	env := testutil.TempDirWithGit(t)
//	defer env.Cleanup()
func TempDirWithGit(t *testing.T) *GitTestEnv {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "fogit-test-git-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Initialize Git repository
	gitRepo, err := gogit.PlainInit(tempDir, false)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to init git repository: %v", err)
	}

	// Configure Git user
	cfg, err := gitRepo.Config()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to get git config: %v", err)
	}
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	if err := gitRepo.SetConfig(cfg); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to set git config: %v", err)
	}

	// Create .fogit directory structure
	fogitDir := filepath.Join(tempDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create .fogit/features dir: %v", err)
	}

	// Create storage repository
	repo := storage.NewFileRepository(fogitDir)

	return &GitTestEnv{
		RootDir:    tempDir,
		FogitDir:   fogitDir,
		GitRepo:    gitRepo,
		Repository: repo,
		cleanup: func() {
			os.RemoveAll(tempDir)
		},
	}
}

// Cleanup removes the temporary test environment.
func (e *GitTestEnv) Cleanup() {
	if e.cleanup != nil {
		e.cleanup()
	}
}

// CreateInitialCommit creates an initial commit in the test Git repository.
func (e *GitTestEnv) CreateInitialCommit(t *testing.T) {
	t.Helper()

	w, err := e.GitRepo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(e.RootDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test Repository\n"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if _, err := w.Add("."); err != nil {
		t.Fatalf("failed to add files: %v", err)
	}

	_, err = w.Commit("Initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}
}

// CreateFeature creates a feature in the test repository and returns it.
func (e *GitTestEnv) CreateFeature(t *testing.T, name string) *fogit.Feature {
	t.Helper()

	feature := fogit.NewFeature(name)
	ctx := context.Background()

	if err := e.Repository.Create(ctx, feature); err != nil {
		t.Fatalf("failed to create feature %q: %v", name, err)
	}

	return feature
}

// CreateFeatureWithOptions creates a feature with custom settings.
func (e *GitTestEnv) CreateFeatureWithOptions(t *testing.T, name string, opts ...func(*fogit.Feature)) *fogit.Feature {
	t.Helper()

	feature := fogit.NewFeature(name)
	for _, opt := range opts {
		opt(feature)
	}

	ctx := context.Background()
	if err := e.Repository.Create(ctx, feature); err != nil {
		t.Fatalf("failed to create feature %q: %v", name, err)
	}

	return feature
}

// WithPriority returns an option function to set feature priority.
func WithPriority(p fogit.Priority) func(*fogit.Feature) {
	return func(f *fogit.Feature) {
		f.SetPriority(p)
	}
}

// WithType returns an option function to set feature type.
func WithType(featureType string) func(*fogit.Feature) {
	return func(f *fogit.Feature) {
		f.SetType(featureType)
	}
}

// WithDescription returns an option function to set feature description.
func WithDescription(desc string) func(*fogit.Feature) {
	return func(f *fogit.Feature) {
		f.Description = desc
	}
}

// SetupTestRepository creates a temporary FoGit repository for testing.
// This is a simpler version without Git, useful for storage-only tests.
//
// Usage:
//
//	repo, cleanup := testutil.SetupTestRepository(t)
//	defer cleanup()
func SetupTestRepository(t *testing.T) (*storage.FileRepository, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "fogit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repo := storage.NewFileRepository(tempDir)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return repo, cleanup
}

// InDir runs a function in the specified directory and restores the original
// working directory after completion. This is safe to use in tests that need
// to change the working directory temporarily.
//
// Usage:
//
//	tmpDir := t.TempDir()
//	testutil.InDir(t, tmpDir, func() {
//	    // Code runs with tmpDir as working directory
//	})
//	// Original directory is restored here
func InDir(t *testing.T, dir string, fn func()) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to directory %s: %v", dir, err)
	}

	defer func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("failed to restore directory to %s: %v", oldDir, err)
		}
	}()

	fn()
}

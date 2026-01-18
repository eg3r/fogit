package testutil

import (
	"context"
	"testing"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestTempDir(t *testing.T) {
	tempDir, cleanup := TempDir(t)
	defer cleanup()

	if tempDir == "" {
		t.Error("TempDir returned empty string")
	}
}

func TestTempDirWithFogit(t *testing.T) {
	rootDir, fogitDir, cleanup := TempDirWithFogit(t)
	defer cleanup()

	if rootDir == "" {
		t.Error("TempDirWithFogit returned empty rootDir")
	}
	if fogitDir == "" {
		t.Error("TempDirWithFogit returned empty fogitDir")
	}
}

func TestTempDirWithGit(t *testing.T) {
	env := TempDirWithGit(t)
	defer env.Cleanup()

	if env.RootDir == "" {
		t.Error("GitTestEnv.RootDir is empty")
	}
	if env.FogitDir == "" {
		t.Error("GitTestEnv.FogitDir is empty")
	}
	if env.GitRepo == nil {
		t.Error("GitTestEnv.GitRepo is nil")
	}
	if env.Repository == nil {
		t.Error("GitTestEnv.Repository is nil")
	}
}

func TestGitTestEnv_CreateInitialCommit(t *testing.T) {
	env := TempDirWithGit(t)
	defer env.Cleanup()

	// Should not panic
	env.CreateInitialCommit(t)

	// Verify commit exists
	head, err := env.GitRepo.Head()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}
	if head == nil {
		t.Error("HEAD is nil after commit")
	}
}

func TestGitTestEnv_CreateFeature(t *testing.T) {
	env := TempDirWithGit(t)
	defer env.Cleanup()

	feature := env.CreateFeature(t, "Test Feature")

	if feature == nil {
		t.Fatal("CreateFeature returned nil")
	}
	if feature.Name != "Test Feature" {
		t.Errorf("feature name = %q, want %q", feature.Name, "Test Feature")
	}

	// Verify feature was stored
	ctx := context.Background()
	retrieved, err := env.Repository.Get(ctx, feature.ID)
	if err != nil {
		t.Fatalf("failed to retrieve feature: %v", err)
	}
	if retrieved.Name != "Test Feature" {
		t.Errorf("retrieved feature name = %q, want %q", retrieved.Name, "Test Feature")
	}
}

func TestGitTestEnv_CreateFeatureWithOptions(t *testing.T) {
	env := TempDirWithGit(t)
	defer env.Cleanup()

	feature := env.CreateFeatureWithOptions(t, "Test Feature",
		WithPriority(fogit.PriorityHigh),
		WithType("enhancement"),
		WithDescription("A test feature"),
	)

	if feature == nil {
		t.Fatal("CreateFeatureWithOptions returned nil")
	}
	if feature.GetPriority() != fogit.PriorityHigh {
		t.Errorf("priority = %v, want %v", feature.GetPriority(), fogit.PriorityHigh)
	}
	if feature.GetType() != "enhancement" {
		t.Errorf("type = %q, want %q", feature.GetType(), "enhancement")
	}
	if feature.Description != "A test feature" {
		t.Errorf("description = %q, want %q", feature.Description, "A test feature")
	}
}

func TestSetupTestRepository(t *testing.T) {
	repo, cleanup := SetupTestRepository(t)
	defer cleanup()

	if repo == nil {
		t.Fatal("SetupTestRepository returned nil")
	}

	// Verify we can create a feature
	ctx := context.Background()
	feature := fogit.NewFeature("Test")
	err := repo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("failed to create feature: %v", err)
	}
}

func TestAssertEqual(t *testing.T) {
	// This should not fail
	mockT := &testing.T{}
	AssertEqual(mockT, "hello", "hello", "strings should match")

	// We can't easily test that it fails correctly without a mock,
	// but we can at least verify it doesn't panic
}

func TestAssertContains(t *testing.T) {
	mockT := &testing.T{}
	AssertContains(mockT, "hello world", "world", "should contain substring")
}

func TestAssertNotContains(t *testing.T) {
	mockT := &testing.T{}
	AssertNotContains(mockT, "hello world", "xyz", "should not contain substring")
}

func TestAssertTrue(t *testing.T) {
	mockT := &testing.T{}
	AssertTrue(mockT, true, "should be true")
}

func TestAssertFalse(t *testing.T) {
	mockT := &testing.T{}
	AssertFalse(mockT, false, "should be false")
}

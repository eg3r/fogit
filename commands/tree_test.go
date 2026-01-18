package commands

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestTreeCommand_SingleType(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Defaults.TreeType = "contains"
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	ctx := context.Background()

	// Create hierarchy: Parent -> Child1, Child2
	parent := fogit.NewFeature("Parent")
	child1 := fogit.NewFeature("Child 1")
	child2 := fogit.NewFeature("Child 2")

	if err := repo.Create(ctx, parent); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, child1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, child2); err != nil {
		t.Fatal(err)
	}

	// Add contains relationships
	parent.AddRelationship(fogit.NewRelationship("contains", child1.ID, child1.Name))
	parent.AddRelationship(fogit.NewRelationship("contains", child2.ID, child2.Name))
	if err := repo.Update(ctx, parent); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("tree command error = %v", err)
	}
}

func TestTreeCommand_MultipleTypes(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	ctx := context.Background()

	// Create features with different relationship types
	database := fogit.NewFeature("Database Layer")
	auth := fogit.NewFeature("Authentication Service")
	backend := fogit.NewFeature("Backend API")

	if err := repo.Create(ctx, database); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, auth); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, backend); err != nil {
		t.Fatal(err)
	}

	// Auth depends-on Database
	auth.AddRelationship(fogit.NewRelationship("depends-on", database.ID, database.Name))
	if err := repo.Update(ctx, auth); err != nil {
		t.Fatal(err)
	}

	// Backend contains Auth
	backend.AddRelationship(fogit.NewRelationship("contains", auth.ID, auth.Name))
	if err := repo.Update(ctx, backend); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree", "--type", "depends-on", "--type", "contains"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("tree command with multiple types error = %v", err)
	}
}

func TestTreeCommand_InvalidType(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree", "--type", "nonexistent-type"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Fatal("Expected error with invalid relationship type")
	}

	if !strings.Contains(err.Error(), "not defined in config") {
		t.Errorf("Expected 'not defined in config' error, got: %v", err)
	}
}

func TestTreeCommand_NoDefaultType(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with no default tree type and no types at all
	cfg := &fogit.Config{
		Relationships: fogit.RelationshipsConfig{
			Types:      make(map[string]fogit.RelationshipTypeConfig),
			Categories: make(map[string]fogit.RelationshipCategory),
			Defaults: fogit.RelationshipDefaults{
				TreeType: "", // No default
			},
		},
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Fatal("Expected error when no default type and no types defined")
	}

	if !strings.Contains(err.Error(), "no tree relationship type configured") {
		t.Errorf("Expected 'no tree relationship type configured' error, got: %v", err)
	}
}

func TestTreeCommand_DefaultToNonCyclicType(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with no default but with types
	cfg := fogit.DefaultConfig()
	cfg.Relationships.Defaults.TreeType = "" // Remove default
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree"})

	// Should not error - should fall back to first non-cyclic type
	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("tree command should not error with auto-fallback, got: %v", err)
	}
}

func TestTreeCommand_DepthLimit(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	ctx := context.Background()

	// Create deep hierarchy: L0 -> L1 -> L2 -> L3
	l0 := fogit.NewFeature("Level 0")
	l1 := fogit.NewFeature("Level 1")
	l2 := fogit.NewFeature("Level 2")
	l3 := fogit.NewFeature("Level 3")

	for _, f := range []*fogit.Feature{l0, l1, l2, l3} {
		if err := repo.Create(ctx, f); err != nil {
			t.Fatal(err)
		}
	}

	l0.AddRelationship(fogit.NewRelationship("contains", l1.ID, l1.Name))
	l1.AddRelationship(fogit.NewRelationship("contains", l2.ID, l2.Name))
	l2.AddRelationship(fogit.NewRelationship("contains", l3.ID, l3.Name))

	if err := repo.Update(ctx, l0); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, l1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, l2); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree", "--type", "contains", "--depth", "2"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("tree command with depth limit error = %v", err)
	}
}

func TestTreeCommand_CategoryFilter(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	ctx := context.Background()

	// Create features with different categories
	backend := fogit.NewFeature("Backend")
	backend.SetCategory("backend")
	frontend := fogit.NewFeature("Frontend")
	frontend.SetCategory("frontend")

	if err := repo.Create(ctx, backend); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, frontend); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree", "--type", "contains", "--category", "backend"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("tree command with category filter error = %v", err)
	}
}

func TestTreeCommand_StateFilter(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	ctx := context.Background()

	// Create features with different states
	open := fogit.NewFeature("Open Feature")
	// open is already in StateOpen (default from NewFeature)
	inProgress := fogit.NewFeature("In Progress Feature")
	inProgress.UpdateState(fogit.StateInProgress)

	if err := repo.Create(ctx, open); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, inProgress); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree", "--type", "contains", "--state", "open"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("tree command with state filter error = %v", err)
	}
}

func TestTreeCommand_SpecificFeature(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	ctx := context.Background()

	// Create hierarchy
	parent := fogit.NewFeature("Parent Feature")
	child := fogit.NewFeature("Child Feature")

	if err := repo.Create(ctx, parent); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, child); err != nil {
		t.Fatal(err)
	}

	parent.AddRelationship(fogit.NewRelationship("contains", child.ID, child.Name))
	if err := repo.Update(ctx, parent); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree", "--type", "contains", "Parent Feature"})

	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("tree command with specific feature error = %v", err)
	}
}

func TestTreeCommand_NonExistentFeature(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run via rootCmd with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "tree", "--type", "contains", "NonExistent Feature"})

	// Tree command with non-existent feature shows "No features found", not an error
	// This is expected behavior - just no features match the criteria
	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("tree command should not error with non-existent feature, got: %v", err)
	}
}

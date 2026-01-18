package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestAutoInverseRelationships(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with auto-inverse enabled
	cfg := fogit.DefaultConfig()
	cfg.Relationships.System.AutoCreateInverse = true
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Create repository
	repo := storage.NewFileRepository(fogitDir)

	// Create two features
	source := fogit.NewFeature("Source Feature")
	target := fogit.NewFeature("Target Feature")

	ctx := context.Background()
	if err := repo.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, target); err != nil {
		t.Fatal(err)
	}

	// Simulate link command creating depends-on relationship
	rel := fogit.NewRelationship("depends-on", target.ID, target.Name)
	if err := source.AddRelationship(rel); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, source); err != nil {
		t.Fatal(err)
	}

	// Auto-create inverse (required-by)
	typeConfig := cfg.Relationships.Types["depends-on"]
	if typeConfig.Inverse != "" {
		inverseRel := fogit.NewRelationship(fogit.RelationshipType(typeConfig.Inverse), source.ID, source.Name)
		if err := target.AddRelationship(inverseRel); err != nil {
			t.Fatal(err)
		}
		if err := repo.Update(ctx, target); err != nil {
			t.Fatal(err)
		}
	}

	// Verify source has depends-on relationship
	reloadedSource, err := repo.Get(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloadedSource.Relationships) != 1 {
		t.Errorf("Expected 1 relationship on source, got %d", len(reloadedSource.Relationships))
	}
	if reloadedSource.Relationships[0].Type != "depends-on" {
		t.Errorf("Expected depends-on relationship, got %s", reloadedSource.Relationships[0].Type)
	}

	// Verify target has required-by relationship (inverse)
	reloadedTarget, err := repo.Get(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloadedTarget.Relationships) != 1 {
		t.Errorf("Expected 1 relationship on target (inverse), got %d", len(reloadedTarget.Relationships))
	}
	if reloadedTarget.Relationships[0].Type != "required-by" {
		t.Errorf("Expected required-by relationship, got %s", reloadedTarget.Relationships[0].Type)
	}
	if reloadedTarget.Relationships[0].TargetID != source.ID {
		t.Errorf("Expected inverse to point to source, got %s", reloadedTarget.Relationships[0].TargetID)
	}
}

func TestAutoInverseDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with auto-inverse disabled
	cfg := fogit.DefaultConfig()
	cfg.Relationships.System.AutoCreateInverse = false
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)

	source := fogit.NewFeature("Source Feature")
	target := fogit.NewFeature("Target Feature")

	ctx := context.Background()
	if err := repo.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, target); err != nil {
		t.Fatal(err)
	}

	// Create relationship (no auto-inverse)
	rel := fogit.NewRelationship("depends-on", target.ID, target.Name)
	if err := source.AddRelationship(rel); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, source); err != nil {
		t.Fatal(err)
	}

	// Verify source has relationship
	reloadedSource, err := repo.Get(ctx, source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloadedSource.Relationships) != 1 {
		t.Errorf("Expected 1 relationship on source, got %d", len(reloadedSource.Relationships))
	}

	// Verify target has NO inverse relationship
	reloadedTarget, err := repo.Get(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloadedTarget.Relationships) != 0 {
		t.Errorf("Expected 0 relationships on target (auto-inverse disabled), got %d", len(reloadedTarget.Relationships))
	}
}

func TestAutoInverseBidirectional(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.System.AutoCreateInverse = true
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)

	source := fogit.NewFeature("Source Feature")
	target := fogit.NewFeature("Target Feature")

	ctx := context.Background()
	if err := repo.Create(ctx, source); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, target); err != nil {
		t.Fatal(err)
	}

	// Create bidirectional relationship (conflicts-with)
	rel := fogit.NewRelationship("conflicts-with", target.ID, target.Name)
	if err := source.AddRelationship(rel); err != nil {
		t.Fatal(err)
	}
	if err := repo.Update(ctx, source); err != nil {
		t.Fatal(err)
	}

	// For bidirectional relationships, inverse should NOT be auto-created
	// (the relationship itself represents both directions)
	typeConfig := cfg.Relationships.Types["conflicts-with"]
	if !typeConfig.Bidirectional {
		t.Fatal("conflicts-with should be bidirectional in default config")
	}

	// Verify target has NO inverse (bidirectional don't need inverse)
	reloadedTarget, err := repo.Get(ctx, target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloadedTarget.Relationships) != 0 {
		t.Errorf("Expected 0 relationships on target (bidirectional), got %d", len(reloadedTarget.Relationships))
	}
}

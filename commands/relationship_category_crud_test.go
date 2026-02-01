package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestRelationshipCategoryUpdateCommand_Rename(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["old-category"] = fogit.RelationshipCategory{
		Description:    "Old category",
		CycleDetection: "warn",
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories", "update", "old-category", "--name", "new-category"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("command error = %v", err)
	}

	updatedCfg := loadConfigForTest(t, fogitDir)
	if _, exists := updatedCfg.Relationships.Categories["new-category"]; !exists {
		t.Error("expected 'new-category' to exist after rename")
	}
	if _, exists := updatedCfg.Relationships.Categories["old-category"]; exists {
		t.Error("expected 'old-category' to be removed after rename")
	}
}

func TestRelationshipCategoryUpdateCommand_RenameUpdatesTypes(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["old-cat"] = fogit.RelationshipCategory{
		Description:    "Old category",
		CycleDetection: "warn",
	}
	cfg.Relationships.Types["my-type"] = fogit.RelationshipTypeConfig{
		Category:    "old-cat",
		Description: "Type in old category",
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories", "update", "old-cat", "--name", "new-cat"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("command error = %v", err)
	}

	updatedCfg := loadConfigForTest(t, fogitDir)
	if updatedCfg.Relationships.Types["my-type"].Category != "new-cat" {
		t.Errorf("expected type to reference 'new-cat', got '%s'", updatedCfg.Relationships.Types["my-type"].Category)
	}
}

func TestRelationshipCategoryUpdateCommand_UpdateDescription(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["my-category"] = fogit.RelationshipCategory{
		Description:    "Original",
		CycleDetection: "warn",
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories", "update", "my-category", "--description", "Updated description"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("command error = %v", err)
	}

	updatedCfg := loadConfigForTest(t, fogitDir)
	if updatedCfg.Relationships.Categories["my-category"].Description != "Updated description" {
		t.Errorf("expected description 'Updated description', got '%s'", updatedCfg.Relationships.Categories["my-category"].Description)
	}
}

func TestRelationshipCategoryUpdateCommand_CategoryNotFound(t *testing.T) {
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
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories", "update", "nonexistent", "--description", "test"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err == nil {
		t.Error("expected error for nonexistent category")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestRelationshipCategoryDeleteCommand_DeleteEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["to-delete"] = fogit.RelationshipCategory{
		Description:    "Category to delete",
		CycleDetection: "warn",
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories", "delete", "to-delete"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("command error = %v", err)
	}

	updatedCfg := loadConfigForTest(t, fogitDir)
	if _, exists := updatedCfg.Relationships.Categories["to-delete"]; exists {
		t.Error("expected 'to-delete' category to be removed")
	}
}

func TestRelationshipCategoryDeleteCommand_WithTypesFailsWithoutFlag(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["has-types"] = fogit.RelationshipCategory{
		Description:    "Category with types",
		CycleDetection: "warn",
	}
	cfg.Relationships.Types["some-type"] = fogit.RelationshipTypeConfig{
		Category:    "has-types",
		Description: "Type in category",
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories", "delete", "has-types"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err == nil {
		t.Error("expected error when deleting category with types")
	}
	if err != nil && !strings.Contains(err.Error(), "types exist") {
		t.Errorf("expected 'types exist' error, got: %v", err)
	}
}

func TestRelationshipCategoryDeleteCommand_MoveTypesTo(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["source-cat"] = fogit.RelationshipCategory{
		Description:    "Source category",
		CycleDetection: "warn",
	}
	cfg.Relationships.Categories["target-cat"] = fogit.RelationshipCategory{
		Description:    "Target category",
		CycleDetection: "warn",
	}
	cfg.Relationships.Types["moving-type"] = fogit.RelationshipTypeConfig{
		Category:    "source-cat",
		Description: "Type to move",
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories", "delete", "source-cat", "--move-types-to", "target-cat"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("command error = %v", err)
	}

	updatedCfg := loadConfigForTest(t, fogitDir)
	if _, exists := updatedCfg.Relationships.Categories["source-cat"]; exists {
		t.Error("expected 'source-cat' to be deleted")
	}
	if updatedCfg.Relationships.Types["moving-type"].Category != "target-cat" {
		t.Errorf("expected type to be moved to 'target-cat', got '%s'", updatedCfg.Relationships.Types["moving-type"].Category)
	}
}

func TestRelationshipCategoryDeleteCommand_CategoryNotFound(t *testing.T) {
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
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "categories", "delete", "nonexistent"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err == nil {
		t.Error("expected error for nonexistent category")
	}
}

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

// loadConfigForTest loads config from a test fogit directory
func loadConfigForTest(t *testing.T, fogitDir string) *fogit.Config {
	t.Helper()
	cfg, err := config.Load(fogitDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	return cfg
}

func TestRelationshipTypeUpdateCommand_Rename(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config with a type to rename
	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["custom"] = fogit.RelationshipCategory{
		Description:    "Custom category",
		CycleDetection: "warn",
	}
	cfg.Relationships.Types["old-type"] = fogit.RelationshipTypeConfig{
		Category:    "custom",
		Description: "Original type",
	}
	if err := config.Save(fogitDir, cfg); err != nil {
		t.Fatal(err)
	}

	// Create a feature to ensure repo is valid
	repo := storage.NewFileRepository(fogitDir)
	feature := fogit.NewFeature("Test Feature")
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	// Run rename command
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "types", "update", "old-type", "--name", "new-type"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	if err != nil {
		t.Fatalf("command error = %v, output: %s", err, output)
	}

	// Verify rename happened
	updatedCfg := loadConfigForTest(t, fogitDir)
	if _, exists := updatedCfg.Relationships.Types["new-type"]; !exists {
		t.Error("expected 'new-type' to exist after rename")
	}
	if _, exists := updatedCfg.Relationships.Types["old-type"]; exists {
		t.Error("expected 'old-type' to be removed after rename")
	}
}

func TestRelationshipTypeUpdateCommand_UpdateDescription(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["custom"] = fogit.RelationshipCategory{
		Description:    "Custom category",
		CycleDetection: "warn",
	}
	cfg.Relationships.Types["my-type"] = fogit.RelationshipTypeConfig{
		Category:    "custom",
		Description: "Original description",
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
	rootCmd.SetArgs([]string{"-C", tmpDir, "types", "update", "my-type", "--description", "New description"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("command error = %v", err)
	}

	updatedCfg := loadConfigForTest(t, fogitDir)
	if updatedCfg.Relationships.Types["my-type"].Description != "New description" {
		t.Errorf("expected description 'New description', got '%s'", updatedCfg.Relationships.Types["my-type"].Description)
	}
}

func TestRelationshipTypeUpdateCommand_TypeNotFound(t *testing.T) {
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
	rootCmd.SetArgs([]string{"-C", tmpDir, "types", "update", "nonexistent", "--description", "test"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err == nil {
		t.Error("expected error for nonexistent type")
	}
	if err != nil && !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestRelationshipTypeDeleteCommand_DeleteType(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["custom"] = fogit.RelationshipCategory{
		Description:    "Custom category",
		CycleDetection: "warn",
	}
	cfg.Relationships.Types["to-delete"] = fogit.RelationshipTypeConfig{
		Category:    "custom",
		Description: "Type to delete",
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
	rootCmd.SetArgs([]string{"-C", tmpDir, "types", "delete", "to-delete"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("command error = %v", err)
	}

	updatedCfg := loadConfigForTest(t, fogitDir)
	if _, exists := updatedCfg.Relationships.Types["to-delete"]; exists {
		t.Error("expected 'to-delete' type to be removed")
	}
}

func TestRelationshipTypeDeleteCommand_DeleteWithInverse(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(fogitDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := fogit.DefaultConfig()
	cfg.Relationships.Categories["custom"] = fogit.RelationshipCategory{
		Description:    "Custom category",
		CycleDetection: "warn",
	}
	cfg.Relationships.Types["primary"] = fogit.RelationshipTypeConfig{
		Category:    "custom",
		Inverse:     "inverse",
		Description: "Primary type",
	}
	cfg.Relationships.Types["inverse"] = fogit.RelationshipTypeConfig{
		Category:    "custom",
		Inverse:     "primary",
		Description: "Inverse type",
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
	rootCmd.SetArgs([]string{"-C", tmpDir, "types", "delete", "primary"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("command error = %v", err)
	}

	updatedCfg := loadConfigForTest(t, fogitDir)
	if _, exists := updatedCfg.Relationships.Types["primary"]; exists {
		t.Error("expected 'primary' type to be removed")
	}
	if _, exists := updatedCfg.Relationships.Types["inverse"]; exists {
		t.Error("expected 'inverse' type to also be removed")
	}
}

func TestRelationshipTypeDeleteCommand_TypeNotFound(t *testing.T) {
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
	rootCmd.SetArgs([]string{"-C", tmpDir, "types", "delete", "nonexistent"})
	err := ExecuteRootCmd()

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	if err == nil {
		t.Error("expected error for nonexistent type")
	}
}

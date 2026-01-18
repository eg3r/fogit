package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eg3r/fogit/internal/config"
)

func TestInitializeRepository(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Run init
	err := InitializeRepository(tmpDir)
	if err != nil {
		t.Fatalf("InitializeRepository() failed: %v", err)
	}

	// Verify .fogit directory created
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if _, err := os.Stat(fogitDir); os.IsNotExist(err) {
		t.Error(".fogit directory not created")
	}

	// Verify features directory created
	featuresDir := filepath.Join(fogitDir, "features")
	if _, err := os.Stat(featuresDir); os.IsNotExist(err) {
		t.Error("features directory not created")
	}

	// Verify hooks directory created
	hooksDir := filepath.Join(fogitDir, "hooks")
	if _, err := os.Stat(hooksDir); os.IsNotExist(err) {
		t.Error("hooks directory not created")
	}

	// Verify metadata directory created
	metadataDir := filepath.Join(fogitDir, "metadata")
	if _, err := os.Stat(metadataDir); os.IsNotExist(err) {
		t.Error("metadata directory not created")
	}

	// Verify config file created
	configPath := filepath.Join(fogitDir, "config.yml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config.yml not created")
	}

	// Verify config content is valid
	loadedCfg, err := config.Load(fogitDir)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Check repository name is set
	if loadedCfg.Repository.Name == "" {
		t.Error("repository name not set in config")
	}

	// Check workflow defaults
	if loadedCfg.Workflow.Mode != "branch-per-feature" {
		t.Errorf("workflow.mode = %q, want %q", loadedCfg.Workflow.Mode, "branch-per-feature")
	}
}

func TestInitializeRepository_AlreadyInitialized(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize first time
	err := InitializeRepository(tmpDir)
	if err != nil {
		t.Fatalf("first InitializeRepository() failed: %v", err)
	}

	// Try to initialize again
	err = InitializeRepository(tmpDir)
	if err == nil {
		t.Error("expected error when already initialized, got nil")
	}
}

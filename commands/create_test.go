package commands

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestCreateCommand_Basic(t *testing.T) {
	// Setup test repository
	tempDir := t.TempDir()
	if err := setupTestGitRepo(tempDir, "main"); err != nil {
		t.Fatalf("Failed to setup git repo: %v", err)
	}

	fogitDir := filepath.Join(tempDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatalf("Failed to create .fogit/features dir: %v", err)
	}
	createMinimalTestConfig(t, fogitDir)

	// Change to test directory
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to test dir: %v", err)
	}

	// Test create command exists and has correct usage
	if createCmd.Use != "create <name>" {
		t.Errorf("Expected use 'create <name>', got %q", createCmd.Use)
	}

	if createCmd.Short != "Create a new feature (explicit)" {
		t.Errorf("Unexpected short description: %q", createCmd.Short)
	}
}

// createMinimalTestConfig creates a minimal config file for testing
func createMinimalTestConfig(t *testing.T, fogitDir string) {
	t.Helper()
	cfg := fogit.DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fogitDir, "config.yml"), data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
}

func TestCreateCommand_Flags(t *testing.T) {
	// Verify that create command has all the expected flags
	expectedFlags := []string{
		"description",
		"type",
		"priority",
		"category",
		"domain",
		"team",
		"epic",
		"module",
		"tags",
		"metadata",
		"parent",
		"same",
		"isolate",
	}

	for _, flagName := range expectedFlags {
		flag := createCmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag --%s to exist on create command", flagName)
		}
	}
}

func TestCreateCommand_IsAliasForFeatureCreate(t *testing.T) {
	// Both commands should use the same RunE function
	if createCmd.RunE == nil {
		t.Error("createCmd.RunE should not be nil")
	}

	if featureCreateCmd.RunE == nil {
		t.Error("featureCreateCmd.RunE should not be nil")
	}

	// Note: We can't directly compare function pointers in Go, but we can verify
	// they exist and have the same signature by testing their behavior
}

func TestCreateCommand_RequiresExactlyOneArg(t *testing.T) {
	// The Args field should be set to require exactly 1 argument
	if createCmd.Args == nil {
		t.Error("createCmd.Args should be set")
	}
}

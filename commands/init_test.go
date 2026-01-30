package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestInitCommand(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Initialize Git repository first (required)
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = tmpDir
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "init"})
	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("init command failed: %v", err)
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

func TestInitAlreadyInitialized(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize Git repository first (required)
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = tmpDir
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Initialize first time
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "init"})
	err := ExecuteRootCmd()
	if err != nil {
		t.Fatalf("first init failed: %v", err)
	}

	// Try to initialize again
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "init"})
	err = ExecuteRootCmd()
	if err == nil {
		t.Error("init should fail when already initialized")
	}

	// Check error message contains expected prefix (path may vary due to symlinks on macOS)
	if err != nil && !strings.Contains(err.Error(), "fogit repository already initialized in") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitCreatesValidConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize Git repository first (required)
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = tmpDir
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "init"})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Load and validate config
	fogitDir := filepath.Join(tmpDir, ".fogit")
	cfg, err := config.Load(fogitDir)
	if err != nil {
		t.Fatalf("config.Load() failed: %v", err)
	}

	// Verify all default config sections exist
	tests := []struct {
		name  string
		check func() bool
	}{
		{
			name:  "repository.name is set",
			check: func() bool { return cfg.Repository.Name != "" },
		},
		{
			name:  "workflow.mode is set",
			check: func() bool { return cfg.Workflow.Mode != "" },
		},
		{
			name:  "workflow.base_branch is set",
			check: func() bool { return cfg.Workflow.BaseBranch != "" },
		},
		{
			name:  "auto_commit is set",
			check: func() bool { return cfg.AutoCommit == true },
		},
		{
			name:  "ui.default_group_by is set",
			check: func() bool { return cfg.UI.DefaultGroupBy != "" },
		},
		{
			name:  "relationships.types is initialized",
			check: func() bool { return cfg.Relationships.Types != nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check() {
				t.Errorf("config validation failed: %s", tt.name)
			}
		})
	}
}

func TestInitDirectoryPermissions(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize Git repository first (required)
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = tmpDir
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "init"})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Check directory permissions
	fogitDir := filepath.Join(tmpDir, ".fogit")
	info, err := os.Stat(fogitDir)
	if err != nil {
		t.Fatalf("failed to stat .fogit: %v", err)
	}

	if !info.IsDir() {
		t.Error(".fogit is not a directory")
	}

	// Verify we can write to features directory
	featuresDir := filepath.Join(fogitDir, "features")
	testFile := filepath.Join(featuresDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Errorf("cannot write to features directory: %v", err)
	}
}

func TestInitConfigMatchesDefault(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize Git repository first (required)
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = tmpDir
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "init"})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Load created config
	fogitDir := filepath.Join(tmpDir, ".fogit")
	createdCfg, err := config.Load(fogitDir)
	if err != nil {
		t.Fatalf("config.Load() failed: %v", err)
	}

	// Get default config
	defaultCfg := fogit.DefaultConfig()

	// Compare key fields (excluding repository name which is set from directory)
	if createdCfg.Workflow.Mode != defaultCfg.Workflow.Mode {
		t.Errorf("workflow.mode = %q, want %q", createdCfg.Workflow.Mode, defaultCfg.Workflow.Mode)
	}

	// BaseBranch is now dynamically detected from the environment.
	// It should be either main, master, or whatever git config init.defaultBranch is set to.
	validBaseBranches := []string{"main", "master"}
	isValidBaseBranch := false
	for _, valid := range validBaseBranches {
		if createdCfg.Workflow.BaseBranch == valid {
			isValidBaseBranch = true
			break
		}
	}
	if !isValidBaseBranch {
		t.Errorf("workflow.base_branch = %q, want one of %v", createdCfg.Workflow.BaseBranch, validBaseBranches)
	}

	if createdCfg.AutoCommit != defaultCfg.AutoCommit {
		t.Errorf("auto_commit = %v, want %v", createdCfg.AutoCommit, defaultCfg.AutoCommit)
	}

	if createdCfg.UI.DefaultGroupBy != defaultCfg.UI.DefaultGroupBy {
		t.Errorf("ui.default_group_by = %q, want %q", createdCfg.UI.DefaultGroupBy, defaultCfg.UI.DefaultGroupBy)
	}
}

func TestInitInSubdirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "project", "subfolder")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize Git repository in subdirectory (required)
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = subDir
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Reset flags and run with -C flag pointing to subdirectory
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", subDir, "init"})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}

	// Verify .fogit created in subdirectory, not parent
	fogitDir := filepath.Join(subDir, ".fogit")
	if _, err := os.Stat(fogitDir); os.IsNotExist(err) {
		t.Error(".fogit not created in subdirectory")
	}

	// Verify NOT created in parent
	parentFogit := filepath.Join(tmpDir, ".fogit")
	if _, err := os.Stat(parentFogit); err == nil {
		t.Error(".fogit incorrectly created in parent directory")
	}
}

func TestInitWithSpecialCharactersInPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory with special characters
	specialDir := filepath.Join(tmpDir, "project (v2.0) [test]")
	if err := os.MkdirAll(specialDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Initialize Git repository (required)
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = specialDir
	if err := gitCmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", specialDir, "init"})
	if err := ExecuteRootCmd(); err != nil {
		t.Fatalf("init command failed with special chars in path: %v", err)
	}

	// Verify initialization succeeded
	fogitDir := filepath.Join(specialDir, ".fogit")
	if _, err := os.Stat(fogitDir); os.IsNotExist(err) {
		t.Error(".fogit not created in directory with special characters")
	}
}

func TestInitFailsWithoutGitRepository(t *testing.T) {
	// Create temp directory WITHOUT git init
	tmpDir := t.TempDir()

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "init"})
	err := ExecuteRootCmd()

	// Should fail
	if err == nil {
		t.Fatal("init should fail when not in a Git repository")
	}

	// Check error message
	if !strings.Contains(err.Error(), "not a Git repository") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Check exit code is 6 (Git integration error)
	if exitErr, ok := err.(*fogit.ExitCodeError); ok {
		if exitErr.ExitCode != fogit.ExitGitError {
			t.Errorf("exit code = %d, want %d", exitErr.ExitCode, fogit.ExitGitError)
		}
	} else {
		t.Error("expected fogit.ExitCodeError type")
	}

	// Verify .fogit was NOT created
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if _, err := os.Stat(fogitDir); err == nil {
		t.Error(".fogit should not be created when Git repository is missing")
	}
}

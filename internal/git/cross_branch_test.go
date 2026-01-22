package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	return tmpDir
}

// createTestCommit creates a file and commits it
func createTestCommit(t *testing.T, repoPath, filename, content, message string) {
	t.Helper()

	// Create directory if needed
	dir := filepath.Dir(filepath.Join(repoPath, filename))
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	// Write file
	filePath := filepath.Join(repoPath, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Add and commit
	cmd := exec.Command("git", "add", filename)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
}

func TestListBranches(t *testing.T) {
	repoPath := setupTestRepo(t)

	// Create initial commit
	createTestCommit(t, repoPath, "README.md", "# Test", "Initial commit")

	// Create additional branches
	for _, branch := range []string{"feature/test1", "feature/test2"} {
		cmd := exec.Command("git", "branch", branch)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create branch %s: %v", branch, err)
		}
	}

	// Open repository
	repo, err := OpenRepository(repoPath)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}

	// List branches
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}

	// Should have 3 branches: main/master + 2 feature branches
	if len(branches) < 3 {
		t.Errorf("ListBranches() returned %d branches, expected at least 3", len(branches))
	}

	// Check for feature branches
	branchMap := make(map[string]bool)
	for _, b := range branches {
		branchMap[b] = true
	}

	if !branchMap["feature/test1"] {
		t.Error("ListBranches() missing feature/test1")
	}
	if !branchMap["feature/test2"] {
		t.Error("ListBranches() missing feature/test2")
	}
}

func TestGetTrunkBranch(t *testing.T) {
	repoPath := setupTestRepo(t)

	// Create initial commit
	createTestCommit(t, repoPath, "README.md", "# Test", "Initial commit")

	// Open repository
	repo, err := OpenRepository(repoPath)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}

	// Get trunk branch
	trunk, err := repo.GetTrunkBranch()
	if err != nil {
		t.Fatalf("GetTrunkBranch() error = %v", err)
	}

	// Should be either "main" or "master"
	if trunk != "main" && trunk != "master" {
		t.Errorf("GetTrunkBranch() = %q, expected 'main' or 'master'", trunk)
	}
}

func TestListFilesOnBranch(t *testing.T) {
	repoPath := setupTestRepo(t)

	// Create .fogit/features directory and feature file
	featuresDir := filepath.Join(repoPath, ".fogit", "features")
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create features directory: %v", err)
	}

	featureYAML := `id: test-feature-id
name: Test Feature
metadata:
  priority: high
versions:
  "1":
    created_at: 2025-01-01T00:00:00Z
    modified_at: 2025-01-01T00:00:00Z
`
	createTestCommit(t, repoPath, ".fogit/features/test-feature.yml", featureYAML, "Add test feature")

	// Get current branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	currentBranch := string(output[:len(output)-1]) // Remove newline

	// Open repository
	repo, err := OpenRepository(repoPath)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}

	// List files on current branch
	files, err := repo.ListFilesOnBranch(currentBranch, ".fogit/features")
	if err != nil {
		t.Fatalf("ListFilesOnBranch() error = %v", err)
	}

	if len(files) != 1 {
		t.Errorf("ListFilesOnBranch() returned %d files, expected 1", len(files))
	}

	if len(files) > 0 && files[0] != ".fogit/features/test-feature.yml" {
		t.Errorf("ListFilesOnBranch() returned %q, expected '.fogit/features/test-feature.yml'", files[0])
	}
}

func TestReadFileOnBranch(t *testing.T) {
	repoPath := setupTestRepo(t)

	// Create a feature file
	featureYAML := `id: test-feature-id
name: Test Feature
metadata:
  priority: high
versions:
  "1":
    created_at: 2025-01-01T00:00:00Z
    modified_at: 2025-01-01T00:00:00Z
`
	createTestCommit(t, repoPath, ".fogit/features/test-feature.yml", featureYAML, "Add test feature")

	// Get current branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	currentBranch := string(output[:len(output)-1])

	// Open repository
	repo, err := OpenRepository(repoPath)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}

	// Read file from branch
	content, err := repo.ReadFileOnBranch(currentBranch, ".fogit/features/test-feature.yml")
	if err != nil {
		t.Fatalf("ReadFileOnBranch() error = %v", err)
	}

	if string(content) != featureYAML {
		t.Errorf("ReadFileOnBranch() returned different content")
	}

	// Test error cases
	_, err = repo.ReadFileOnBranch(currentBranch, ".fogit/features/nonexistent.yml")
	if err != ErrFileNotFound {
		t.Errorf("ReadFileOnBranch() expected ErrFileNotFound for missing file, got %v", err)
	}

	_, err = repo.ReadFileOnBranch("nonexistent-branch", ".fogit/features/test-feature.yml")
	if err != ErrBranchNotFound {
		t.Errorf("ReadFileOnBranch() expected ErrBranchNotFound for missing branch, got %v", err)
	}
}

func TestBranchExists(t *testing.T) {
	repoPath := setupTestRepo(t)

	// Create initial commit
	createTestCommit(t, repoPath, "README.md", "# Test", "Initial commit")

	// Create a feature branch
	cmd := exec.Command("git", "branch", "feature/test")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Open repository
	repo, err := OpenRepository(repoPath)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}

	// Check existing branch
	if !repo.BranchExists("feature/test") {
		t.Error("BranchExists() returned false for existing branch")
	}

	// Check non-existing branch
	if repo.BranchExists("nonexistent") {
		t.Error("BranchExists() returned true for non-existing branch")
	}
}

func TestCrossBranchFeatureDiscovery(t *testing.T) {
	// This is an integration test that verifies the full cross-branch discovery flow
	repoPath := setupTestRepo(t)

	// Create initial commit on main/master
	createTestCommit(t, repoPath, "README.md", "# Test Project", "Initial commit")

	// Get the trunk branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	trunkBranch := string(output[:len(output)-1])

	// Create feature branch 1 and add a feature
	cmd = exec.Command("git", "checkout", "-b", "feature/auth")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature branch: %v", err)
	}

	authFeatureYAML := `id: auth-feature-id
name: User Authentication
metadata:
  priority: high
  type: software-feature
versions:
  "1":
    created_at: 2025-01-01T00:00:00Z
    modified_at: 2025-01-01T00:00:00Z
`
	createTestCommit(t, repoPath, ".fogit/features/user-authentication.yml", authFeatureYAML, "Add auth feature")

	// Switch to trunk and create feature branch 2
	cmd = exec.Command("git", "checkout", trunkBranch)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to checkout trunk: %v", err)
	}

	cmd = exec.Command("git", "checkout", "-b", "feature/oauth")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create oauth branch: %v", err)
	}

	oauthFeatureYAML := `id: oauth-feature-id
name: OAuth Provider
metadata:
  priority: medium
  type: software-feature
versions:
  "1":
    created_at: 2025-01-02T00:00:00Z
    modified_at: 2025-01-02T00:00:00Z
`
	createTestCommit(t, repoPath, ".fogit/features/oauth-provider.yml", oauthFeatureYAML, "Add oauth feature")

	// Open repository
	repo, err := OpenRepository(repoPath)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}

	// List all branches - should include trunk, feature/auth, feature/oauth
	branches, err := repo.ListBranches()
	if err != nil {
		t.Fatalf("ListBranches() error = %v", err)
	}

	branchMap := make(map[string]bool)
	for _, b := range branches {
		branchMap[b] = true
	}

	if !branchMap["feature/auth"] {
		t.Error("ListBranches() missing feature/auth")
	}
	if !branchMap["feature/oauth"] {
		t.Error("ListBranches() missing feature/oauth")
	}

	// Read feature from feature/auth branch (while we're on feature/oauth)
	content, err := repo.ReadFileOnBranch("feature/auth", ".fogit/features/user-authentication.yml")
	if err != nil {
		t.Fatalf("ReadFileOnBranch() error reading from feature/auth: %v", err)
	}

	if string(content) != authFeatureYAML {
		t.Error("ReadFileOnBranch() returned different content for auth feature")
	}

	// List files on feature/auth from feature/oauth
	files, err := repo.ListFilesOnBranch("feature/auth", ".fogit/features")
	if err != nil {
		t.Fatalf("ListFilesOnBranch() error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("ListFilesOnBranch() returned %d files, expected 1", len(files))
	}
}

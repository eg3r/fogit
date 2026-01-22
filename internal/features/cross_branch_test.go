package features

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// setupCrossBranchTestRepo creates a temporary git repository with multiple branches and features
func setupCrossBranchTestRepo(t *testing.T) (string, *git.Repository) {
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

	// Open repository
	repo, err := git.OpenRepository(tmpDir)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}

	return tmpDir, repo
}

// createTestCommitInRepo creates a file and commits it
func createTestCommitInRepo(t *testing.T, repoPath, filename, content, message string) {
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

func TestFindAcrossBranches(t *testing.T) {
	repoPath, gitRepo := setupCrossBranchTestRepo(t)

	// Create initial commit on main/master
	createTestCommitInRepo(t, repoPath, "README.md", "# Test", "Initial commit")

	// Get trunk branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	trunkBranch := string(output[:len(output)-1])

	// Create feature branch and add feature
	cmd = exec.Command("git", "checkout", "-b", "feature/auth")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature branch: %v", err)
	}

	// Create feature YAML
	authFeatureYAML := `id: auth-feature-123
name: User Authentication
description: User login and registration
metadata:
  priority: high
  type: software-feature
versions:
  "1":
    created_at: 2025-01-01T00:00:00Z
    modified_at: 2025-01-01T00:00:00Z
`
	createTestCommitInRepo(t, repoPath, ".fogit/features/user-authentication.yml", authFeatureYAML, "Add auth feature")

	// Switch back to trunk
	cmd = exec.Command("git", "checkout", trunkBranch)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to checkout trunk: %v", err)
	}

	// Create storage repository pointing to current branch (trunk - which has no features)
	fogitDir := filepath.Join(repoPath, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatalf("failed to create fogit dir: %v", err)
	}
	storageRepo := storage.NewFileRepository(fogitDir)

	// Configure minimal config
	cfg := &fogit.Config{
		FeatureSearch: fogit.FeatureSearchConfig{
			FuzzyMatch:     true,
			MinSimilarity:  60.0,
			MaxSuggestions: 5,
		},
	}

	tests := []struct {
		name            string
		identifier      string
		wantFeatureName string
		wantBranch      string
		wantErr         bool
	}{
		{
			name:            "find by ID from other branch",
			identifier:      "auth-feature-123",
			wantFeatureName: "User Authentication",
			wantBranch:      "feature/auth",
			wantErr:         false,
		},
		{
			name:            "find by name from other branch",
			identifier:      "User Authentication",
			wantFeatureName: "User Authentication",
			wantBranch:      "feature/auth",
			wantErr:         false,
		},
		{
			name:            "find by lowercase name from other branch",
			identifier:      "user authentication",
			wantFeatureName: "User Authentication",
			wantBranch:      "feature/auth",
			wantErr:         false,
		},
		{
			name:       "not found anywhere",
			identifier: "nonexistent-feature",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FindAcrossBranches(context.Background(), storageRepo, gitRepo, tt.identifier, cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindAcrossBranches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Fatal("FindAcrossBranches() returned nil result")
				}
				if result.Feature == nil {
					t.Fatal("FindAcrossBranches() returned nil feature")
				}
				if result.Feature.Name != tt.wantFeatureName {
					t.Errorf("FindAcrossBranches() feature name = %q, want %q", result.Feature.Name, tt.wantFeatureName)
				}
				if result.Branch != tt.wantBranch {
					t.Errorf("FindAcrossBranches() branch = %q, want %q", result.Branch, tt.wantBranch)
				}
			}
		})
	}
}

func TestListFeaturesAcrossBranches(t *testing.T) {
	repoPath, gitRepo := setupCrossBranchTestRepo(t)

	// Create initial commit
	createTestCommitInRepo(t, repoPath, "README.md", "# Test", "Initial commit")

	// Get trunk branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	trunkBranch := string(output[:len(output)-1])

	// Create feature on feature/auth branch
	cmd = exec.Command("git", "checkout", "-b", "feature/auth")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature/auth branch: %v", err)
	}

	authFeatureYAML := `id: auth-feature-id
name: User Authentication
metadata:
  priority: high
versions:
  "1":
    created_at: 2025-01-01T00:00:00Z
    modified_at: 2025-01-01T00:00:00Z
`
	createTestCommitInRepo(t, repoPath, ".fogit/features/auth.yml", authFeatureYAML, "Add auth feature")

	// Switch to trunk and create feature/oauth branch
	cmd = exec.Command("git", "checkout", trunkBranch)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to checkout trunk: %v", err)
	}

	cmd = exec.Command("git", "checkout", "-b", "feature/oauth")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature/oauth branch: %v", err)
	}

	oauthFeatureYAML := `id: oauth-feature-id
name: OAuth Provider
metadata:
  priority: medium
versions:
  "1":
    created_at: 2025-01-02T00:00:00Z
    modified_at: 2025-01-02T00:00:00Z
`
	createTestCommitInRepo(t, repoPath, ".fogit/features/oauth.yml", oauthFeatureYAML, "Add oauth feature")

	// Create storage repository
	fogitDir := filepath.Join(repoPath, ".fogit")
	storageRepo := storage.NewFileRepository(fogitDir)

	// List features across all branches
	crossBranchFeatures, err := ListFeaturesAcrossBranches(context.Background(), storageRepo, gitRepo)
	if err != nil {
		t.Fatalf("ListFeaturesAcrossBranches() error = %v", err)
	}

	// Should have 2 features (auth and oauth)
	if len(crossBranchFeatures) != 2 {
		t.Errorf("ListFeaturesAcrossBranches() returned %d features, expected 2", len(crossBranchFeatures))
	}

	// Verify features
	featureMap := make(map[string]*CrossBranchFeature)
	for _, cbf := range crossBranchFeatures {
		featureMap[cbf.Feature.ID] = cbf
	}

	if auth, ok := featureMap["auth-feature-id"]; ok {
		if auth.Branch != "feature/auth" {
			t.Errorf("Auth feature branch = %q, expected 'feature/auth'", auth.Branch)
		}
	} else {
		t.Error("ListFeaturesAcrossBranches() missing auth feature")
	}

	if oauth, ok := featureMap["oauth-feature-id"]; ok {
		if oauth.Branch != "feature/oauth" {
			t.Errorf("OAuth feature branch = %q, expected 'feature/oauth'", oauth.Branch)
		}
		if oauth.Feature.Name != "OAuth Provider" {
			t.Errorf("OAuth feature name = %q, expected 'OAuth Provider'", oauth.Feature.Name)
		}
	} else {
		t.Error("ListFeaturesAcrossBranches() missing oauth feature")
	}
}

func TestFindAcrossBranches_CurrentBranchFirst(t *testing.T) {
	// Test that features on the current branch are found first
	repoPath, gitRepo := setupCrossBranchTestRepo(t)

	// Create initial commit
	createTestCommitInRepo(t, repoPath, "README.md", "# Test", "Initial commit")

	// Get trunk branch name
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}
	trunkBranch := string(output[:len(output)-1])

	// Create a feature on trunk
	trunkFeatureYAML := `id: trunk-feature-id
name: Trunk Feature
metadata:
  priority: low
versions:
  "1":
    created_at: 2025-01-01T00:00:00Z
    modified_at: 2025-01-01T00:00:00Z
`
	createTestCommitInRepo(t, repoPath, ".fogit/features/trunk-feature.yml", trunkFeatureYAML, "Add trunk feature")

	// Create storage repository
	fogitDir := filepath.Join(repoPath, ".fogit")
	storageRepo := storage.NewFileRepository(fogitDir)

	cfg := &fogit.Config{
		FeatureSearch: fogit.FeatureSearchConfig{
			FuzzyMatch:     true,
			MinSimilarity:  60.0,
			MaxSuggestions: 5,
		},
	}

	// Find feature that exists on current branch
	result, err := FindAcrossBranches(context.Background(), storageRepo, gitRepo, "Trunk Feature", cfg)
	if err != nil {
		t.Fatalf("FindAcrossBranches() error = %v", err)
	}

	if result.Feature.Name != "Trunk Feature" {
		t.Errorf("FindAcrossBranches() feature name = %q, expected 'Trunk Feature'", result.Feature.Name)
	}

	if result.Branch != trunkBranch {
		t.Errorf("FindAcrossBranches() branch = %q, expected %q", result.Branch, trunkBranch)
	}

	if result.IsRemote {
		t.Error("FindAcrossBranches() IsRemote = true, expected false for local branch")
	}
}

func TestMatchesIdentifier(t *testing.T) {
	feature := fogit.NewFeature("Test Feature")
	feature.ID = "test-id-123"

	tests := []struct {
		name       string
		identifier string
		want       bool
	}{
		{
			name:       "match by ID",
			identifier: "test-id-123",
			want:       true,
		},
		{
			name:       "match by exact name",
			identifier: "Test Feature",
			want:       true,
		},
		{
			name:       "match by lowercase name",
			identifier: "test feature",
			want:       true,
		},
		{
			name:       "match by uppercase name",
			identifier: "TEST FEATURE",
			want:       true,
		},
		{
			name:       "no match - different name",
			identifier: "Other Feature",
			want:       false,
		},
		{
			name:       "no match - partial name",
			identifier: "Test",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesIdentifier(feature, tt.identifier)
			if got != tt.want {
				t.Errorf("matchesIdentifier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsYAMLFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{".fogit/features/auth.yml", true},
		{".fogit/features/auth.yaml", true},
		{".fogit/features/auth.YML", true},
		{".fogit/features/auth.YAML", true},
		{".fogit/features/auth.json", false},
		{".fogit/config", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isYAMLFile(tt.path)
			if got != tt.want {
				t.Errorf("isYAMLFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestExtractLocalBranchName(t *testing.T) {
	tests := []struct {
		remote string
		want   string
	}{
		{"origin/feature/auth", "feature/auth"},
		{"origin/main", "main"},
		{"upstream/develop", "develop"},
		{"main", "main"},
	}

	for _, tt := range tests {
		t.Run(tt.remote, func(t *testing.T) {
			got := extractLocalBranchName(tt.remote)
			if got != tt.want {
				t.Errorf("extractLocalBranchName(%q) = %q, want %q", tt.remote, got, tt.want)
			}
		})
	}
}

package common

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsTrunkBranch(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{"main is trunk", "main", true},
		{"master is trunk", "master", true},
		{"trunk is trunk", "trunk", true},
		{"feature branch is not trunk", "feature/auth", false},
		{"develop is not trunk", "develop", false},
		{"empty is not trunk", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTrunkBranch(tt.branch); got != tt.want {
				t.Errorf("IsTrunkBranch(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

func TestTrunkBranchCandidates(t *testing.T) {
	// Verify the candidates are in the expected order
	if len(TrunkBranchCandidates) != 3 {
		t.Errorf("TrunkBranchCandidates length = %d, want 3", len(TrunkBranchCandidates))
	}
	if TrunkBranchCandidates[0] != "main" {
		t.Errorf("TrunkBranchCandidates[0] = %q, want %q", TrunkBranchCandidates[0], "main")
	}
	if TrunkBranchCandidates[1] != "master" {
		t.Errorf("TrunkBranchCandidates[1] = %q, want %q", TrunkBranchCandidates[1], "master")
	}
	if TrunkBranchCandidates[2] != "trunk" {
		t.Errorf("TrunkBranchCandidates[2] = %q, want %q", TrunkBranchCandidates[2], "trunk")
	}
}

func TestDetectTrunkBranch(t *testing.T) {
	// Create a temp directory with a git repo
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Get the detected trunk branch
	branch := DetectTrunkBranch(tmpDir)

	// Should be either main or master (depends on git version/config)
	if branch != "main" && branch != "master" {
		t.Errorf("DetectTrunkBranch() = %q, want 'main' or 'master'", branch)
	}
}

func TestDetectTrunkBranch_NonGitDir(t *testing.T) {
	// Non-git directory should fallback to git's init.defaultBranch or "main"
	tmpDir := t.TempDir()

	branch := DetectTrunkBranch(tmpDir)
	// Should be main or master (depends on git config init.defaultBranch)
	if branch != "main" && branch != "master" {
		t.Errorf("DetectTrunkBranch() on non-git dir = %q, want 'main' or 'master'", branch)
	}
}

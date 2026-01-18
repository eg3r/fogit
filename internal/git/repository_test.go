package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestIsGitRepository(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fogit-git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Should not be a git repo initially
	if IsGitRepository(tmpDir) {
		t.Error("Expected false for non-git directory")
	}

	// Initialize git repo
	_, err = git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Should be a git repo now
	if !IsGitRepository(tmpDir) {
		t.Error("Expected true for git directory")
	}
}

func TestFindGitRoot(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fogit-git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	_, err = git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "sub", "deep")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Should find git root from subdirectory
	root, err := FindGitRoot(subDir)
	if err != nil {
		t.Fatalf("Failed to find git root: %v", err)
	}

	if root != tmpDir {
		t.Errorf("Expected root %q, got %q", tmpDir, root)
	}
}

func TestOpenRepository(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fogit-git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Should fail for non-git directory
	_, err = OpenRepository(tmpDir)
	if err != ErrNotGitRepo {
		t.Errorf("Expected ErrNotGitRepo, got %v", err)
	}

	// Initialize git repo
	_, err = git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Should succeed for git directory
	repo, err := OpenRepository(tmpDir)
	if err != nil {
		t.Fatalf("Failed to open repository: %v", err)
	}

	if repo == nil {
		t.Error("Expected non-nil repository")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fogit-git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	_, err = git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Create initial commit
	repo, err := git.PlainOpen(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	// Create a file
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Add("test.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Open with our wrapper
	fogitRepo, err := OpenRepository(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Get current branch
	branch, err := fogitRepo.GetCurrentBranch()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	// Default branch should be "master" or "main"
	if branch != "master" && branch != "main" {
		t.Errorf("Expected 'master' or 'main', got %q", branch)
	}
}

func TestGetUserConfig(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fogit-git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	_, err = git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Open with our wrapper
	repo, err := OpenRepository(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Get user config (might be empty in test environment)
	_, _, err = repo.GetUserConfig()
	if err != nil {
		t.Fatalf("Failed to get user config: %v", err)
	}

	// Just verify no error - values might be empty
}

func TestGetLog(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fogit-git-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	gitRepo, err := git.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Configure git user
	cfg, err := gitRepo.Config()
	if err != nil {
		t.Fatal(err)
	}
	cfg.User.Name = "Test User"
	cfg.User.Email = "test@example.com"
	if err := gitRepo.SetConfig(cfg); err != nil {
		t.Fatal(err)
	}

	// Get worktree
	w, err := gitRepo.Worktree()
	if err != nil {
		t.Fatal(err)
	}

	// Create test file and commit
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content 1"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Add("test.txt"); err != nil {
		t.Fatal(err)
	}
	commit1, err := w.Commit("First commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Alice",
			Email: "alice@example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Second commit by different author
	if err := os.WriteFile(testFile, []byte("content 2"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Add("test.txt"); err != nil {
		t.Fatal(err)
	}
	commit2, err := w.Commit("Second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Bob",
			Email: "bob@example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Third commit by Alice
	if err := os.WriteFile(testFile, []byte("content 3"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Add("test.txt"); err != nil {
		t.Fatal(err)
	}
	_, err = w.Commit("Third commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Alice",
			Email: "alice@example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Open with our wrapper
	repo, err := OpenRepository(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get all commits", func(t *testing.T) {
		logs, err := repo.GetLog("", "", nil, 0)
		if err != nil {
			t.Fatalf("GetLog failed: %v", err)
		}

		if len(logs) != 3 {
			t.Errorf("Expected 3 commits, got %d", len(logs))
		}

		// Verify commit order (newest first)
		if logs[0].Message != "Third commit" {
			t.Errorf("Expected 'Third commit' first, got %q", logs[0].Message)
		}
		if logs[2].Message != "First commit" {
			t.Errorf("Expected 'First commit' last, got %q", logs[2].Message)
		}
	})

	t.Run("filter by author", func(t *testing.T) {
		logs, err := repo.GetLog("", "alice", nil, 0)
		if err != nil {
			t.Fatalf("GetLog failed: %v", err)
		}

		if len(logs) != 2 {
			t.Errorf("Expected 2 commits by Alice, got %d", len(logs))
		}

		for _, log := range logs {
			if !contains(log.Author, "alice") {
				t.Errorf("Expected Alice as author, got %q", log.Author)
			}
		}
	})

	t.Run("filter by file path", func(t *testing.T) {
		logs, err := repo.GetLog("test.txt", "", nil, 0)
		if err != nil {
			t.Fatalf("GetLog failed: %v", err)
		}

		if len(logs) != 3 {
			t.Errorf("Expected 3 commits for test.txt, got %d", len(logs))
		}
	})

	t.Run("limit results", func(t *testing.T) {
		logs, err := repo.GetLog("", "", nil, 2)
		if err != nil {
			t.Fatalf("GetLog failed: %v", err)
		}

		if len(logs) != 2 {
			t.Errorf("Expected 2 commits (limit), got %d", len(logs))
		}
	})

	t.Run("verify commit hashes", func(t *testing.T) {
		logs, err := repo.GetLog("", "", nil, 2)
		if err != nil {
			t.Fatalf("GetLog failed: %v", err)
		}

		// Check that second log entry matches commit2
		if logs[1].Hash != commit2.String() {
			t.Errorf("Expected hash %s, got %s", commit2.String(), logs[1].Hash)
		}

		// Check that third log entry matches commit1
		logs, err = repo.GetLog("", "", nil, 0)
		if err != nil {
			t.Fatalf("GetLog failed: %v", err)
		}
		if logs[2].Hash != commit1.String() {
			t.Errorf("Expected hash %s, got %s", commit1.String(), logs[2].Hash)
		}
	})

	t.Run("empty results for non-existent file", func(t *testing.T) {
		logs, err := repo.GetLog("nonexistent.txt", "", nil, 0)
		if err != nil {
			t.Fatalf("GetLog failed: %v", err)
		}

		if len(logs) != 0 {
			t.Errorf("Expected 0 commits for non-existent file, got %d", len(logs))
		}
	})
}

func TestTagErrors(t *testing.T) {
	// Test error constants are defined
	if ErrTagExists == nil {
		t.Error("ErrTagExists is nil")
	}

	if ErrTagNotFound == nil {
		t.Error("ErrTagNotFound is nil")
	}

	// Test error messages
	if ErrTagExists.Error() != "tag already exists" {
		t.Errorf("unexpected ErrTagExists message: %v", ErrTagExists)
	}

	if ErrTagNotFound.Error() != "tag not found" {
		t.Errorf("unexpected ErrTagNotFound message: %v", ErrTagNotFound)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

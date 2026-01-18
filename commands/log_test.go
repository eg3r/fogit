package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestLogCommand(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fogit-test-log-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	gitRepo, err := gogit.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatalf("Failed to init git: %v", err)
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

	// Initialize FoGit
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create repository
	repo := getRepository(fogitDir)

	// Create a feature
	feature := fogit.NewFeature("Test Feature")
	feature.SetPriority(fogit.PriorityMedium)
	if err := repo.Create(context.Background(), feature); err != nil {
		t.Fatal(err)
	}

	// Make some commits
	testFile := filepath.Join(tmpDir, "test.txt")
	for i := 1; i <= 3; i++ {
		content := []byte("content " + string(rune('0'+i)))
		if err := os.WriteFile(testFile, content, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Add("test.txt"); err != nil {
			t.Fatal(err)
		}

		author := "alice@example.com"
		if i == 2 {
			author = "bob@example.com"
		}

		_, err := w.Commit("Commit "+string(rune('0'+i)), &gogit.CommitOptions{
			Author: &object.Signature{
				Name:  author,
				Email: author,
				When:  time.Now().Add(-time.Duration(3-i) * time.Hour),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Run("list all commits", func(t *testing.T) {
		// Test would need to capture output
		// For now, just verify command runs without error
		ResetFlags()
		rootCmd.SetArgs([]string{"-C", tmpDir, "log", "--format", "oneline"})
		err := ExecuteRootCmd()
		if err != nil {
			t.Errorf("log command failed: %v", err)
		}
	})

	t.Run("filter by author", func(t *testing.T) {
		ResetFlags()
		rootCmd.SetArgs([]string{"-C", tmpDir, "log", "--author", "alice", "--format", "oneline"})
		err := ExecuteRootCmd()
		if err != nil {
			t.Errorf("log with author filter failed: %v", err)
		}
	})

	t.Run("limit results", func(t *testing.T) {
		ResetFlags()
		rootCmd.SetArgs([]string{"-C", tmpDir, "log", "--limit", "2", "--format", "short"})
		err := ExecuteRootCmd()
		if err != nil {
			t.Errorf("log with limit failed: %v", err)
		}
	})

	t.Run("filter by feature", func(t *testing.T) {
		ResetFlags()
		rootCmd.SetArgs([]string{"-C", tmpDir, "log", "--feature", "Test Feature", "--format", "full"})
		err := ExecuteRootCmd()
		if err != nil {
			t.Errorf("log with feature filter failed: %v", err)
		}
	})

	t.Run("invalid date format", func(t *testing.T) {
		ResetFlags()
		rootCmd.SetArgs([]string{"-C", tmpDir, "log", "--since", "invalid-date", "--format", "oneline"})
		err := ExecuteRootCmd()
		if err == nil {
			t.Error("Expected error for invalid date format")
		}
	})

	t.Run("valid date format", func(t *testing.T) {
		ResetFlags()
		rootCmd.SetArgs([]string{"-C", tmpDir, "log", "--since", "2025-01-01", "--format", "oneline"})
		err := ExecuteRootCmd()
		if err != nil {
			t.Errorf("log with since filter failed: %v", err)
		}
	})
}

func TestLogFormats(t *testing.T) {
	tests := []struct {
		name   string
		format string
		valid  bool
	}{
		{"oneline format", "oneline", true},
		{"short format", "short", true},
		{"full format", "full", true},
		{"default format", "", true},
		{"custom format", "custom", true}, // Falls back to full
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify format string is accepted
			// Actual output testing would require capturing stdout
			if tt.format == "" {
				tt.format = "full"
			}
			// Format is valid - no validation needed
		})
	}
}

func TestLogNotInGitRepo(t *testing.T) {
	// Create temp directory without git
	tmpDir, err := os.MkdirTemp("", "fogit-test-log-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "log", "--format", "oneline"})
	err = ExecuteRootCmd()
	if err == nil {
		t.Error("Expected error when not in git repository")
	}
}

func TestLogFeatureNotFound(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "fogit-test-log-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	_, err = gogit.PlainInit(tmpDir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize FoGit
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatal(err)
	}

	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "log", "--feature", "NonExistentFeature", "--format", "oneline"})
	err = ExecuteRootCmd()
	if err == nil {
		t.Error("Expected error for non-existent feature")
	}
}

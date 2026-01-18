package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// captureStdout captures stdout during execution of a function
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestFilterCommand(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create test features
	repo := storage.NewFileRepository(fogitDir)

	// Feature 1: High priority, open, security category
	f1 := fogit.NewFeature("User Authentication")
	f1.SetPriority(fogit.PriorityHigh)
	f1.SetCategory("security")
	f1.Tags = []string{"auth", "security"}
	if err := repo.Create(context.Background(), f1); err != nil {
		t.Fatal(err)
	}

	// Feature 2: Low priority, open, backend category
	f2 := fogit.NewFeature("Database Optimization")
	f2.SetPriority(fogit.PriorityLow)
	f2.SetCategory("backend")
	f2.Tags = []string{"performance", "database"}
	if err := repo.Create(context.Background(), f2); err != nil {
		t.Fatal(err)
	}

	// Feature 3: Critical priority, closed, security category
	f3 := fogit.NewFeature("Password Reset")
	f3.SetPriority(fogit.PriorityCritical)
	f3.SetCategory("security")
	f3.UpdateState(fogit.StateClosed)
	if err := repo.Create(context.Background(), f3); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		args       []string
		wantCount  int
		wantNames  []string
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:      "filter by priority high",
			args:      []string{"priority:high"},
			wantCount: 1,
			wantNames: []string{"User Authentication"},
		},
		{
			name:      "filter by state open",
			args:      []string{"state:open"},
			wantCount: 2,
			wantNames: []string{"User Authentication", "Database Optimization"},
		},
		{
			name:      "filter by state closed",
			args:      []string{"state:closed"},
			wantCount: 1,
			wantNames: []string{"Password Reset"},
		},
		{
			name:      "filter by category security",
			args:      []string{"category:security"},
			wantCount: 2,
			wantNames: []string{"User Authentication", "Password Reset"},
		},
		{
			name:      "filter by priority AND state",
			args:      []string{"priority:high AND state:open"},
			wantCount: 1,
			wantNames: []string{"User Authentication"},
		},
		{
			name:      "filter by category OR category",
			args:      []string{"category:security OR category:backend"},
			wantCount: 3,
		},
		{
			name:      "filter NOT closed",
			args:      []string{"NOT state:closed"},
			wantCount: 2,
		},
		{
			name:      "filter by name wildcard",
			args:      []string{"name:*Auth*"},
			wantCount: 1,
			wantNames: []string{"User Authentication"},
		},
		{
			name:      "filter by tags",
			args:      []string{"tags:security"},
			wantCount: 1,
			wantNames: []string{"User Authentication"},
		},
		{
			name:      "filter complex expression",
			args:      []string{"(priority:high OR priority:critical) AND category:security"},
			wantCount: 2,
			wantNames: []string{"User Authentication", "Password Reset"},
		},
		{
			name:      "filter priority comparison",
			args:      []string{"priority:>=high"},
			wantCount: 2, // high and critical
		},
		{
			name:      "filter no matches",
			args:      []string{"category:nonexistent"},
			wantCount: 0,
		},
		{
			name:       "invalid expression",
			args:       []string{"priority:"},
			wantErr:    true,
			wantErrMsg: "invalid filter expression",
		},
		{
			name:       "missing expression argument",
			args:       []string{},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags and run with -C flag
			ResetFlags()
			args := append([]string{"-C", tmpDir, "filter"}, tt.args...)
			rootCmd.SetArgs(args)

			var err error
			var output string

			if tt.wantErr {
				// For error cases, we don't need to capture stdout
				err = ExecuteRootCmd()
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.wantErrMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.wantErrMsg, err.Error())
				}
				return
			}

			// Capture stdout for successful execution
			output = captureStdout(t, func() {
				err = ExecuteRootCmd()
			})

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check for expected feature count
			if tt.wantCount == 0 {
				if !strings.Contains(output, "No features found") {
					t.Errorf("Expected 'No features found' message, got: %s", output)
				}
				return
			}

			// Check expected names are in output
			for _, name := range tt.wantNames {
				if !strings.Contains(output, name) {
					t.Errorf("Expected output to contain %q, got: %s", name, output)
				}
			}
		})
	}
}

func TestFilterCommand_JSONFormat(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create test feature
	repo := storage.NewFileRepository(fogitDir)
	f := fogit.NewFeature("Test Feature")
	f.SetPriority(fogit.PriorityHigh)
	if err := repo.Create(context.Background(), f); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "filter", "--format", "json", "priority:high"})

	var err error
	output := captureStdout(t, func() {
		err = ExecuteRootCmd()
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify JSON output
	var features []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &features); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	if len(features) != 1 {
		t.Errorf("Expected 1 feature, got %d", len(features))
	}

	// JSON uses capitalized field names from struct
	if features[0]["Name"] != "Test Feature" {
		t.Errorf("Expected feature name 'Test Feature', got %v", features[0]["Name"])
	}
}

func TestFilterCommand_InvalidFormat(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "filter", "--format", "invalid", "state:open"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Error("Expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("Expected 'invalid format' error, got: %v", err)
	}
}

func TestFilterCommand_InvalidSort(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatal(err)
	}

	// Reset flags and run with -C flag
	ResetFlags()
	rootCmd.SetArgs([]string{"-C", tmpDir, "filter", "--sort", "invalid", "state:open"})

	err := ExecuteRootCmd()
	if err == nil {
		t.Error("Expected error for invalid sort field")
	}
	if !strings.Contains(err.Error(), "invalid sort field") {
		t.Errorf("Expected 'invalid sort field' error, got: %v", err)
	}
}

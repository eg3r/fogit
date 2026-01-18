package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestStatusCommand(t *testing.T) {
	tests := []struct {
		name            string
		setupFeatures   func(dir string) error
		format          string
		wantInOutput    []string
		wantNotInOutput []string
	}{
		{
			name: "empty repository",
			setupFeatures: func(dir string) error {
				return nil
			},
			format: "text",
			wantInOutput: []string{
				"FoGit Repository Status",
				"Features: 0 total",
				"Open:        0",
				"In Progress: 0",
				"Closed:      0",
			},
		},
		{
			name: "repository with features",
			setupFeatures: func(dir string) error {
				// Create open feature
				f1 := fogit.NewFeature("Feature One")
				f1.SetPriority(fogit.PriorityHigh)
				if err := storage.WriteFeatureFile(filepath.Join(dir, "features", "feature-one.yml"), f1); err != nil {
					return err
				}

				// Create in-progress feature
				f2 := fogit.NewFeature("Feature Two")
				f2.UpdateState(fogit.StateInProgress)
				if err := storage.WriteFeatureFile(filepath.Join(dir, "features", "feature-two.yml"), f2); err != nil {
					return err
				}

				// Create closed feature
				f3 := fogit.NewFeature("Feature Three")
				f3.UpdateState(fogit.StateClosed)
				return storage.WriteFeatureFile(filepath.Join(dir, "features", "feature-three.yml"), f3)
			},
			format: "text",
			wantInOutput: []string{
				"Features: 3 total",
				"Open:        1",
				"In Progress: 1",
				"Closed:      1",
			},
		},
		{
			name: "repository with relationships",
			setupFeatures: func(dir string) error {
				f1 := fogit.NewFeature("Feature One")
				f2 := fogit.NewFeature("Feature Two")

				// Add relationship
				f1.Relationships = append(f1.Relationships, fogit.Relationship{
					ID:        "rel-1",
					Type:      "depends-on",
					TargetID:  f2.ID,
					CreatedAt: time.Now(),
				})

				if err := storage.WriteFeatureFile(filepath.Join(dir, "features", "feature-one.yml"), f1); err != nil {
					return err
				}
				return storage.WriteFeatureFile(filepath.Join(dir, "features", "feature-two.yml"), f2)
			},
			format: "text",
			wantInOutput: []string{
				"Relationships: 1 total",
			},
		},
		{
			name: "json output format",
			setupFeatures: func(dir string) error {
				f1 := fogit.NewFeature("Test Feature")
				return storage.WriteFeatureFile(filepath.Join(dir, "features", "test.yml"), f1)
			},
			format: "json",
			wantInOutput: []string{
				`"total_features": 1`,
				`"open": 1`,
				`"in_progress": 0`,
				`"closed": 0`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "fogit-status-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Create .fogit directory structure
			fogitDir := filepath.Join(tmpDir, ".fogit")
			featuresDir := filepath.Join(fogitDir, "features")
			if err := os.MkdirAll(featuresDir, 0755); err != nil {
				t.Fatalf("Failed to create features dir: %v", err)
			}

			// Create default config
			cfg := fogit.DefaultConfig()
			configPath := filepath.Join(fogitDir, "config.yml")
			if err := writeTestConfig(configPath, cfg); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			// Setup features
			if err := tt.setupFeatures(fogitDir); err != nil {
				t.Fatalf("Failed to setup features: %v", err)
			}

			// Reset flags and global config
			ResetFlags()

			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command via rootCmd with -C flag
			args := []string{"-C", tmpDir, "status"}
			if tt.format != "" && tt.format != "text" {
				args = append(args, "--format", tt.format)
			}
			rootCmd.SetArgs(args)
			err = ExecuteRootCmd()

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Fatalf("runStatus returned error: %v", err)
			}

			// Check expected strings are present
			for _, want := range tt.wantInOutput {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing expected string %q\nGot:\n%s", want, output)
				}
			}

			// Check unexpected strings are absent
			for _, notWant := range tt.wantNotInOutput {
				if strings.Contains(output, notWant) {
					t.Errorf("Output contains unexpected string %q\nGot:\n%s", notWant, output)
				}
			}
		})
	}
}

// Note: TestGetRecentChanges, TestFindFeaturesOnBranch, and TestBuildStatusReport
// tests are in internal/features/status_test.go

// Helper to write test config
func writeTestConfig(path string, cfg *fogit.Config) error {
	data := []byte(`repository:
  name: test-repo
  version: "1.0"
workflow:
  mode: branch-per-feature
  base_branch: main
`)
	return os.WriteFile(path, data, 0600)
}

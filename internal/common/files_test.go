package common

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestShouldSkipFogitFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "fogit feature file",
			path:     ".fogit/features/test.yml",
			expected: true,
		},
		{
			name:     "fogit config file",
			path:     ".fogit/config.yml",
			expected: true,
		},
		{
			name:     "fogit root directory",
			path:     ".fogit/",
			expected: true,
		},
		{
			name:     "regular source file",
			path:     "src/auth/login.go",
			expected: false,
		},
		{
			name:     "test file",
			path:     "test/auth_test.go",
			expected: false,
		},
		{
			name:     "file starting with dot but not fogit",
			path:     ".gitignore",
			expected: false,
		},
		{
			name:     "windows style path",
			path:     ".fogit\\features\\test.yml",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldSkipFogitFile(tt.path)
			if result != tt.expected {
				t.Errorf("ShouldSkipFogitFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDirExists(t *testing.T) {
	// Test with temp directory
	tmpDir := t.TempDir()

	if !DirExists(tmpDir) {
		t.Errorf("DirExists() should return true for existing directory")
	}

	if DirExists(tmpDir + "/nonexistent") {
		t.Errorf("DirExists() should return false for nonexistent path")
	}
}

func TestFileExists(t *testing.T) {
	// Test with temp directory (should return false for directory)
	tmpDir := t.TempDir()

	if FileExists(tmpDir) {
		t.Errorf("FileExists() should return false for directory")
	}

	if FileExists(tmpDir + "/nonexistent.txt") {
		t.Errorf("FileExists() should return false for nonexistent file")
	}
}

func TestIsYAMLFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"test.yml", true},
		{"test.yaml", true},
		{"test.json", false},
		{"test.go", false},
		{"yml", false},
	}

	for _, tt := range tests {
		result := IsYAMLFile(tt.filename)
		if result != tt.expected {
			t.Errorf("IsYAMLFile(%q) = %v, want %v", tt.filename, result, tt.expected)
		}
	}
}

func TestIsJSONFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"test.json", true},
		{"test.yml", false},
		{"test.yaml", false},
		{"json", false},
	}

	for _, tt := range tests {
		result := IsJSONFile(tt.filename)
		if result != tt.expected {
			t.Errorf("IsJSONFile(%q) = %v, want %v", tt.filename, result, tt.expected)
		}
	}
}

func TestParseMetadataFlags(t *testing.T) {
	tests := []struct {
		name     string
		pairs    []string
		expected map[string]interface{}
	}{
		{
			name:     "empty pairs",
			pairs:    []string{},
			expected: nil,
		},
		{
			name:     "single pair",
			pairs:    []string{"key=value"},
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "multiple pairs",
			pairs:    []string{"key1=value1", "key2=value2"},
			expected: map[string]interface{}{"key1": "value1", "key2": "value2"},
		},
		{
			name:     "empty value ignored",
			pairs:    []string{"key="},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseMetadataFlags(tt.pairs)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("ParseMetadataFlags() = %v, want nil", result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("ParseMetadataFlags() length = %d, want %d", len(result), len(tt.expected))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("ParseMetadataFlags()[%q] = %v, want %v", k, result[k], v)
				}
			}
		})
	}
}

func TestValidateOutputPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty path is valid (stdout)",
			path:    "",
			wantErr: false,
		},
		{
			name:    "simple filename is valid",
			path:    "output.json",
			wantErr: false,
		},
		{
			name:    "subdirectory is valid",
			path:    "exports/output.json",
			wantErr: false,
		},
		{
			name:    "path traversal with double dots",
			path:    "../output.json",
			wantErr: true,
			errMsg:  "cannot contain '..'",
		},
		{
			name:    "deep path traversal",
			path:    "foo/../../../etc/passwd",
			wantErr: true,
			errMsg:  "cannot contain '..'",
		},
		{
			name:    "path traversal at end",
			path:    "foo/bar/..",
			wantErr: true,
			errMsg:  "cannot contain '..'",
		},
		{
			name:    "windows style path traversal",
			path:    "..\\output.json",
			wantErr: true,
			errMsg:  "cannot contain '..'",
		},
		{
			name:    "hidden path traversal",
			path:    "safe/../../malicious",
			wantErr: true,
			errMsg:  "cannot contain '..'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" {
				if err == nil || !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("ValidateOutputPath(%q) error = %v, want error containing %q", tt.path, err, tt.errMsg)
				}
			}
		})
	}
}

// containsSubstring checks if s contains substr (case-insensitive not needed here)
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestAtomicWriteFile(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.txt")
		content := "test content"

		err := AtomicWriteFile(outputPath, func(f *os.File) error {
			_, err := f.WriteString(content)
			return err
		})
		if err != nil {
			t.Fatalf("AtomicWriteFile() error = %v", err)
		}

		// Verify file exists and has correct content
		data, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}
		if string(data) != content {
			t.Errorf("File content = %q, want %q", string(data), content)
		}
	})

	t.Run("write failure leaves no file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.txt")
		writeErr := fmt.Errorf("simulated write error")

		err := AtomicWriteFile(outputPath, func(f *os.File) error {
			return writeErr
		})
		if err != writeErr {
			t.Errorf("AtomicWriteFile() error = %v, want %v", err, writeErr)
		}

		// Verify no file was created
		if FileExists(outputPath) {
			t.Error("Output file should not exist after write failure")
		}

		// Verify no temp files left behind
		entries, _ := os.ReadDir(tmpDir)
		for _, entry := range entries {
			if entry.Name() != "." && entry.Name() != ".." {
				t.Errorf("Unexpected file left behind: %s", entry.Name())
			}
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "subdir", "nested", "output.txt")

		err := AtomicWriteFile(outputPath, func(f *os.File) error {
			_, err := f.WriteString("test")
			return err
		})
		if err != nil {
			t.Fatalf("AtomicWriteFile() error = %v", err)
		}

		if !FileExists(outputPath) {
			t.Error("Output file should exist")
		}
	})

	t.Run("overwrites existing file atomically", func(t *testing.T) {
		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "output.txt")

		// Create initial file
		if err := os.WriteFile(outputPath, []byte("old content"), 0644); err != nil {
			t.Fatalf("Failed to create initial file: %v", err)
		}

		// Overwrite atomically
		newContent := "new content"
		err := AtomicWriteFile(outputPath, func(f *os.File) error {
			_, err := f.WriteString(newContent)
			return err
		})
		if err != nil {
			t.Fatalf("AtomicWriteFile() error = %v", err)
		}

		// Verify new content
		data, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}
		if string(data) != newContent {
			t.Errorf("File content = %q, want %q", string(data), newContent)
		}
	})
}

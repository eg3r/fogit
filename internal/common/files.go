package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DirExists checks if a directory exists at the given path.
// Returns false if the path doesn't exist or is not a directory.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// FileExists checks if a file exists at the given path.
// Returns false if the path doesn't exist or is a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// PathExists checks if a path (file or directory) exists.
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsYAMLFile checks if a filename has a .yml or .yaml extension.
func IsYAMLFile(filename string) bool {
	return strings.HasSuffix(filename, ".yml") || strings.HasSuffix(filename, ".yaml")
}

// IsJSONFile checks if a filename has a .json extension.
func IsJSONFile(filename string) bool {
	return strings.HasSuffix(filename, ".json")
}

// ParseMetadataFlags parses key=value pairs into a metadata map.
// Empty values are ignored.
func ParseMetadataFlags(pairs []string) map[string]interface{} {
	if len(pairs) == 0 {
		return nil
	}

	metadata := make(map[string]interface{})
	for _, pair := range pairs {
		key, value := SplitKeyValueEquals(pair)
		if value != "" {
			metadata[key] = value
		}
	}

	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

// ShouldSkipFogitFile returns true if the given path is inside the .fogit directory
// and should be excluded from auto-linking or other file operations.
func ShouldSkipFogitFile(path string) bool {
	return strings.HasPrefix(path, ".fogit/") || strings.HasPrefix(path, ".fogit\\")
}

// ValidateOutputPath validates that an output path is safe to write to.
// It prevents path traversal attacks by ensuring:
// 1. The path doesn't contain traversal sequences (..)
// 2. The resolved absolute path is within the current working directory
//
// Returns an error if the path is unsafe, nil if safe.
func ValidateOutputPath(outputPath string) error {
	if outputPath == "" {
		return nil // Empty path means stdout, which is safe
	}

	// Clean the path to resolve any . or .. components
	cleanPath := filepath.Clean(outputPath)

	// Check for explicit path traversal sequences in the original path
	// This catches attempts like "foo/../../../etc/passwd"
	if strings.Contains(outputPath, "..") {
		return fmt.Errorf("output path cannot contain '..': %s", outputPath)
	}

	// Get absolute paths for comparison
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	// Get current working directory as the allowed base
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Resolve symlinks for cwd (fixes macOS /var -> /private/var issue)
	// Do this for cwd since it always exists
	resolvedCwd, err := filepath.EvalSymlinks(cwd)
	if err == nil {
		cwd = resolvedCwd
	}

	// For absPath, try to resolve symlinks for the parent directory
	// since the file itself may not exist yet
	parentDir := filepath.Dir(absPath)
	resolvedParent, err := filepath.EvalSymlinks(parentDir)
	if err == nil {
		absPath = filepath.Join(resolvedParent, filepath.Base(absPath))
	}

	// Ensure the absolute path is within the current working directory
	// We add a separator to prevent prefix matching issues
	// e.g., /home/user/project vs /home/user/project-other
	cwdWithSep := cwd + string(filepath.Separator)
	if !strings.HasPrefix(absPath, cwdWithSep) && absPath != cwd {
		// Also allow files directly in cwd (not in subdirectory)
		if filepath.Dir(absPath) != cwd {
			return fmt.Errorf("output path must be within current directory: %s is outside %s", absPath, cwd)
		}
	}

	return nil
}

// AtomicWriteFile writes data to a file atomically using a temp file + rename pattern.
// This ensures that on failure, no partial/corrupted file is left behind.
// The writeFn is called with the temp file to perform the actual write.
func AtomicWriteFile(outputPath string, writeFn func(*os.File) error) error {
	// Create temp file in the same directory as the target
	// (required for atomic rename to work on the same filesystem)
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".fogit-export-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on any failure path
	success := false
	defer func() {
		if !success {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Perform the write
	if err := writeFn(tmpFile); err != nil {
		return err
	}

	// Close the temp file before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename to final destination
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return fmt.Errorf("failed to rename temp file to output: %w", err)
	}

	success = true
	return nil
}

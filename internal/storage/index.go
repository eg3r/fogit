package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// IDIndex provides O(1) lookup from feature ID to filename.
// This is stored in .fogit/metadata/id_index.json
type IDIndex struct {
	// Entries maps feature ID to filename (e.g., "550e8400..." -> "user-authentication.yml")
	Entries map[string]string `json:"entries"`

	basePath string // Path to .fogit directory
	mu       sync.RWMutex
}

// indexPath returns the path to the index file
func (idx *IDIndex) indexPath() string {
	return filepath.Join(idx.basePath, "metadata", "id_index.json")
}

// NewIDIndex creates a new ID index for the given .fogit directory
func NewIDIndex(basePath string) *IDIndex {
	return &IDIndex{
		Entries:  make(map[string]string),
		basePath: basePath,
	}
}

// Load reads the index from disk. If the file doesn't exist, returns an empty index.
func (idx *IDIndex) Load() error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	path := idx.indexPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Index doesn't exist yet, start fresh
			idx.Entries = make(map[string]string)
			return nil
		}
		return fmt.Errorf("failed to read index: %w", err)
	}

	var entries map[string]string
	if err := json.Unmarshal(data, &entries); err != nil {
		// Corrupted index, rebuild will be needed
		idx.Entries = make(map[string]string)
		return nil
	}

	idx.Entries = entries
	return nil
}

// Save writes the index to disk atomically
func (idx *IDIndex) Save() error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	path := idx.indexPath()

	// Ensure metadata directory exists
	metadataDir := filepath.Dir(path)
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(idx.Entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Atomic write using temp file + rename
	tmpFile, err := os.CreateTemp(metadataDir, ".id_index-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write index: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename index file: %w", err)
	}

	return nil
}

// Get returns the filename for a given ID, or empty string if not found
func (idx *IDIndex) Get(id string) string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.Entries[id]
}

// Set adds or updates an ID -> filename mapping
func (idx *IDIndex) Set(id, filename string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.Entries[id] = filename
}

// Delete removes an ID from the index
func (idx *IDIndex) Delete(id string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	delete(idx.Entries, id)
}

// Has checks if an ID exists in the index
func (idx *IDIndex) Has(id string) bool {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	_, exists := idx.Entries[id]
	return exists
}

// Rebuild reconstructs the index by scanning all feature files.
// This is useful for recovery or initial setup.
func (idx *IDIndex) Rebuild(featuresDir string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Clear existing entries
	idx.Entries = make(map[string]string)

	// Check if features directory exists
	if _, err := os.Stat(featuresDir); os.IsNotExist(err) {
		return nil // No features yet
	}

	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		return fmt.Errorf("failed to read features directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !isYAMLFile(entry.Name()) {
			continue
		}

		path := filepath.Join(featuresDir, entry.Name())
		feature, err := ReadFeatureFile(path)
		if err != nil {
			// Skip corrupted files
			continue
		}

		idx.Entries[feature.ID] = entry.Name()
	}

	return nil
}

// isYAMLFile checks if filename has a YAML extension
func isYAMLFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yml" || ext == ".yaml"
}

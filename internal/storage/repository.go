package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/eg3r/fogit/internal/common"
	"github.com/eg3r/fogit/pkg/fogit"
)

// FileRepository implements fogit.Repository using YAML files
type FileRepository struct {
	basePath string   // Path to .fogit directory
	index    *IDIndex // ID-to-filename index for O(1) lookups
	indexMu  sync.Once
}

// NewFileRepository creates a new file-based repository
func NewFileRepository(basePath string) *FileRepository {
	return &FileRepository{
		basePath: basePath,
	}
}

// getIndex returns the ID index, lazily loading/rebuilding it if needed
func (r *FileRepository) getIndex() *IDIndex {
	r.indexMu.Do(func() {
		r.index = NewIDIndex(r.basePath)
		if err := r.index.Load(); err != nil {
			// Failed to load, start fresh
			r.index = NewIDIndex(r.basePath)
		}
		// If index is empty but features exist, rebuild it
		if len(r.index.Entries) == 0 {
			_ = r.index.Rebuild(r.featuresDir())
			_ = r.index.Save() // Best effort save
		}
	})
	return r.index
}

// featuresDir returns the path to the features directory
func (r *FileRepository) featuresDir() string {
	return filepath.Join(r.basePath, "features")
}

// findFeatureFile searches for a feature file by ID using the index for O(1) lookup.
// Falls back to directory scan if index miss (rebuilds index on fallback hit).
// Returns the full path to the file, or error if not found.
func (r *FileRepository) findFeatureFile(ctx context.Context, id string) (string, error) {
	featuresDir := r.featuresDir()
	idx := r.getIndex()

	// Fast path: check index first
	if filename := idx.Get(id); filename != "" {
		path := filepath.Join(featuresDir, filename)
		// Verify file still exists
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		// File was deleted externally, remove from index
		idx.Delete(id)
		_ = idx.Save() // Best effort
	}

	// Slow path: scan directory (index miss or stale entry)
	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fogit.ErrNotFound
		}
		return "", fmt.Errorf("failed to read features directory: %w", err)
	}

	for _, entry := range entries {
		// Check for cancellation before processing each file (if context provided)
		if ctx != nil {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
			}
		}

		// Support both .yml (new) and .yaml (legacy) extensions
		if entry.IsDir() || !common.IsYAMLFile(entry.Name()) {
			continue
		}

		path := filepath.Join(featuresDir, entry.Name())
		feature, err := ReadFeatureFile(path)
		if err != nil {
			continue
		}

		if feature.ID == id {
			// Update index with found entry (cache for next lookup)
			idx.Set(id, entry.Name())
			_ = idx.Save() // Best effort
			return path, nil
		}
	}

	return "", fogit.ErrNotFound
}

// featurePath generates the path for a new feature file using slugified name
// For updates/deletes, use findFeatureFile() instead
func (r *FileRepository) featurePath(ctx context.Context, feature *fogit.Feature) (string, error) {
	return r.featurePathExcluding(ctx, feature, "")
}

// featurePathExcluding generates a path for a feature file, optionally excluding
// a filename from collision detection. Used during updates when the file already exists.
func (r *FileRepository) featurePathExcluding(ctx context.Context, feature *fogit.Feature, excludeFilename string) (string, error) {
	// Check for cancellation before starting (if context provided)
	if ctx != nil {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
	}

	featuresDir := r.featuresDir()

	// Get existing filenames for collision detection
	existingFiles := make(map[string]bool)
	entries, err := os.ReadDir(featuresDir)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to read features directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() != excludeFilename {
			existingFiles[entry.Name()] = true
		}
	}

	filename := generateFilename(feature.Name, feature.ID, existingFiles)
	return filepath.Join(featuresDir, filename), nil
}

// Create creates a new feature in the repository
func (r *FileRepository) Create(ctx context.Context, feature *fogit.Feature) error {
	if feature == nil {
		return fmt.Errorf("feature cannot be nil")
	}

	if err := feature.Validate(); err != nil {
		return fmt.Errorf("invalid feature: %w", err)
	}

	// Check if feature with this ID already exists
	if existingPath, err := r.findFeatureFile(ctx, feature.ID); err == nil {
		// File exists
		if _, statErr := os.Stat(existingPath); statErr == nil {
			return fogit.ErrFeatureAlreadyExists
		}
	}

	// Generate path with slugified name
	path, err := r.featurePath(ctx, feature)
	if err != nil {
		return fmt.Errorf("failed to generate feature path: %w", err)
	}

	if err := WriteFeatureFile(path, feature); err != nil {
		return err
	}

	// Update index with new entry
	idx := r.getIndex()
	idx.Set(feature.ID, filepath.Base(path))
	_ = idx.Save() // Best effort

	return nil
}

// Get retrieves a feature by ID
func (r *FileRepository) Get(ctx context.Context, id string) (*fogit.Feature, error) {
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	path, err := r.findFeatureFile(ctx, id)
	if err != nil {
		return nil, err
	}

	return ReadFeatureFile(path)
}

// List retrieves features matching the given filter
func (r *FileRepository) List(ctx context.Context, filter *fogit.Filter) ([]*fogit.Feature, error) {
	featuresDir := r.featuresDir()

	// Check if features directory exists
	if _, err := os.Stat(featuresDir); os.IsNotExist(err) {
		return []*fogit.Feature{}, nil
	}

	entries, err := os.ReadDir(featuresDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read features directory: %w", err)
	}

	var features []*fogit.Feature
	for _, entry := range entries {
		// Check for cancellation before processing each file (if context provided)
		if ctx != nil {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
		}

		// Support both .yml (new) and .yaml (legacy) extensions
		if entry.IsDir() || !common.IsYAMLFile(entry.Name()) {
			continue
		}

		path := filepath.Join(featuresDir, entry.Name())
		feature, err := ReadFeatureFile(path)
		if err != nil {
			// Log error but continue processing other features
			continue
		}

		// Use Filter.Matches method for consistent filtering
		if filter == nil || filter.Matches(feature) {
			features = append(features, feature)
		}
	}

	return features, nil
}

// Update updates an existing feature
func (r *FileRepository) Update(ctx context.Context, feature *fogit.Feature) error {
	if feature == nil {
		return fmt.Errorf("feature cannot be nil")
	}

	if err := feature.Validate(); err != nil {
		return fmt.Errorf("invalid feature: %w", err)
	}

	// Find existing file by ID
	oldPath, err := r.findFeatureFile(ctx, feature.ID)
	if err != nil {
		return err
	}

	// Read the old feature to check if name changed
	oldFeature, err := ReadFeatureFile(oldPath)
	if err != nil {
		return fmt.Errorf("failed to read existing feature: %w", err)
	}

	// If name hasn't changed, just update in place (no rename needed)
	if oldFeature.Name == feature.Name {
		return WriteFeatureFile(oldPath, feature)
	}

	// Name changed - need to generate new path and rename file
	newPath, err := r.featurePathExcluding(ctx, feature, filepath.Base(oldPath))
	if err != nil {
		return fmt.Errorf("failed to generate feature path: %w", err)
	}

	// Write to new location
	if err := WriteFeatureFile(newPath, feature); err != nil {
		return err
	}
	// Remove old file
	if err := os.Remove(oldPath); err != nil {
		// Try to clean up the new file
		os.Remove(newPath)
		return fmt.Errorf("failed to remove old feature file: %w", err)
	}
	// Update index with new filename
	idx := r.getIndex()
	idx.Set(feature.ID, filepath.Base(newPath))
	_ = idx.Save() // Best effort
	return nil
}

// Delete removes a feature from the repository
func (r *FileRepository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("id cannot be empty")
	}

	path, err := r.findFeatureFile(ctx, id)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete feature: %w", err)
	}

	// Remove from index
	idx := r.getIndex()
	idx.Delete(id)
	_ = idx.Save() // Best effort

	return nil
}

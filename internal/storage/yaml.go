package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/eg3r/fogit/pkg/fogit"
)

// MarshalFeature serializes a feature to YAML bytes
func MarshalFeature(feature *fogit.Feature) ([]byte, error) {
	if feature == nil {
		return nil, fmt.Errorf("feature cannot be nil")
	}

	data, err := yaml.Marshal(feature)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal feature: %w", err)
	}

	return data, nil
}

// UnmarshalFeature deserializes YAML bytes to a feature
func UnmarshalFeature(data []byte) (*fogit.Feature, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	var feature fogit.Feature
	if err := yaml.Unmarshal(data, &feature); err != nil {
		return nil, fmt.Errorf("failed to unmarshal feature: %w", err)
	}

	return &feature, nil
}

// WriteFeatureFile writes a feature to a YAML file atomically
// It writes to a temp file first, then renames to ensure atomicity
func WriteFeatureFile(path string, feature *fogit.Feature) error {
	if err := feature.Validate(); err != nil {
		return fmt.Errorf("invalid feature: %w", err)
	}

	data, err := MarshalFeature(feature)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temp file
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath) // Clean up temp file on error
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// ReadFeatureFile reads a feature from a YAML file
func ReadFeatureFile(path string) (*fogit.Feature, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fogit.ErrNotFound
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	feature, err := UnmarshalFeature(data)
	if err != nil {
		return nil, err
	}

	if err := feature.Validate(); err != nil {
		return nil, fmt.Errorf("invalid feature in file: %w", err)
	}

	return feature, nil
}

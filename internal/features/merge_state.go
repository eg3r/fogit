package features

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// MergeState tracks the state of an in-progress fogit merge
type MergeState struct {
	FeatureBranch string   `yaml:"feature_branch"`
	BaseBranch    string   `yaml:"base_branch"`
	FeatureIDs    []string `yaml:"feature_ids"`
	NoDelete      bool     `yaml:"no_delete"`
	Squash        bool     `yaml:"squash"`
	ConflictFiles []string `yaml:"conflict_files,omitempty"`
}

const mergeStateFile = "MERGE_STATE"

// SaveMergeState saves the merge state to .fogit/MERGE_STATE
func SaveMergeState(fogitDir string, state *MergeState) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal merge state: %w", err)
	}

	statePath := filepath.Join(fogitDir, mergeStateFile)
	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("failed to save merge state: %w", err)
	}

	return nil
}

// LoadMergeState loads the merge state from .fogit/MERGE_STATE
func LoadMergeState(fogitDir string) (*MergeState, error) {
	statePath := filepath.Join(fogitDir, mergeStateFile)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No merge in progress
		}
		return nil, fmt.Errorf("failed to read merge state: %w", err)
	}

	var state MergeState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse merge state: %w", err)
	}

	return &state, nil
}

// ClearMergeState removes the merge state file
func ClearMergeState(fogitDir string) error {
	statePath := filepath.Join(fogitDir, mergeStateFile)
	err := os.Remove(statePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear merge state: %w", err)
	}
	return nil
}

// HasMergeState checks if there's an in-progress fogit merge
func HasMergeState(fogitDir string) bool {
	statePath := filepath.Join(fogitDir, mergeStateFile)
	_, err := os.Stat(statePath)
	return err == nil
}

package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/eg3r/fogit/pkg/fogit"
)

// ImportOptions contains options for import operation
type ImportOptions struct {
	Merge     bool // Skip existing features, import only new ones
	Overwrite bool // Replace existing features with imported data
	DryRun    bool // Preview changes without applying them
}

// ImportResult tracks the results of an import operation
type ImportResult struct {
	Created int
	Updated int
	Skipped int
	Errors  []string
	Actions []ImportAction // For dry-run reporting
}

// ImportAction represents a single import action for reporting
type ImportAction struct {
	Type        string // "CREATE", "UPDATE", "SKIP"
	FeatureName string
	FeatureID   string
	Reason      string
}

// Import imports features from export data
func Import(ctx context.Context, repo fogit.Repository, data *ExportData, opts ImportOptions) (*ImportResult, error) {
	// Validate import data
	if err := ValidateImportData(data); err != nil {
		return nil, fmt.Errorf("invalid import data: %w", err)
	}

	// Get existing features for conflict detection
	existingFeatures, err := repo.List(ctx, &fogit.Filter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list existing features: %w", err)
	}

	existingIDs := make(map[string]*fogit.Feature)
	existingNames := make(map[string]*fogit.Feature)
	for _, f := range existingFeatures {
		existingIDs[f.ID] = f
		existingNames[f.Name] = f
	}

	// Build set of all feature IDs (existing + to be imported)
	allFeatureIDs := make(map[string]bool)
	for id := range existingIDs {
		allFeatureIDs[id] = true
	}
	for _, f := range data.Features {
		allFeatureIDs[f.ID] = true
	}

	// Validate relationship targets (warning only)
	ValidateRelationshipTargets(data.Features, allFeatureIDs)

	// Process import
	result := &ImportResult{
		Actions: make([]ImportAction, 0),
	}

	for _, ef := range data.Features {
		existing := existingIDs[ef.ID]
		if existing == nil {
			// Also check by name for potential conflicts
			existing = existingNames[ef.Name]
		}

		if existing != nil {
			// Handle conflict
			if !opts.Merge && !opts.Overwrite {
				result.Errors = append(result.Errors,
					fmt.Sprintf("feature '%s' (ID: %s) already exists", ef.Name, ef.ID))
				continue
			}

			if opts.Merge {
				result.Skipped++
				result.Actions = append(result.Actions, ImportAction{
					Type:        "SKIP",
					FeatureName: ef.Name,
					FeatureID:   ef.ID,
					Reason:      "already exists",
				})
				continue
			}

			// Overwrite - update existing feature
			result.Actions = append(result.Actions, ImportAction{
				Type:        "UPDATE",
				FeatureName: ef.Name,
				FeatureID:   ef.ID,
			})

			if !opts.DryRun {
				feature := ConvertFromExportFeature(ef)
				if err := repo.Update(ctx, feature); err != nil {
					result.Errors = append(result.Errors,
						fmt.Sprintf("failed to update '%s': %v", ef.Name, err))
					continue
				}
			}
			result.Updated++
		} else {
			// New feature
			result.Actions = append(result.Actions, ImportAction{
				Type:        "CREATE",
				FeatureName: ef.Name,
				FeatureID:   ef.ID,
			})

			if !opts.DryRun {
				feature := ConvertFromExportFeature(ef)
				if err := repo.Create(ctx, feature); err != nil {
					result.Errors = append(result.Errors,
						fmt.Sprintf("failed to create '%s': %v", ef.Name, err))
					continue
				}
			}
			result.Created++
		}
	}

	return result, nil
}

// ReadImportFile reads and parses an import file (JSON or YAML)
func ReadImportFile(filePath string) (*ExportData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var importData ExportData

	// Try JSON first
	if err := json.Unmarshal(data, &importData); err == nil {
		return &importData, nil
	}

	// Try YAML
	if err := yaml.Unmarshal(data, &importData); err == nil {
		return &importData, nil
	}

	return nil, fmt.Errorf("failed to parse file as JSON or YAML")
}

// ValidateImportData validates the structure of import data
func ValidateImportData(data *ExportData) error {
	if data.FogitVersion == "" {
		return fmt.Errorf("missing fogit_version field")
	}

	if len(data.Features) == 0 {
		return fmt.Errorf("no features to import")
	}

	for i, f := range data.Features {
		if f.ID == "" {
			return fmt.Errorf("feature at index %d is missing ID", i)
		}
		if f.Name == "" {
			return fmt.Errorf("feature '%s' is missing name", f.ID)
		}
	}

	return nil
}

// ValidateRelationshipTargets checks if all relationship targets exist
// Returns warnings but doesn't fail the import
func ValidateRelationshipTargets(features []*ExportFeature, allIDs map[string]bool) []string {
	var warnings []string

	for _, f := range features {
		for _, r := range f.Relationships {
			if !allIDs[r.TargetID] {
				warnings = append(warnings,
					fmt.Sprintf("feature '%s' has relationship to unknown target '%s' (ID: %s)",
						f.Name, r.TargetName, r.TargetID))
			}
		}
	}

	return warnings
}

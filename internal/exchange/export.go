package exchange

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/eg3r/fogit/pkg/fogit"
)

// ExportOptions contains options for export operation
type ExportOptions struct {
	Format   string // "json", "yaml", or "csv"
	Filter   *fogit.Filter
	FogitDir string
	Pretty   bool
}

// Export exports features to the specified format
func Export(ctx context.Context, repo fogit.Repository, opts ExportOptions) (*ExportData, error) {
	// Get features
	features, err := repo.List(ctx, opts.Filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	return ExportWithFeatures(features, opts)
}

// ExportWithFeatures exports pre-loaded features to the specified format.
// This is useful for cross-branch export where features come from multiple branches.
func ExportWithFeatures(features []*fogit.Feature, opts ExportOptions) (*ExportData, error) {
	// Build feature ID set for target existence check
	featureIDs := make(map[string]bool)
	for _, f := range features {
		featureIDs[f.ID] = true
	}

	// Get repository name from directory
	repoName := filepath.Base(filepath.Dir(opts.FogitDir))

	// Build export data
	exportData := &ExportData{
		FogitVersion: "1.0",
		ExportedAt:   time.Now().UTC().Format(time.RFC3339),
		Repository:   repoName,
		Features:     make([]*ExportFeature, 0, len(features)),
	}

	for _, f := range features {
		exportFeature := ConvertToExportFeature(f, featureIDs)
		exportData.Features = append(exportData.Features, exportFeature)
	}

	return exportData, nil
}

// WriteJSON writes export data as JSON
func WriteJSON(w io.Writer, data *ExportData, pretty bool) error {
	encoder := json.NewEncoder(w)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

// WriteYAML writes export data as YAML
func WriteYAML(w io.Writer, data *ExportData) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(data)
}

// WriteCSV writes export data as CSV
func WriteCSV(w io.Writer, features []*ExportFeature) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{
		"ID", "Name", "Description", "State", "CurrentVersion",
		"Tags", "Type", "Priority", "Category", "Domain", "Team",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, f := range features {
		// Extract metadata fields with defaults
		fType := ""
		priority := ""
		category := ""
		domain := ""
		team := ""
		if f.Metadata != nil {
			if v, ok := f.Metadata["type"].(string); ok {
				fType = v
			}
			if v, ok := f.Metadata["priority"].(string); ok {
				priority = v
			}
			if v, ok := f.Metadata["category"].(string); ok {
				category = v
			}
			if v, ok := f.Metadata["domain"].(string); ok {
				domain = v
			}
			if v, ok := f.Metadata["team"].(string); ok {
				team = v
			}
		}

		tags := ""
		if len(f.Tags) > 0 {
			tags = fmt.Sprintf("%v", f.Tags)
		}

		row := []string{
			f.ID,
			f.Name,
			f.Description,
			f.State,
			f.CurrentVersion,
			tags,
			fType,
			priority,
			category,
			domain,
			team,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/common"
	"github.com/eg3r/fogit/internal/config"
	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	relExportOutput         string
	relExportTypesOnly      bool
	relExportCategoriesOnly bool
)

// RelationshipExport represents the export schema for relationship definitions
type RelationshipExport struct {
	FogitVersion           string                                `json:"fogit_version" yaml:"fogit_version"`
	ExportedAt             string                                `json:"exported_at" yaml:"exported_at"`
	Repository             string                                `json:"repository,omitempty" yaml:"repository,omitempty"`
	RelationshipCategories map[string]RelationshipCategoryExport `json:"relationship_categories,omitempty" yaml:"relationship_categories,omitempty"`
	RelationshipTypes      map[string]RelationshipTypeExport     `json:"relationship_types,omitempty" yaml:"relationship_types,omitempty"`
}

// RelationshipCategoryExport represents a category in the export format
type RelationshipCategoryExport struct {
	Description     string `json:"description,omitempty" yaml:"description,omitempty"`
	AllowCycles     bool   `json:"allow_cycles" yaml:"allow_cycles"`
	CycleDetection  string `json:"cycle_detection" yaml:"cycle_detection"`
	IncludeInImpact bool   `json:"include_in_impact" yaml:"include_in_impact"`
}

// RelationshipTypeExport represents a type in the export format
type RelationshipTypeExport struct {
	Category      string   `json:"category" yaml:"category"`
	Description   string   `json:"description,omitempty" yaml:"description,omitempty"`
	Inverse       string   `json:"inverse,omitempty" yaml:"inverse,omitempty"`
	Bidirectional bool     `json:"bidirectional,omitempty" yaml:"bidirectional,omitempty"`
	Aliases       []string `json:"aliases,omitempty" yaml:"aliases,omitempty"`
}

var relationshipExportCmd = &cobra.Command{
	Use:   "export [format]",
	Short: "Export relationship type and category definitions",
	Long: `Export relationship type and category definitions to JSON or YAML.

This enables sharing configurations across repositories and teams.

Examples:
  # Export all definitions to stdout (JSON format)
  fogit relationship export

  # Export in YAML format
  fogit relationship export yaml

  # Export to a file
  fogit relationship export yaml --output team-standards.yaml

  # Export only relationship types
  fogit relationship export json --types-only > types.json

  # Export only categories
  fogit relationship export yaml --categories-only --output categories.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRelationshipExport,
}

func init() {
	relationshipExportCmd.Flags().StringVarP(&relExportOutput, "output", "o", "", "Output file (default: stdout)")
	relationshipExportCmd.Flags().BoolVar(&relExportTypesOnly, "types-only", false, "Export only relationship types")
	relationshipExportCmd.Flags().BoolVar(&relExportCategoriesOnly, "categories-only", false, "Export only relationship categories")

	// Add as subcommand to relationship
	relationshipCmd.AddCommand(relationshipExportCmd)
}

func runRelationshipExport(cmd *cobra.Command, args []string) error {
	// Determine format
	format := "json"
	if len(args) > 0 {
		format = args[0]
	}

	if format != "json" && format != "yaml" {
		return fmt.Errorf("invalid format '%s': must be json or yaml", format)
	}

	// Validate flags
	if relExportTypesOnly && relExportCategoriesOnly {
		return fmt.Errorf("cannot specify both --types-only and --categories-only")
	}

	fogitDir, err := getFogitDir()
	if err != nil {
		return fmt.Errorf("failed to get .fogit directory: %w", err)
	}

	// Load config
	cfg, err := config.Load(fogitDir)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Build export structure
	export := RelationshipExport{
		FogitVersion: "1.0",
		ExportedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	// Try to get repository name
	if repoName := getRepositoryName(fogitDir); repoName != "" {
		export.Repository = repoName
	}

	// Add categories if not types-only
	if !relExportTypesOnly {
		export.RelationshipCategories = convertCategoriesToExport(cfg.Relationships.Categories)
	}

	// Add types if not categories-only
	if !relExportCategoriesOnly {
		export.RelationshipTypes = convertTypesToExport(cfg.Relationships.Types)
	}

	// Write to file or stdout
	if relExportOutput != "" {
		// Validate output path to prevent path traversal attacks
		if err := common.ValidateOutputPath(relExportOutput); err != nil {
			return err
		}

		// Use atomic write to prevent partial/corrupted files on failure
		if err := common.AtomicWriteFile(relExportOutput, func(f *os.File) error {
			return printer.OutputFormatted(f, format, export, nil)
		}); err != nil {
			return fmt.Errorf("failed to write export: %w", err)
		}

		fmt.Printf("Exported relationship definitions to %s\n", relExportOutput)
		return nil
	}

	// Write to stdout (no atomic write needed)
	if err := printer.OutputFormatted(os.Stdout, format, export, nil); err != nil {
		return fmt.Errorf("failed to write export: %w", err)
	}

	return nil
}

// convertCategoriesToExport converts config categories to export format
func convertCategoriesToExport(categories map[string]fogit.RelationshipCategory) map[string]RelationshipCategoryExport {
	if len(categories) == 0 {
		return nil
	}

	result := make(map[string]RelationshipCategoryExport)
	for name, cat := range categories {
		result[name] = RelationshipCategoryExport{
			Description:     cat.Description,
			AllowCycles:     cat.AllowCycles,
			CycleDetection:  cat.CycleDetection,
			IncludeInImpact: cat.IncludeInImpact,
		}
	}
	return result
}

// convertTypesToExport converts config types to export format
func convertTypesToExport(types map[string]fogit.RelationshipTypeConfig) map[string]RelationshipTypeExport {
	if len(types) == 0 {
		return nil
	}

	result := make(map[string]RelationshipTypeExport)
	for name, typ := range types {
		export := RelationshipTypeExport{
			Category:    typ.Category,
			Description: typ.Description,
			Inverse:     typ.Inverse,
		}
		// Only include bidirectional if true
		if typ.Bidirectional {
			export.Bidirectional = true
		}
		// Only include aliases if non-empty
		if len(typ.Aliases) > 0 {
			export.Aliases = typ.Aliases
		}
		result[name] = export
	}
	return result
}

// getRepositoryName tries to get the repository name from Git or config
func getRepositoryName(fogitDir string) string {
	// Get the parent directory (the actual git repo)
	repoPath := filepath.Dir(fogitDir)

	// Try to get from Git remote
	gitRepo, err := git.OpenRepository(repoPath)
	if err == nil {
		if name := gitRepo.GetRepositoryName(); name != "" {
			return name
		}
	}

	// Fallback: try config
	cfg, err := config.Load(fogitDir)
	if err == nil && cfg.Repository.Name != "" {
		return cfg.Repository.Name
	}

	return ""
}

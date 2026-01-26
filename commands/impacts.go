package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/printer"
)

var impactsCmd = &cobra.Command{
	Use:   "impacts <feature>",
	Short: "Find features impacted by changes to a feature",
	Long: `Analyze which features would be affected by changes to a specified feature.

By default, only relationships in categories with 'include_in_impact: true' are included.
Use --all-categories to include all relationship types.

Examples:
  fogit impacts "User Authentication"
  fogit impacts "API Core" --depth 3
  fogit impacts "Payment Service" --all-categories
  fogit impacts "Database Schema" --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runImpacts,
}

var (
	impactsDepth           int
	impactsFormat          string
	impactsIncludeCategory []string
	impactsExcludeCategory []string
	impactsAllCategories   bool
)

func init() {
	impactsCmd.Flags().IntVar(&impactsDepth, "depth", 0, "Maximum relationship depth to traverse (0 = unlimited)")
	impactsCmd.Flags().StringVar(&impactsFormat, "format", "text", "Output format: text, json, yaml, tree")
	impactsCmd.Flags().StringSliceVar(&impactsIncludeCategory, "include-category", nil, "Include specific category")
	impactsCmd.Flags().StringSliceVar(&impactsExcludeCategory, "exclude-category", nil, "Exclude specific category")
	impactsCmd.Flags().BoolVar(&impactsAllCategories, "all-categories", false, "Include all categories")
	rootCmd.AddCommand(impactsCmd)
}

func runImpacts(cmd *cobra.Command, args []string) error {
	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	cfg := cmdCtx.Config

	// Find the feature using cross-branch discovery
	feature, err := FindFeatureCrossBranch(cmd.Context(), cmdCtx, args[0], "fogit impacts <id>")
	if err != nil {
		return err
	}

	// Build impact options from flags
	opts := features.ImpactOptions{
		MaxDepth:          impactsDepth,
		IncludeCategories: impactsIncludeCategory,
		ExcludeCategories: impactsExcludeCategory,
		AllCategories:     impactsAllCategories,
	}

	// Determine which categories to include using the service
	includedCategories := features.GetIncludedCategories(cfg, opts)

	// Get all features using cross-branch discovery for impact analysis
	allFeatures, err := ListFeaturesCrossBranch(cmd.Context(), cmdCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	// Build impact analysis using the service with cross-branch features
	result, err := features.AnalyzeImpactsWithFeatures(cmd.Context(), feature, allFeatures, cfg, includedCategories, impactsDepth)
	if err != nil {
		return fmt.Errorf("failed to analyze impacts: %w", err)
	}

	// Format and output
	return outputImpactResult(result, impactsFormat)
}

func outputImpactResult(result *features.ImpactResult, format string) error {
	textFn := func(w io.Writer) error {
		if format == "tree" {
			fmt.Fprintf(w, "Features impacted by \"%s\":\n\n", result.Feature)
			printImpactTree(result.ImpactedFeatures)
			fmt.Fprintf(w, "\nTotal: %d features affected\n", result.TotalAffected)
			return nil
		}

		// Default text format
		fmt.Fprintf(w, "Features impacted by \"%s\":\n\n", result.Feature)
		if len(result.ImpactedFeatures) == 0 {
			fmt.Fprintln(w, "No features are impacted.")
		} else {
			for _, f := range result.ImpactedFeatures {
				fmt.Fprintf(w, "  %s (%s) [depth: %d]\n", f.Name, f.Relationship, f.Depth)
				fmt.Fprintf(w, "    Path: %s\n", strings.Join(f.Path, " → "))
				if f.Warning != "" {
					fmt.Fprintf(w, "    WARNING: %s\n", f.Warning)
				}
			}
		}
		fmt.Fprintf(w, "\nTotal: %d features affected\n", result.TotalAffected)
		fmt.Fprintf(w, "Categories included: %s\n", strings.Join(result.CategoriesIncluded, ", "))
		return nil
	}

	return printer.OutputFormatted(os.Stdout, format, result, textFn)
}

func printImpactTree(impacts []features.ImpactedFeature) {
	// Group by depth for tree display
	byDepth := make(map[int][]features.ImpactedFeature)
	for _, imp := range impacts {
		byDepth[imp.Depth] = append(byDepth[imp.Depth], imp)
	}

	// Simple tree output
	for depth := 1; depth <= len(byDepth); depth++ {
		for i, imp := range byDepth[depth] {
			prefix := strings.Repeat("│  ", depth-1)
			connector := "├─"
			if i == len(byDepth[depth])-1 {
				connector = "└─"
			}
			warningMarker := ""
			if imp.Warning != "" {
				warningMarker = " [!]"
			}
			fmt.Printf("%s%s %s (%s)%s\n", prefix, connector, imp.Name, imp.Relationship, warningMarker)
			if imp.Warning != "" {
				warnPrefix := strings.Repeat("│  ", depth)
				fmt.Printf("%s   WARNING: %s\n", warnPrefix, imp.Warning)
			}
		}
	}
}

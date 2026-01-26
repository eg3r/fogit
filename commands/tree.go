package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	treeDepth    int
	treeCategory string
	treeState    string
	treeFormat   string
	treeTypes    []string // Changed from treeType to support multiple types
)

var treeCmd = &cobra.Command{
	Use:   "tree [feature]",
	Short: "Show hierarchical tree view of features",
	Long: `Show hierarchical tree view of features based on their relationships.

If a feature is specified, shows that feature as the root.
Otherwise, shows all top-level features (those without parents).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTree,
}

func init() {
	treeCmd.Flags().IntVar(&treeDepth, "depth", -1, "Maximum depth to show (-1 for unlimited)")
	treeCmd.Flags().StringVar(&treeCategory, "category", "", "Filter by category")
	treeCmd.Flags().StringVar(&treeState, "state", "", "Filter by state")
	treeCmd.Flags().StringVar(&treeFormat, "format", "tree", "Output format: tree, json")
	treeCmd.Flags().StringSliceVar(&treeTypes, "type", []string{}, "Relationship types for hierarchy (repeatable, default from config)")
	rootCmd.AddCommand(treeCmd)
}

func runTree(cmd *cobra.Command, args []string) error {
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return fmt.Errorf("failed to initialize: %w", err)
	}

	ctx := cmd.Context()
	cfg := cmdCtx.Config

	// Determine which relationship types to use for tree hierarchy using the service
	hierarchyTypes, err := features.DetermineTreeRelationshipTypes(cfg, treeTypes)
	if err != nil {
		return err
	}

	// Build filter
	filter := &fogit.Filter{}
	if treeCategory != "" {
		filter.Category = treeCategory
	}
	if treeState != "" {
		filter.State = fogit.State(treeState)
	}

	// Get all features using cross-branch discovery
	allFeatures, err := ListFeaturesCrossBranch(ctx, cmdCtx, filter)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	if len(allFeatures) == 0 {
		fmt.Println("No features found")
		return nil
	}

	// If specific feature requested, start from there
	if len(args) > 0 {
		feature, err := FindFeatureCrossBranch(ctx, cmdCtx, args[0], "fogit tree <id>")
		if err != nil {
			return err
		}
		return printer.OutputTree(os.Stdout, feature, allFeatures, hierarchyTypes, treeDepth)
	}

	// Otherwise, find all root features (no relationships of hierarchy types pointing outward)
	roots := features.FindRoots(allFeatures, hierarchyTypes)

	if len(roots) == 0 {
		typeList := strings.Join(hierarchyTypes, ", ")
		fmt.Printf("No root features found (all features have '%s' relationships)\n", typeList)
		return nil
	}

	// Display each root
	for i, root := range roots {
		if i > 0 {
			fmt.Println()
		}
		if err := printer.OutputTree(os.Stdout, root, allFeatures, hierarchyTypes, treeDepth); err != nil {
			return err
		}
	}

	return nil
}

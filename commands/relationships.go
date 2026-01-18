package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/printer"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	relDirection string
	relTypes     []string
	relFormat    string
	relRecursive bool
	relDepth     int
)

var relationshipsCmd = &cobra.Command{
	Use:     "relationships <feature>",
	Aliases: []string{"links", "rels"},
	Short:   "Show relationships for a feature",
	Long: `Show relationships for a feature.

By default, shows both incoming and outgoing relationships.`,
	Args: cobra.ExactArgs(1),
	RunE: runRelationships,
}

func init() {
	relationshipsCmd.Flags().StringVar(&relDirection, "direction", "both", "Show incoming, outgoing, or both relationships")
	relationshipsCmd.Flags().StringSliceVar(&relTypes, "type", nil, "Filter by relationship type (can be repeated)")
	relationshipsCmd.Flags().StringVar(&relFormat, "format", "text", "Output format: text, json")
	relationshipsCmd.Flags().BoolVar(&relRecursive, "recursive", false, "Follow relationships recursively")
	relationshipsCmd.Flags().IntVar(&relDepth, "depth", 0, "Maximum depth when using --recursive (0 = unlimited)")
	rootCmd.AddCommand(relationshipsCmd)
}

func runRelationships(cmd *cobra.Command, args []string) error {
	identifier := args[0]

	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	// Find feature using the consolidated helper
	feature, err := FindFeatureWithSuggestions(ctx, cmdCtx.Repo, identifier, cmdCtx.Config, "fogit relationships <id>")
	if err != nil {
		return err
	}

	// If recursive mode, use the traversal service
	if relRecursive {
		return runRecursiveRelationships(cmdCtx.Repo, ctx, feature)
	}

	// Non-recursive mode: direct relationships only
	var outgoing []fogit.Relationship
	if relDirection == "outgoing" || relDirection == "both" {
		outgoing = filterRelationshipsByTypesLocal(feature.GetRelationships(""), relTypes)
	}

	// Get incoming relationships (by scanning all features)
	var incoming []features.RelationshipWithSource
	if relDirection == "incoming" || relDirection == "both" {
		incoming, err = features.FindIncomingRelationshipsMultiType(cmdCtx.Repo, ctx, feature.ID, relTypes)
		if err != nil {
			return fmt.Errorf("failed to find incoming relationships: %w", err)
		}
	}

	// Output results
	if relFormat == "text" {
		err = printer.OutputRelationshipsText(os.Stdout, feature, outgoing, incoming)
		if err != nil {
			return err
		}
	} else if relFormat == "json" {
		err = printer.OutputRelationshipsJSON(os.Stdout, feature, outgoing, incoming)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid format: %s (valid: text, json)", relFormat)
	}

	return nil
}

// filterRelationshipsByTypesLocal filters relationships by type list (empty = all)
func filterRelationshipsByTypesLocal(rels []fogit.Relationship, types []string) []fogit.Relationship {
	if len(types) == 0 {
		return rels
	}
	var filtered []fogit.Relationship
	for _, rel := range rels {
		for _, t := range types {
			if string(rel.Type) == t {
				filtered = append(filtered, rel)
				break
			}
		}
	}
	return filtered
}

func runRecursiveRelationships(repo fogit.Repository, ctx context.Context, feature *fogit.Feature) error {
	// Use the traversal service
	opts := features.TraversalOptions{
		Direction: relDirection,
		Types:     relTypes,
		MaxDepth:  relDepth,
	}

	result, err := features.TraverseRelationshipsRecursive(ctx, repo, feature, opts)
	if err != nil {
		return fmt.Errorf("failed to traverse relationships: %w", err)
	}

	// Output results
	return outputRecursiveRelationships(result, relFormat)
}

func outputRecursiveRelationships(result *features.TraversalResult, format string) error {
	output := struct {
		Feature       string                           `json:"feature" yaml:"feature"`
		FeatureID     string                           `json:"feature_id" yaml:"feature_id"`
		Recursive     bool                             `json:"recursive" yaml:"recursive"`
		Depth         int                              `json:"max_depth" yaml:"max_depth"`
		Direction     string                           `json:"direction" yaml:"direction"`
		Types         []string                         `json:"types,omitempty" yaml:"types,omitempty"`
		Relationships []features.RecursiveRelationship `json:"relationships" yaml:"relationships"`
		Total         int                              `json:"total" yaml:"total"`
	}{
		Feature:       result.Feature.Name,
		FeatureID:     result.Feature.ID,
		Recursive:     true,
		Depth:         result.MaxDepth,
		Direction:     result.Direction,
		Types:         result.Types,
		Relationships: result.Relationships,
		Total:         result.Total,
	}

	textFn := func(w io.Writer) error {
		fmt.Fprintf(w, "Recursive relationships for: %s\n", result.Feature.Name)
		fmt.Fprintf(w, "Direction: %s, Max Depth: %d\n\n", result.Direction, result.MaxDepth)

		if len(result.Relationships) == 0 {
			fmt.Fprintln(w, "No relationships found")
			return nil
		}

		// Group by depth
		maxDepth := 0
		for _, r := range result.Relationships {
			if r.Depth > maxDepth {
				maxDepth = r.Depth
			}
		}

		for d := 1; d <= maxDepth; d++ {
			fmt.Fprintf(w, "Depth %d:\n", d)
			for _, r := range result.Relationships {
				if r.Depth != d {
					continue
				}
				indent := strings.Repeat("  ", d)
				if result.Direction == "incoming" {
					fmt.Fprintf(w, "%s[%s] %s <- %s\n", indent, r.Type, r.TargetName, r.SourceName)
				} else {
					fmt.Fprintf(w, "%s[%s] %s -> %s\n", indent, r.Type, r.SourceName, r.TargetName)
				}
			}
		}
		fmt.Fprintf(w, "\nTotal: %d relationships\n", result.Total)
		return nil
	}

	return printer.OutputFormatted(os.Stdout, format, output, textFn)
}

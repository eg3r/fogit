package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/pkg/fogit"
)

var (
	linkDescription       string
	linkVersionConstraint string
)

var linkCmd = &cobra.Command{
	Use:   "link <source> <target> <type>",
	Short: "Create a relationship between features",
	Long: `Create a relationship between two features.

Common relationship types:
  depends-on    - Source depends on target
  blocks        - Source blocks target
  contains      - Source contains target
  relates-to    - General relationship
  implements    - Source implements target
  tests         - Source tests target`,
	Args: cobra.ExactArgs(3),
	RunE: runLink,
}

func init() {
	linkCmd.Flags().StringVarP(&linkDescription, "description", "d", "", "Human-readable description of the relationship")
	linkCmd.Flags().StringVar(&linkVersionConstraint, "version-constraint", "", "Version requirement (e.g., \">=2\")")
	rootCmd.AddCommand(linkCmd)
}

func runLink(cmd *cobra.Command, args []string) error {
	sourceIdentifier := args[0]
	targetIdentifier := args[1]
	relType := fogit.RelationshipType(args[2])

	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	ctx := cmd.Context()
	cfg := cmdCtx.Config

	// Per spec/specification/07-git-integration.md#cross-branch-feature-discovery:
	// In branch-per-feature mode, cross-branch discovery is required for
	// "Creating relationships between features on different branches"
	// FindFeatureCrossBranchWithResult handles mode check internally

	// Find source feature (cross-branch if in branch-per-feature mode)
	sourceResult, err := FindFeatureCrossBranchWithResult(ctx, cmdCtx, sourceIdentifier, "fogit link <id> ...")
	if err != nil {
		return fmt.Errorf("source feature not found: %w", err)
	}
	source := sourceResult.Feature
	sourceBranch := sourceResult.Branch

	// Find target feature (cross-branch if in branch-per-feature mode)
	targetResult, err := FindFeatureCrossBranchWithResult(ctx, cmdCtx, targetIdentifier, "fogit link ... <id> ...")
	if err != nil {
		return fmt.Errorf("target feature not found: %w", err)
	}
	target := targetResult.Feature
	targetBranch := targetResult.Branch

	// Create relationship object for validation and cycle detection
	rel := &fogit.Relationship{
		Type:     relType,
		TargetID: target.ID,
	}

	// Validate relationship against config
	if validateErr := rel.ValidateWithConfig(cfg); validateErr != nil {
		return fmt.Errorf("invalid relationship: %w", validateErr)
	}

	// Check for cycles based on category settings
	if cycleErr := fogit.DetectCycleWithConfig(ctx, source, rel, cmdCtx.Repo, cfg); cycleErr != nil {
		return fmt.Errorf("cannot create relationship: %w", cycleErr)
	}

	// Link features with cross-branch support
	var linkOpts *features.LinkOptions
	if cfg.Workflow.Mode == "branch-per-feature" && cmdCtx.Git != nil && cmdCtx.Git.GetGitRepo() != nil {
		linkOpts = &features.LinkOptions{
			GitRepo:      cmdCtx.Git.GetGitRepo(),
			SourceBranch: sourceBranch,
			TargetBranch: targetBranch,
		}
	}

	newRel, err := features.LinkWithOptions(ctx, cmdCtx.Repo, source, target, relType, linkDescription, linkVersionConstraint, cfg, cmdCtx.FogitDir, linkOpts)
	if err != nil {
		if err == fogit.ErrDuplicateRelationship {
			return fmt.Errorf("relationship already exists: %s -> %s (%s)", source.Name, target.Name, relType)
		}
		return err
	}

	fmt.Printf("Created relationship: %s -> %s (%s)\n", source.Name, target.Name, relType)
	fmt.Printf("  ID: %s\n", newRel.ID)
	if newRel.Description != "" {
		fmt.Printf("  Description: %s\n", newRel.Description)
	}
	if newRel.VersionConstraint != nil {
		fmt.Printf("  Version Constraint: %s%s\n", newRel.VersionConstraint.Operator, newRel.VersionConstraint.GetVersionString())
		if newRel.VersionConstraint.Note != "" {
			fmt.Printf("    Note: %s\n", newRel.VersionConstraint.Note)
		}
	}

	return nil
}

package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/common"
	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/internal/interactive"
	"github.com/eg3r/fogit/internal/logger"
	"github.com/eg3r/fogit/internal/search"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

var featureCmd = &cobra.Command{
	Use:   "feature <name>",
	Short: "Create or switch to a feature",
	Long: `Create a new feature or switch to an existing feature.

If the feature doesn't exist, it will be created.
If it exists, this command will switch to it (future: with Git integration).

Examples:
  # Create a new feature
  fogit feature "User Authentication"

  # Create with description and type
  fogit feature "Login Page" -d "User login form" --type ui-component

  # Create with priority and tags
  fogit feature "API Rate Limiting" -p high --tags security,performance

  # Create with organization metadata
  fogit feature "Payment Integration" --category billing --team payments --epic checkout

  # Create on isolated branch (new Git branch)
  fogit feature "Experimental Feature" --isolate

  # Create on same branch (shared strategy)
  fogit feature "Quick Fix" --same

  # Reopen a closed feature with new version
  fogit feature "User Auth" --minor
  fogit feature "User Auth" --version 2.0.0`,
	Args: cobra.ExactArgs(1),
	RunE: runFeature,
}

var featureCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new feature (explicit)",
	Long: `Create a new feature. Fails if feature already exists.

Examples:
  # Create a new feature explicitly
  fogit feature create "New Feature"

  # Create with all metadata
  fogit feature create "Dashboard" -d "Main dashboard" --type ui-component -p high`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureCreate,
}

var (
	featureDescription string
	featureType        string
	featurePriority    string
	featureCategory    string
	featureDomain      string
	featureTeam        string
	featureEpic        string
	featureModule      string
	featureTags        []string
	featureMetadata    []string // key=value pairs
	featureParent      string
	featureSame        bool   // Stay on current branch (shared strategy)
	featureIsolate     bool   // Create new branch (isolated strategy)
	featureVersion     string // Explicit version number (e.g., "5" or "2.0.0")
	featurePatch       bool   // Increment patch version (semantic only)
	featureMinor       bool   // Increment minor version
	featureMajor       bool   // Increment major version
	featureNewVersion  bool   // Auto-increment to next version (skip prompt)
)

// registerFeatureFlags registers common feature flags on a command.
// This consolidates the duplicate flag registration between featureCmd and featureCreateCmd.
func registerFeatureFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&featureDescription, "description", "d", "", "Feature description")
	cmd.Flags().StringVar(&featureType, "type", "", "Feature type (free-form, e.g., software-feature, api-endpoint)")
	cmd.Flags().StringVarP(&featurePriority, "priority", "p", "medium", "Priority (low, medium, high, critical)")
	cmd.Flags().StringVar(&featureCategory, "category", "", "Business category")
	cmd.Flags().StringVar(&featureDomain, "domain", "", "Technical domain")
	cmd.Flags().StringVar(&featureTeam, "team", "", "Owning team")
	cmd.Flags().StringVar(&featureEpic, "epic", "", "Epic grouping")
	cmd.Flags().StringVar(&featureModule, "module", "", "Code module")
	cmd.Flags().StringSliceVar(&featureTags, "tags", []string{}, "Comma-separated tags")
	cmd.Flags().StringSliceVar(&featureMetadata, "metadata", []string{}, "Custom metadata (key=value, repeatable)")
	cmd.Flags().StringVar(&featureParent, "parent", "", "Parent feature ID or name")
	cmd.Flags().BoolVar(&featureSame, "same", false, "Stay on current branch (shared strategy, requires allow_shared_branches: true)")
	cmd.Flags().BoolVar(&featureIsolate, "isolate", false, "Create new branch (isolated strategy, overrides default)")
}

func init() {
	// Register common flags for both commands
	registerFeatureFlags(featureCmd)
	registerFeatureFlags(featureCreateCmd)

	// Version flags only for main feature command (for reopening closed features)
	featureCmd.Flags().StringVar(&featureVersion, "version", "", "Explicit version number (e.g., \"5\" or \"2.0.0\")")
	featureCmd.Flags().BoolVar(&featurePatch, "patch", false, "Increment patch version (semantic: v1.0.1)")
	featureCmd.Flags().BoolVar(&featureMinor, "minor", false, "Increment minor version (semantic: v1.1.0, simple: v2)")
	featureCmd.Flags().BoolVar(&featureMajor, "major", false, "Increment major version (semantic: v2.0.0, simple: v2)")
	featureCmd.Flags().BoolVar(&featureNewVersion, "new-version", false, "Auto-increment to next version (skips prompt)")

	// Add subcommands
	featureCmd.AddCommand(featureCreateCmd)
	rootCmd.AddCommand(featureCmd)
}

func runFeature(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	repo := cmdCtx.Repo
	cfg := cmdCtx.Config

	// Get all features for fuzzy search
	allFeatures, err := repo.List(cmd.Context(), nil)
	if err != nil {
		return fmt.Errorf("failed to list features: %w", err)
	}

	// Check for exact match first
	for _, f := range allFeatures {
		if strings.EqualFold(f.Name, name) {
			// Feature exists
			state := f.DeriveState()
			if state == fogit.StateClosed {
				// Feature is closed - handle versioning per spec
				return handleClosedFeature(cmd, f, cmdCtx)
			}

			// Feature is open or in-progress - switch to it (future implementation)
			fmt.Printf("Feature exists: %s\n", f.ID)
			fmt.Printf("  Name: %s\n", f.Name)
			if fType := f.GetType(); fType != "" {
				fmt.Printf("  Type: %s\n", fType)
			}
			fmt.Printf("  State: %s\n", state)
			fmt.Printf("\nNote: Switching to features not yet implemented.\n")
			return nil
		}
	}

	// No exact match - check for similar features using fuzzy search
	if cfg.FeatureSearch.FuzzyMatch {
		searchCfg := search.SearchConfig{
			FuzzyMatch:     cfg.FeatureSearch.FuzzyMatch,
			MinSimilarity:  cfg.FeatureSearch.MinSimilarity,
			MaxSuggestions: cfg.FeatureSearch.MaxSuggestions,
		}

		matches := search.FindSimilar(name, allFeatures, searchCfg)
		if len(matches) > 0 {
			// Use interactive prompter for feature selection
			prompter := interactive.NewPrompter()
			selectedFeature, createNew, err := prompter.SelectFromSimilarFeatures(name, matches)
			if err != nil {
				return fmt.Errorf("failed to get selection: %w", err)
			}

			if !createNew && selectedFeature != nil {
				fmt.Printf("\nSelected feature: %s\n", selectedFeature.Name)
				fmt.Printf("  ID: %s\n", selectedFeature.ID)
				fmt.Printf("  State: %s\n", selectedFeature.DeriveState())
				fmt.Printf("\nNote: Switching to features not yet implemented.\n")
				return nil
			}
			// User chose to create new - continue below
			fmt.Println()
		}
	}

	// Feature doesn't exist - create it
	return createFeature(cmd, name, cmdCtx)
}

// handleClosedFeature handles versioning when working with a closed feature
// Per spec 08-interface.md: prompts for version type or uses flags
func handleClosedFeature(cmd *cobra.Command, feature *fogit.Feature, cmdCtx *CommandContext) error {
	cfg := cmdCtx.Config
	repo := cmdCtx.Repo

	currentVersionStr := feature.GetCurrentVersionKey()
	fmt.Printf("Feature \"%s\" is closed (version %s)\n", feature.Name, currentVersionStr)

	// Check for version flags
	versionFlags := 0
	if featureVersion != "" {
		versionFlags++
	}
	if featurePatch {
		versionFlags++
	}
	if featureMinor {
		versionFlags++
	}
	if featureMajor {
		versionFlags++
	}
	if featureNewVersion {
		versionFlags++
	}

	if versionFlags > 1 {
		return fmt.Errorf("cannot specify multiple version flags (--version, --patch, --minor, --major, --new-version)")
	}

	var increment fogit.VersionIncrement
	var useExplicitVersion bool
	var explicitVersion string

	if featureVersion != "" {
		// Explicit version specified
		useExplicitVersion = true
		explicitVersion = featureVersion
	} else if featurePatch {
		increment = fogit.VersionIncrementPatch
	} else if featureMinor {
		increment = fogit.VersionIncrementMinor
	} else if featureMajor {
		increment = fogit.VersionIncrementMajor
	} else if featureNewVersion {
		// Default to minor for auto-increment
		increment = fogit.VersionIncrementMinor
	} else {
		// No flags - prompt user per spec using interactive package
		prompter := interactive.NewPrompter()
		var err error
		increment, err = prompter.SelectVersionIncrement(currentVersionStr, cfg.Workflow.VersionFormat)
		if err != nil {
			return fmt.Errorf("canceled")
		}
	}

	// Calculate new version
	var newVersion string
	var err error

	if useExplicitVersion {
		newVersion = explicitVersion
	} else {
		newVersion, err = fogit.IncrementVersion(currentVersionStr, cfg.Workflow.VersionFormat, increment)
		if err != nil {
			return fmt.Errorf("failed to calculate new version: %w", err)
		}
	}
	// Reopen feature
	branch, err := features.Reopen(cmd.Context(), repo, feature, newVersion, currentVersionStr)
	if err != nil {
		return err
	}

	// Handle branch creation
	if err := features.HandleBranchCreation(feature.Name, cfg, featureSame, featureIsolate); err != nil {
		logger.Warn("branch creation issue", "error", err)
	}

	fmt.Printf("\n✓ Created version %s of \"%s\"\n", newVersion, feature.Name)
	fmt.Printf("  State: %s\n", feature.DeriveState())
	fmt.Printf("  Branch: %s\n", branch)

	return nil
}

func runFeatureCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	return createFeature(cmd, name, cmdCtx)
}

func createFeature(cmd *cobra.Command, name string, cmdCtx *CommandContext) error {
	// Prepare options
	opts := features.CreateOptions{
		Name:          name,
		Description:   featureDescription,
		Type:          featureType,
		Priority:      featurePriority,
		Category:      featureCategory,
		Domain:        featureDomain,
		Team:          featureTeam,
		Epic:          featureEpic,
		Module:        featureModule,
		Tags:          featureTags,
		ParentID:      featureParent,
		SameBranch:    featureSame,
		IsolateBranch: featureIsolate,
	}

	// Parse metadata key=value pairs
	if len(featureMetadata) > 0 {
		opts.Metadata = make(map[string]interface{})
		for _, pair := range featureMetadata {
			key, value := common.SplitKeyValueEquals(pair)
			if value != "" {
				opts.Metadata[key] = value
			}
		}
	}

	repo := cmdCtx.Repo
	cfg := cmdCtx.Config
	fogitDir := cmdCtx.FogitDir

	feature, err := features.Create(cmd.Context(), repo, opts, cfg, fogitDir)
	if err != nil {
		return err
	}

	// Output success
	fmt.Printf("Created feature: %s\n", feature.ID)
	fmt.Printf("  Name: %s\n", feature.Name)
	if feature.Description != "" {
		fmt.Printf("  Description: %s\n", feature.Description)
	}
	if fType := feature.GetType(); fType != "" {
		fmt.Printf("  Type: %s\n", fType)
	}
	fmt.Printf("  State: %s\n", feature.DeriveState())
	if priority := feature.GetPriority(); priority != "" {
		fmt.Printf("  Priority: %s\n", priority)
	}
	if len(feature.Tags) > 0 {
		fmt.Printf("  Tags: %v\n", feature.Tags)
	}

	// Calculate the filename that was used
	existingFiles := make(map[string]bool)
	filename := storage.GenerateFeatureFilename(feature.Name, feature.ID, existingFiles)
	fmt.Printf("\nFeature saved to: .fogit/features/%s\n", filename)

	// Auto-commit if enabled in config
	if cfg.AutoCommit {
		if err := features.AutoCommitFeature(feature, "Create", cfg); err != nil {
			// Don't fail the command if auto-commit fails
			// Just warn the user
			logger.Warn("failed to auto-commit feature", "error", err, "feature", feature.Name)
			fmt.Println("You can manually commit with: git add .fogit/ && git commit")
		} else {
			fmt.Println("\n✓ Feature auto-committed to Git")
		}
	}

	return nil
}

// autoCommitFeature commits the feature file to Git if in a Git repository

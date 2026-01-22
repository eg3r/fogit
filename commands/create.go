package commands

import (
	"github.com/spf13/cobra"
)

// createCmd is a top-level alias for "fogit feature create"
// Per spec/specification/08-interface.md: fogit create is a shorthand for explicit feature creation
var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new feature (explicit)",
	Long: `Create a new feature. Fails if feature already exists.

This is an alias for 'fogit feature create'. Use 'fogit feature' for smart 
create-or-switch behavior that checks if a feature exists first.

Examples:
  # Create a new feature explicitly
  fogit create "New Feature"

  # Create with description and type
  fogit create "Login Page" -d "User login form" --type ui-component

  # Create with priority and tags
  fogit create "API Rate Limiting" -p high --tags security,performance

  # Create with organization metadata
  fogit create "Payment Integration" --category billing --team payments --epic checkout

  # Create on isolated branch (new Git branch)
  fogit create "Experimental Feature" --isolate

  # Create on same branch (shared strategy)
  fogit create "Quick Fix" --same`,
	Args: cobra.ExactArgs(1),
	RunE: runFeatureCreate, // Reuse the existing function from feature.go
}

func init() {
	// Register the same flags as featureCreateCmd
	registerFeatureFlags(createCmd)
	rootCmd.AddCommand(createCmd)
}

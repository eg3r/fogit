package commands

import (
	"github.com/spf13/cobra"
)

// CommonFilterFlags holds values for common filter flags used across commands.
type CommonFilterFlags struct {
	State    string
	Priority string
	Type     string
	Category string
}

// RegisterFilterFlags adds common filter flags to a command.
// These flags are used by list, search, and other filter-based commands.
//
// Usage:
//
//	var filterFlags CommonFilterFlags
//	func init() {
//	    RegisterFilterFlags(myCmd, &filterFlags)
//	}
func RegisterFilterFlags(cmd *cobra.Command, flags *CommonFilterFlags) {
	cmd.Flags().StringVar(&flags.State, "state", "", "Filter by state (open, in-progress, closed)")
	cmd.Flags().StringVar(&flags.Priority, "priority", "", "Filter by priority (low, medium, high, critical)")
	cmd.Flags().StringVar(&flags.Type, "type", "", "Filter by type")
	cmd.Flags().StringVar(&flags.Category, "category", "", "Filter by category")
}

// OrganizationFlags holds values for organization-related flags.
type OrganizationFlags struct {
	Domain string
	Team   string
	Epic   string
}

// RegisterOrganizationFlags adds organization-related flags to a command.
// These flags are used by list, update, and feature commands.
func RegisterOrganizationFlags(cmd *cobra.Command, flags *OrganizationFlags) {
	cmd.Flags().StringVar(&flags.Domain, "domain", "", "Filter/set by domain")
	cmd.Flags().StringVar(&flags.Team, "team", "", "Filter/set by team")
	cmd.Flags().StringVar(&flags.Epic, "epic", "", "Filter/set by epic")
}

// OutputFlags holds values for output formatting flags.
type OutputFlags struct {
	Format string
	Sort   string
}

// RegisterOutputFlags adds output formatting flags to a command.
// These flags are used by list, search, show, and other output commands.
func RegisterOutputFlags(cmd *cobra.Command, flags *OutputFlags) {
	cmd.Flags().StringVar(&flags.Format, "format", "table", "Output format: table, json, csv")
	cmd.Flags().StringVar(&flags.Sort, "sort", "created", "Sort by field: name, priority, created, modified")
}

// RegisterFormatFlag adds only the format flag to a command.
// Use this when sort is not needed (e.g., search command).
func RegisterFormatFlag(cmd *cobra.Command, format *string) {
	cmd.Flags().StringVar(format, "format", "table", "Output format: table, json, csv")
}

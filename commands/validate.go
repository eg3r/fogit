package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eg3r/fogit/internal/features/validator"
	"github.com/eg3r/fogit/internal/printer"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate feature data and relationships",
	Long: `Validate all features and relationships for integrity issues.

Checks performed:
  [E001] Orphaned relationships - target feature doesn't exist
  [E002] Missing inverse relationships - bidirectional relationship incomplete
  [E003] Dangling inverse - inverse exists but forward relationship missing
  [E004] Schema violations - invalid relationship structure
  [E005] Cycle violations - cycles in categories where not allowed
  [E006] Version constraint violations - target version doesn't satisfy constraint

Use --fix to attempt automatic repair of fixable issues.`,
	RunE: runValidate,
}

var (
	validateFix    bool
	validateReport string
	validateQuiet  bool
)

func init() {
	validateCmd.Flags().BoolVar(&validateFix, "fix", false, "Attempt to fix issues automatically")
	validateCmd.Flags().StringVar(&validateReport, "report", "", "Write validation report to file")
	validateCmd.Flags().BoolVar(&validateQuiet, "quiet", false, "Suppress output, only set exit code")
	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	// Get command context
	cmdCtx, err := GetCommandContext()
	if err != nil {
		return err
	}

	repo := cmdCtx.Repo
	cfg := cmdCtx.Config

	// Apply timeout for validation operation
	ctx, cancel := WithValidateTimeout(cmd.Context())
	defer cancel()

	if !validateQuiet {
		fmt.Println("Validating .fogit/features/...")
		fmt.Println()
	}

	// Create validator and run validation
	v := validator.New(repo, cfg)
	result, err := v.Validate(ctx)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Attempt fixes if requested
	if validateFix && result.HasFixableIssues() {
		fixer := validator.NewAutoFixer(repo, cfg, false)
		fixResult, err := fixer.AttemptFixes(ctx, result.Issues)
		if err != nil {
			return fmt.Errorf("auto-fix failed: %w", err)
		}

		if !validateQuiet {
			printer.PrintFixResult(fixResult)
		}

		// Re-validate after fixes
		result, err = v.Validate(ctx)
		if err != nil {
			return fmt.Errorf("re-validation failed: %w", err)
		}
	}

	// Output results
	if !validateQuiet {
		printer.PrintValidationResult(result)
	}

	// Write report if requested
	if validateReport != "" {
		output := printer.FormatValidationReport(result)
		if err := os.WriteFile(validateReport, []byte(output), 0600); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
		if !validateQuiet {
			fmt.Printf("Report written to %s\n", validateReport)
		}
	}

	// Return exit code 4 if errors found (per spec)
	if result.HasErrors() {
		os.Exit(4)
	}

	return nil
}

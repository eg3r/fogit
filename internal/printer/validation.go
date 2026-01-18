package printer

import (
	"fmt"
	"strings"

	"github.com/eg3r/fogit/internal/features/validator"
)

// PrintValidationResult outputs validation results to stdout
func PrintValidationResult(result *validator.ValidationResult) {
	errors := result.FilterBySeverity(validator.SeverityError)
	warnings := result.FilterBySeverity(validator.SeverityWarning)

	if len(errors) > 0 {
		fmt.Println("ERRORS:")
		for _, issue := range errors {
			fmt.Printf("  [%s] %s: %s\n", issue.Code, issue.FileName, issue.Message)
		}
		fmt.Println()
	}

	if len(warnings) > 0 {
		fmt.Println("WARNINGS:")
		for _, issue := range warnings {
			fmt.Printf("  [%s] %s: %s\n", issue.Code, issue.FileName, issue.Message)
		}
		fmt.Println()
	}

	if len(result.Issues) == 0 {
		fmt.Printf("All %d features validated\n", result.FeaturesCount)
		fmt.Printf("All %d relationships consistent\n", result.RelCount)
		fmt.Println("No cycles detected")
	} else {
		fmt.Printf("Summary: %d errors, %d warnings\n", result.Errors, result.Warnings)
		if result.HasFixableIssues() {
			fmt.Println("Run 'fogit validate --fix' to attempt automatic repair.")
		}
	}
}

// PrintFixResult outputs fix results to stdout
func PrintFixResult(result *validator.FixResult) {
	if !result.HasFixes() && !result.HasFailures() {
		return
	}

	if result.HasFixes() {
		fmt.Printf("Fixed %d issues:\n", result.TotalFixed())
		for _, msg := range result.Fixed {
			fmt.Printf("  %s\n", msg)
		}
	}

	if result.HasFailures() {
		fmt.Printf("Failed to fix %d issues:\n", result.TotalFailed())
		for _, msg := range result.Failed {
			fmt.Printf("  %s\n", msg)
		}
	}

	fmt.Println()
}

// FormatValidationReport formats a validation result as a report string
func FormatValidationReport(result *validator.ValidationResult) string {
	var sb strings.Builder

	errors := result.FilterBySeverity(validator.SeverityError)
	warnings := result.FilterBySeverity(validator.SeverityWarning)

	if len(errors) > 0 {
		sb.WriteString("ERRORS:\n")
		for _, issue := range errors {
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", issue.Code, issue.FileName, issue.Message))
		}
		sb.WriteString("\n")
	}

	if len(warnings) > 0 {
		sb.WriteString("WARNINGS:\n")
		for _, issue := range warnings {
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", issue.Code, issue.FileName, issue.Message))
		}
		sb.WriteString("\n")
	}

	if len(result.Issues) == 0 {
		sb.WriteString(fmt.Sprintf("All %d features validated\n", result.FeaturesCount))
		sb.WriteString(fmt.Sprintf("All %d relationships consistent\n", result.RelCount))
		sb.WriteString("No cycles detected\n")
	} else {
		sb.WriteString(fmt.Sprintf("Summary: %d errors, %d warnings\n", result.Errors, result.Warnings))
		if result.HasFixableIssues() {
			sb.WriteString("Run 'fogit validate --fix' to attempt automatic repair.\n")
		}
	}

	return sb.String()
}

// FormatFixReport formats a fix result as a report string
func FormatFixReport(result *validator.FixResult) string {
	var sb strings.Builder

	if result.HasFixes() {
		sb.WriteString(fmt.Sprintf("Fixed %d issues:\n", result.TotalFixed()))
		for _, msg := range result.Fixed {
			sb.WriteString(fmt.Sprintf("  %s\n", msg))
		}
	}

	if result.HasFailures() {
		sb.WriteString(fmt.Sprintf("Failed to fix %d issues:\n", result.TotalFailed()))
		for _, msg := range result.Failed {
			sb.WriteString(fmt.Sprintf("  %s\n", msg))
		}
	}

	return sb.String()
}

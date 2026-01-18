package validator

// IssueCode represents validation error codes
type IssueCode string

const (
	CodeOrphanedRelationship       IssueCode = "E001"
	CodeMissingInverse             IssueCode = "E002"
	CodeDanglingInverse            IssueCode = "E003"
	CodeSchemaViolation            IssueCode = "E004"
	CodeCycleViolation             IssueCode = "E005"
	CodeVersionConstraintViolation IssueCode = "E006"
)

// IssueCodeDescriptions provides human-readable descriptions for each code
var IssueCodeDescriptions = map[IssueCode]string{
	CodeOrphanedRelationship:       "Orphaned relationship - target feature doesn't exist",
	CodeMissingInverse:             "Missing inverse relationship - bidirectional relationship incomplete",
	CodeDanglingInverse:            "Dangling inverse - inverse exists but forward relationship missing",
	CodeSchemaViolation:            "Schema violation - invalid relationship structure",
	CodeCycleViolation:             "Cycle violation - cycles in categories where not allowed",
	CodeVersionConstraintViolation: "Version constraint violation - target version doesn't satisfy constraint",
}

// Severity represents issue severity
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// ValidationIssue represents a validation problem found
type ValidationIssue struct {
	Code        IssueCode         // E001, E002, etc.
	Severity    Severity          // error, warning
	FeatureID   string            // Feature ID where issue was found
	FeatureName string            // Feature name for display
	FileName    string            // Feature file name
	Message     string            // Human-readable description
	Fixable     bool              // Can be auto-fixed
	Context     map[string]string // Additional context (targetID, relType, etc.)
}

// ValidationResult contains all validation results
type ValidationResult struct {
	Issues        []ValidationIssue
	FeaturesCount int
	RelCount      int
	Errors        int // Count of error-severity issues
	Warnings      int // Count of warning-severity issues
}

// HasErrors returns true if any error-severity issues exist
func (r *ValidationResult) HasErrors() bool {
	return r.Errors > 0
}

// HasWarnings returns true if any warning-severity issues exist
func (r *ValidationResult) HasWarnings() bool {
	return r.Warnings > 0
}

// HasFixableIssues returns true if any issues can be auto-fixed
func (r *ValidationResult) HasFixableIssues() bool {
	for _, issue := range r.Issues {
		if issue.Fixable {
			return true
		}
	}
	return false
}

// FilterBySeverity returns issues matching the given severity
func (r *ValidationResult) FilterBySeverity(severity Severity) []ValidationIssue {
	var filtered []ValidationIssue
	for _, issue := range r.Issues {
		if issue.Severity == severity {
			filtered = append(filtered, issue)
		}
	}
	return filtered
}

// FixResult represents the outcome of an auto-fix attempt
type FixResult struct {
	Fixed   []string // Descriptions of what was fixed
	Failed  []string // Descriptions of failed fixes
	Skipped []string // Descriptions of skipped (not fixable)
}

// TotalFixed returns the count of successfully fixed issues
func (r *FixResult) TotalFixed() int {
	return len(r.Fixed)
}

// TotalFailed returns the count of failed fix attempts
func (r *FixResult) TotalFailed() int {
	return len(r.Failed)
}

// HasFixes returns true if any fixes were applied
func (r *FixResult) HasFixes() bool {
	return len(r.Fixed) > 0
}

// HasFailures returns true if any fixes failed
func (r *FixResult) HasFailures() bool {
	return len(r.Failed) > 0
}

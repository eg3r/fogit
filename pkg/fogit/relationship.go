package fogit

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// VersionConstraint specifies a version requirement for a relationship target
// Per spec 06-data-model.md (commit 0a355fc): version can be either:
// - Simple versioning: positive integer (1, 2, 3)
// - Semantic versioning: semver string ("1.0.0", "1.1.0")
type VersionConstraint struct {
	Operator string      `yaml:"operator"`       // One of: =, >, <, >=, <=
	Version  interface{} `yaml:"version"`        // int for simple, string for semantic versioning
	Note     string      `yaml:"note,omitempty"` // Optional explanation
}

// ValidOperators lists all valid version constraint operators
var ValidOperators = []string{"=", ">", "<", ">=", "<="}

// semverRegex validates semantic version format (MAJOR.MINOR.PATCH)
var semverRegex = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// IsSimpleVersion returns true if this constraint uses simple (integer) versioning
func (vc *VersionConstraint) IsSimpleVersion() bool {
	if vc == nil {
		return false
	}
	switch vc.Version.(type) {
	case int, int64, float64:
		return true
	case string:
		// Check if it's a string representation of an integer
		s, ok := vc.Version.(string)
		if !ok {
			return false
		}
		_, err := strconv.Atoi(s)
		return err == nil && !strings.Contains(s, ".")
	}
	return false
}

// IsSemanticVersion returns true if this constraint uses semantic versioning
func (vc *VersionConstraint) IsSemanticVersion() bool {
	if vc == nil {
		return false
	}
	s, ok := vc.Version.(string)
	if !ok {
		return false
	}
	return semverRegex.MatchString(s)
}

// GetSimpleVersion returns the integer version value, or 0 if not applicable
func (vc *VersionConstraint) GetSimpleVersion() int {
	if vc == nil {
		return 0
	}
	switch v := vc.Version.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}

// GetSemanticVersion returns the semver string, or empty if not applicable
func (vc *VersionConstraint) GetSemanticVersion() string {
	if vc == nil {
		return ""
	}
	s, ok := vc.Version.(string)
	if !ok {
		return ""
	}
	if semverRegex.MatchString(s) {
		return s
	}
	return ""
}

// GetVersionString returns a string representation of the version for display
func (vc *VersionConstraint) GetVersionString() string {
	if vc == nil {
		return ""
	}
	switch v := vc.Version.(type) {
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.Itoa(int(v))
	case string:
		return v
	}
	return fmt.Sprintf("%v", vc.Version)
}

// IsValid checks if the version constraint is valid
func (vc *VersionConstraint) IsValid() error {
	if vc == nil {
		return nil
	}

	// Check operator
	validOp := false
	for _, op := range ValidOperators {
		if vc.Operator == op {
			validOp = true
			break
		}
	}
	if !validOp {
		return fmt.Errorf("invalid version constraint operator '%s', must be one of: =, >, <, >=, <=", vc.Operator)
	}

	// Check version is valid
	if vc.IsSimpleVersion() {
		v := vc.GetSimpleVersion()
		if v < 1 {
			return fmt.Errorf("version constraint version must be a positive integer >= 1, got %d", v)
		}
	} else if vc.IsSemanticVersion() {
		// Semantic version is valid if it matches the regex (already checked in IsSemanticVersion)
		return nil
	} else {
		return fmt.Errorf("invalid version constraint version '%v', must be a positive integer or semantic version (x.y.z)", vc.Version)
	}

	return nil
}

// IsSatisfiedBy checks if the given target version satisfies this constraint
// Per spec 06-data-model.md:
// - Simple versioning: integer comparison
// - Semantic versioning: semver ordering (MAJOR.MINOR.PATCH)
func (vc *VersionConstraint) IsSatisfiedBy(targetVersionStr string) bool {
	if vc == nil {
		return true // No constraint means any version is acceptable
	}

	if vc.IsSemanticVersion() {
		return vc.compareSemver(targetVersionStr)
	}
	return vc.compareSimple(targetVersionStr)
}

// compareSimple compares using simple integer versioning
func (vc *VersionConstraint) compareSimple(targetVersionStr string) bool {
	targetInt := extractMajorVersion(targetVersionStr)
	constraintInt := vc.GetSimpleVersion()

	switch vc.Operator {
	case "=":
		return targetInt == constraintInt
	case ">":
		return targetInt > constraintInt
	case "<":
		return targetInt < constraintInt
	case ">=":
		return targetInt >= constraintInt
	case "<=":
		return targetInt <= constraintInt
	}
	return false
}

// compareSemver compares using semantic versioning (MAJOR.MINOR.PATCH)
func (vc *VersionConstraint) compareSemver(targetVersionStr string) bool {
	constraintSemver := vc.GetSemanticVersion()
	if constraintSemver == "" {
		return false
	}

	// Parse target version - if it's a simple integer, treat as x.0.0
	targetParts := parseSemverParts(targetVersionStr)
	constraintParts := parseSemverParts(constraintSemver)

	cmp := compareSemverParts(targetParts, constraintParts)

	switch vc.Operator {
	case "=":
		return cmp == 0
	case ">":
		return cmp > 0
	case "<":
		return cmp < 0
	case ">=":
		return cmp >= 0
	case "<=":
		return cmp <= 0
	}
	return false
}

// extractMajorVersion extracts the major version from a version string
// "3" -> 3, "2.1.0" -> 2
func extractMajorVersion(versionStr string) int {
	// Try parsing as integer first (simple versioning)
	if v, err := strconv.Atoi(versionStr); err == nil {
		return v
	}

	// Try extracting major from semantic version
	parts := strings.Split(versionStr, ".")
	if len(parts) >= 1 {
		if v, err := strconv.Atoi(parts[0]); err == nil {
			return v
		}
	}

	return 0
}

// parseSemverParts parses a version string into [major, minor, patch]
func parseSemverParts(versionStr string) [3]int {
	// Handle simple integer version as x.0.0
	if v, err := strconv.Atoi(versionStr); err == nil {
		return [3]int{v, 0, 0}
	}

	parts := strings.Split(versionStr, ".")
	var result [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		if v, err := strconv.Atoi(parts[i]); err == nil {
			result[i] = v
		}
	}
	return result
}

// compareSemverParts compares two semver part arrays
// Returns -1 if a < b, 0 if a == b, 1 if a > b
func compareSemverParts(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// Relationship represents a link between features
type Relationship struct {
	ID                string             `yaml:"id"`
	Type              RelationshipType   `yaml:"type"`
	TargetID          string             `yaml:"target_id"`
	TargetName        string             `yaml:"target_name"`
	Description       string             `yaml:"description,omitempty"`
	CreatedAt         time.Time          `yaml:"created_at"`
	VersionConstraint *VersionConstraint `yaml:"version_constraint,omitempty"`
}

// RelationshipType represents the type of relationship (dynamic, configured via config)
type RelationshipType string

// NewRelationship creates a new relationship
func NewRelationship(relType RelationshipType, targetID, targetName string) Relationship {
	return Relationship{
		ID:         uuid.New().String(),
		Type:       relType,
		TargetID:   targetID,
		TargetName: targetName,
		CreatedAt:  time.Now().UTC(),
	}
}

// ValidateWithConfig checks if the relationship is valid against the provided config
func (r *Relationship) ValidateWithConfig(config *Config) error {
	// Check if type exists in config
	typeConfig, exists := config.Relationships.Types[string(r.Type)]
	if !exists {
		// Check aliases
		found := false
		for typeName, tc := range config.Relationships.Types {
			for _, alias := range tc.Aliases {
				if alias == string(r.Type) {
					// Update type to canonical name
					r.Type = RelationshipType(typeName)
					typeConfig = tc
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return fmt.Errorf("relationship type '%s' not defined in config", r.Type)
		}
	}

	// Check category exists
	_, exists = config.Relationships.Categories[typeConfig.Category]
	if !exists {
		return fmt.Errorf("relationship type '%s' references unknown category '%s'", r.Type, typeConfig.Category)
	}

	if r.TargetID == "" {
		return ErrEmptyTargetID
	}

	return nil
}

// GetCategory returns the category of this relationship based on config
func (r *Relationship) GetCategory(config *Config) string {
	if typeConfig, exists := config.Relationships.Types[string(r.Type)]; exists {
		return typeConfig.Category
	}
	return config.Relationships.Defaults.Category
}

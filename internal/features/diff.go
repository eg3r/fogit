package features

import (
	"reflect"
	"strings"
	"time"

	"github.com/eg3r/fogit/internal/common"
	"github.com/eg3r/fogit/pkg/fogit"
)

// VersionDiff represents the differences between two feature versions
type VersionDiff struct {
	FeatureID      string      `json:"feature_id" yaml:"feature_id"`
	FeatureName    string      `json:"feature_name" yaml:"feature_name"`
	Version1       string      `json:"version1" yaml:"version1"`
	Version2       string      `json:"version2" yaml:"version2"`
	Changes        []FieldDiff `json:"changes" yaml:"changes"`
	HasDifferences bool        `json:"has_differences" yaml:"has_differences"`
}

// FieldDiff represents a difference in a single field
type FieldDiff struct {
	Field      string `json:"field" yaml:"field"`
	OldValue   string `json:"old_value" yaml:"old_value"`
	NewValue   string `json:"new_value" yaml:"new_value"`
	ChangeType string `json:"change_type" yaml:"change_type"` // "added", "removed", "modified"
}

// CalculateVersionDiff compares two versions of a feature and returns the differences
func CalculateVersionDiff(feature *fogit.Feature, v1Key string, v1 *fogit.FeatureVersion, v2Key string, v2 *fogit.FeatureVersion) *VersionDiff {
	diff := &VersionDiff{
		FeatureID:   feature.ID,
		FeatureName: feature.Name,
		Version1:    v1Key,
		Version2:    v2Key,
		Changes:     []FieldDiff{},
	}

	// Compare CreatedAt
	if !v1.CreatedAt.Equal(v2.CreatedAt) {
		diff.Changes = append(diff.Changes, FieldDiff{
			Field:      "created_at",
			OldValue:   common.FormatDateTime(v1.CreatedAt),
			NewValue:   common.FormatDateTime(v2.CreatedAt),
			ChangeType: "modified",
		})
	}

	// Compare ModifiedAt
	if !v1.ModifiedAt.Equal(v2.ModifiedAt) {
		diff.Changes = append(diff.Changes, FieldDiff{
			Field:      "modified_at",
			OldValue:   common.FormatDateTime(v1.ModifiedAt),
			NewValue:   common.FormatDateTime(v2.ModifiedAt),
			ChangeType: "modified",
		})
	}

	// Compare ClosedAt
	v1Closed := FormatClosedAt(v1.ClosedAt)
	v2Closed := FormatClosedAt(v2.ClosedAt)
	if v1Closed != v2Closed {
		changeType := "modified"
		if v1.ClosedAt == nil && v2.ClosedAt != nil {
			changeType = "added"
		} else if v1.ClosedAt != nil && v2.ClosedAt == nil {
			changeType = "removed"
		}
		diff.Changes = append(diff.Changes, FieldDiff{
			Field:      "closed_at",
			OldValue:   v1Closed,
			NewValue:   v2Closed,
			ChangeType: changeType,
		})
	}

	// Compare Branch
	if v1.Branch != v2.Branch {
		changeType := "modified"
		if v1.Branch == "" {
			changeType = "added"
		} else if v2.Branch == "" {
			changeType = "removed"
		}
		diff.Changes = append(diff.Changes, FieldDiff{
			Field:      "branch",
			OldValue:   v1.Branch,
			NewValue:   v2.Branch,
			ChangeType: changeType,
		})
	}

	// Compare Authors
	if !reflect.DeepEqual(v1.Authors, v2.Authors) {
		changeType := "modified"
		v1Authors := FormatAuthors(v1.Authors)
		v2Authors := FormatAuthors(v2.Authors)
		if len(v1.Authors) == 0 {
			changeType = "added"
		} else if len(v2.Authors) == 0 {
			changeType = "removed"
		}
		diff.Changes = append(diff.Changes, FieldDiff{
			Field:      "authors",
			OldValue:   v1Authors,
			NewValue:   v2Authors,
			ChangeType: changeType,
		})
	}

	// Compare Notes
	if v1.Notes != v2.Notes {
		changeType := "modified"
		if v1.Notes == "" {
			changeType = "added"
		} else if v2.Notes == "" {
			changeType = "removed"
		}
		diff.Changes = append(diff.Changes, FieldDiff{
			Field:      "notes",
			OldValue:   v1.Notes,
			NewValue:   v2.Notes,
			ChangeType: changeType,
		})
	}

	diff.HasDifferences = len(diff.Changes) > 0

	return diff
}

// FormatClosedAt formats a closed timestamp for display
func FormatClosedAt(t *time.Time) string {
	if t == nil {
		return "(open)"
	}
	return common.FormatDateTime(*t)
}

// FormatAuthors formats a list of authors for display
func FormatAuthors(authors []string) string {
	if len(authors) == 0 {
		return "(none)"
	}
	return strings.Join(authors, ", ")
}

// GetChangeSymbol returns a symbol for the change type
func GetChangeSymbol(changeType string) string {
	switch changeType {
	case "added":
		return "[+]"
	case "removed":
		return "[-]"
	default:
		return "[~]"
	}
}

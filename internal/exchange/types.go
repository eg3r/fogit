// Package exchange provides import/export functionality for features.
package exchange

import (
	"time"

	"github.com/eg3r/fogit/pkg/fogit"
)

// ExportData represents the exported data structure per spec
type ExportData struct {
	FogitVersion string           `json:"fogit_version" yaml:"fogit_version"`
	ExportedAt   string           `json:"exported_at" yaml:"exported_at"`
	Repository   string           `json:"repository" yaml:"repository"`
	Features     []*ExportFeature `json:"features" yaml:"features"`
}

// ExportFeature represents a feature in export format
type ExportFeature struct {
	ID            string                    `json:"id" yaml:"id"`
	Name          string                    `json:"name" yaml:"name"`
	Description   string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Tags          []string                  `json:"tags,omitempty" yaml:"tags,omitempty"`
	Files         []string                  `json:"files,omitempty" yaml:"files,omitempty"`
	Versions      map[string]*ExportVersion `json:"versions,omitempty" yaml:"versions,omitempty"`
	Relationships []ExportRelationship      `json:"relationships,omitempty" yaml:"relationships,omitempty"`
	Metadata      map[string]interface{}    `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	// Computed fields for convenience
	State          string `json:"state" yaml:"state"`
	CurrentVersion string `json:"current_version" yaml:"current_version"`
}

// ExportVersion represents a feature version in export format
type ExportVersion struct {
	CreatedAt  string   `json:"created_at" yaml:"created_at"`
	ModifiedAt string   `json:"modified_at,omitempty" yaml:"modified_at,omitempty"`
	ClosedAt   string   `json:"closed_at,omitempty" yaml:"closed_at,omitempty"`
	Branch     string   `json:"branch,omitempty" yaml:"branch,omitempty"`
	Authors    []string `json:"authors,omitempty" yaml:"authors,omitempty"`
	Notes      string   `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// ExportRelationship represents a relationship in export format
type ExportRelationship struct {
	ID                string                   `json:"id" yaml:"id"`
	Type              string                   `json:"type" yaml:"type"`
	TargetID          string                   `json:"target_id" yaml:"target_id"`
	TargetName        string                   `json:"target_name" yaml:"target_name"`
	TargetExists      bool                     `json:"target_exists" yaml:"target_exists"`
	Description       string                   `json:"description,omitempty" yaml:"description,omitempty"`
	CreatedAt         string                   `json:"created_at" yaml:"created_at"`
	VersionConstraint *ExportVersionConstraint `json:"version_constraint,omitempty" yaml:"version_constraint,omitempty"`
}

// ExportVersionConstraint represents a version constraint in export format
type ExportVersionConstraint struct {
	Operator string      `json:"operator" yaml:"operator"`
	Version  interface{} `json:"version" yaml:"version"`
	Note     string      `json:"note,omitempty" yaml:"note,omitempty"`
}

// ConvertToExportFeature converts a domain feature to export format
func ConvertToExportFeature(f *fogit.Feature, featureIDs map[string]bool) *ExportFeature {
	ef := &ExportFeature{
		ID:             f.ID,
		Name:           f.Name,
		Description:    f.Description,
		Tags:           f.Tags,
		Files:          f.Files,
		Metadata:       f.Metadata,
		State:          string(f.DeriveState()),
		CurrentVersion: f.GetCurrentVersionKey(),
	}

	// Convert versions
	if len(f.Versions) > 0 {
		ef.Versions = make(map[string]*ExportVersion)
		for key, v := range f.Versions {
			ev := &ExportVersion{
				CreatedAt: v.CreatedAt.Format(time.RFC3339),
				Branch:    v.Branch,
				Authors:   v.Authors,
				Notes:     v.Notes,
			}
			if !v.ModifiedAt.IsZero() {
				ev.ModifiedAt = v.ModifiedAt.Format(time.RFC3339)
			}
			if v.ClosedAt != nil {
				ev.ClosedAt = v.ClosedAt.Format(time.RFC3339)
			}
			ef.Versions[key] = ev
		}
	}

	// Convert relationships
	if len(f.Relationships) > 0 {
		ef.Relationships = make([]ExportRelationship, 0, len(f.Relationships))
		for _, r := range f.Relationships {
			er := ExportRelationship{
				ID:           r.ID,
				Type:         string(r.Type),
				TargetID:     r.TargetID,
				TargetName:   r.TargetName,
				TargetExists: featureIDs[r.TargetID],
				Description:  r.Description,
				CreatedAt:    r.CreatedAt.Format(time.RFC3339),
			}
			if r.VersionConstraint != nil {
				er.VersionConstraint = &ExportVersionConstraint{
					Operator: r.VersionConstraint.Operator,
					Version:  r.VersionConstraint.Version,
					Note:     r.VersionConstraint.Note,
				}
			}
			ef.Relationships = append(ef.Relationships, er)
		}
	}

	return ef
}

// ConvertFromExportFeature converts an export feature to domain format
func ConvertFromExportFeature(ef *ExportFeature) *fogit.Feature {
	f := &fogit.Feature{
		ID:          ef.ID,
		Name:        ef.Name,
		Description: ef.Description,
		Tags:        ef.Tags,
		Files:       ef.Files,
		Metadata:    ef.Metadata,
	}

	// Convert versions
	if len(ef.Versions) > 0 {
		f.Versions = make(map[string]*fogit.FeatureVersion)
		for key, ev := range ef.Versions {
			fv := &fogit.FeatureVersion{
				Branch:  ev.Branch,
				Authors: ev.Authors,
				Notes:   ev.Notes,
			}

			if t, err := time.Parse(time.RFC3339, ev.CreatedAt); err == nil {
				fv.CreatedAt = t
			}
			if ev.ModifiedAt != "" {
				if t, err := time.Parse(time.RFC3339, ev.ModifiedAt); err == nil {
					fv.ModifiedAt = t
				}
			}
			if ev.ClosedAt != "" {
				if t, err := time.Parse(time.RFC3339, ev.ClosedAt); err == nil {
					fv.ClosedAt = &t
				}
			}

			f.Versions[key] = fv
		}
	}

	// Convert relationships
	if len(ef.Relationships) > 0 {
		f.Relationships = make([]fogit.Relationship, 0, len(ef.Relationships))
		for _, er := range ef.Relationships {
			r := fogit.Relationship{
				ID:          er.ID,
				Type:        fogit.RelationshipType(er.Type),
				TargetID:    er.TargetID,
				TargetName:  er.TargetName,
				Description: er.Description,
			}

			if t, err := time.Parse(time.RFC3339, er.CreatedAt); err == nil {
				r.CreatedAt = t
			}

			if er.VersionConstraint != nil {
				r.VersionConstraint = &fogit.VersionConstraint{
					Operator: er.VersionConstraint.Operator,
					Version:  er.VersionConstraint.Version,
					Note:     er.VersionConstraint.Note,
				}
			}

			f.Relationships = append(f.Relationships, r)
		}
	}

	return f
}

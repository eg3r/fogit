package fogit

import (
	"errors"
	"strings"
)

// Filter represents filtering criteria for listing features.
type Filter struct {
	// State filters
	State State // Filter by state (empty means all states)

	// Priority filters
	Priority Priority // Filter by priority (empty means all priorities)

	// Type filter
	Type string // Filter by type (empty means all types)

	// Organization filters
	Category string // Filter by category
	Domain   string // Filter by domain
	Team     string // Filter by team
	Epic     string // Filter by epic

	// Hierarchy filters
	Parent string // Filter by parent feature ID

	// Tags filter (AND logic)
	Tags []string // Filter by tags (all tags must match)

	// Contributor filter
	Contributor string // Filter by contributor email (matches any version author)

	// Search filter
	Search string // Search in name and description (case-insensitive)

	// Sorting
	SortBy    SortField // Field to sort by
	SortOrder SortOrder // Sort order (ascending/descending)
}

// SortField represents the field to sort by.
type SortField string

const (
	SortByName     SortField = "name"
	SortByPriority SortField = "priority"
	SortByCreated  SortField = "created"
	SortByModified SortField = "modified"
)

// SortOrder represents sort direction.
type SortOrder string

const (
	SortAscending  SortOrder = "asc"
	SortDescending SortOrder = "desc"
)

// Validation errors for filters.
var (
	ErrInvalidSortField = errors.New("invalid sort field: must be one of name, priority, created, modified")
	ErrInvalidSortOrder = errors.New("invalid sort order: must be asc or desc")
)

// IsValid checks if the sort field is valid.
func (s SortField) IsValid() bool {
	switch s {
	case SortByName, SortByPriority, SortByCreated, SortByModified:
		return true
	case "": // Empty is valid (no sorting specified)
		return true
	default:
		return false
	}
}

// IsValid checks if the sort order is valid.
func (s SortOrder) IsValid() bool {
	switch s {
	case SortAscending, SortDescending:
		return true
	case "": // Empty is valid (default to ascending)
		return true
	default:
		return false
	}
}

// Validate validates the filter criteria.
func (f *Filter) Validate() error {
	// Validate state if specified
	if f.State != "" && !f.State.IsValid() {
		return ErrInvalidState
	}

	// Validate priority if specified
	if f.Priority != "" && !f.Priority.IsValid() {
		return ErrInvalidPriority
	}

	// Validate sort field if specified
	if !f.SortBy.IsValid() {
		return ErrInvalidSortField
	}

	// Validate sort order if specified
	if !f.SortOrder.IsValid() {
		return ErrInvalidSortOrder
	}

	return nil
}

// Matches checks if a feature matches the filter criteria.
// Uses accessor methods to support both new (metadata) and deprecated (direct field) storage.
func (f *Filter) Matches(feature *Feature) bool {
	// State filter - use derived state
	if f.State != "" && feature.DeriveState() != f.State {
		return false
	}

	// Priority filter - use accessor for metadata support
	if f.Priority != "" && feature.GetPriority() != f.Priority {
		return false
	}

	// Type filter (case-insensitive) - use accessor for metadata support
	if f.Type != "" && !strings.EqualFold(feature.GetType(), f.Type) {
		return false
	}

	// Category filter (case-insensitive) - use accessor for metadata support
	if f.Category != "" && !strings.EqualFold(feature.GetCategory(), f.Category) {
		return false
	}

	// Domain filter (case-insensitive) - use accessor for metadata support
	if f.Domain != "" && !strings.EqualFold(feature.GetDomain(), f.Domain) {
		return false
	}

	// Team filter (case-insensitive) - use accessor for metadata support
	if f.Team != "" && !strings.EqualFold(feature.GetTeam(), f.Team) {
		return false
	}

	// Epic filter (case-insensitive) - use accessor for metadata support
	if f.Epic != "" && !strings.EqualFold(feature.GetEpic(), f.Epic) {
		return false
	}

	// Parent filter - check if feature has this parent via contained-by relationship
	if f.Parent != "" {
		hasParent := false
		for _, rel := range feature.Relationships {
			// Check for contained-by (spec) or parent (legacy)
			if (rel.Type == "contained-by" || rel.Type == "parent") && rel.TargetID == f.Parent {
				hasParent = true
				break
			}
		}
		if !hasParent {
			return false
		}
	}

	// Tags filter (AND logic - all tags must match)
	if len(f.Tags) > 0 {
		featureTagSet := make(map[string]bool)
		for _, tag := range feature.Tags {
			featureTagSet[strings.ToLower(tag)] = true
		}
		for _, filterTag := range f.Tags {
			if !featureTagSet[strings.ToLower(filterTag)] {
				return false
			}
		}
	}

	// Contributor filter - check if contributor is in any version's authors
	if f.Contributor != "" {
		hasContributor := false
		contributorLower := strings.ToLower(f.Contributor)
		for _, version := range feature.Versions {
			for _, author := range version.Authors {
				if strings.ToLower(author) == contributorLower || strings.Contains(strings.ToLower(author), contributorLower) {
					hasContributor = true
					break
				}
			}
			if hasContributor {
				break
			}
		}
		if !hasContributor {
			return false
		}
	}

	// Search filter (case-insensitive in name and description)
	if f.Search != "" {
		search := strings.ToLower(f.Search)
		nameMatch := strings.Contains(strings.ToLower(feature.Name), search)
		descMatch := strings.Contains(strings.ToLower(feature.Description), search)
		if !nameMatch && !descMatch {
			return false
		}
	}

	return true
}

// NewFilter creates a new Filter with default values.
func NewFilter() *Filter {
	return &Filter{
		SortBy:    SortByCreated,
		SortOrder: SortDescending,
	}
}

package features

import (
	"fmt"

	"github.com/eg3r/fogit/pkg/fogit"
)

// DetermineTreeRelationshipTypes determines which relationship types to use for tree hierarchy.
// It requires explicit types to be provided via --type flag.
func DetermineTreeRelationshipTypes(cfg *fogit.Config, explicitTypes []string) ([]string, error) {
	// Types must be explicitly provided via --type flag
	if len(explicitTypes) == 0 {
		return nil, fmt.Errorf("--type flag is required: specify which relationship type(s) to use for the tree hierarchy")
	}

	// Validate provided types exist in config
	for _, hierarchyType := range explicitTypes {
		if _, exists := cfg.Relationships.Types[hierarchyType]; !exists {
			return nil, fmt.Errorf("relationship type '%s' not defined in config", hierarchyType)
		}
	}
	return explicitTypes, nil
}

// FindRoots finds features that are roots of the hierarchy (have no outgoing relationships of the specified types)
func FindRoots(features []*fogit.Feature, hierarchyTypes []string) []*fogit.Feature {
	// Build set of all features that have any of the hierarchy relationships (are "children")
	childIDs := make(map[string]bool)
	for _, f := range features {
		for _, rel := range f.Relationships {
			for _, hType := range hierarchyTypes {
				if string(rel.Type) == hType {
					childIDs[f.ID] = true
					break
				}
			}
		}
	}

	// Find features that aren't children (no outgoing hierarchy relationships)
	var roots []*fogit.Feature
	for _, f := range features {
		if !childIDs[f.ID] {
			roots = append(roots, f)
		}
	}

	return roots
}

// FindChildren finds features that have a relationship of the specified types pointing to the parentID
func FindChildren(parentID string, allFeatures []*fogit.Feature, hierarchyTypes []string) []*fogit.Feature {
	var children []*fogit.Feature
	for _, f := range allFeatures {
		for _, rel := range f.Relationships {
			// Check if this feature has any of the hierarchy relationships to parentID
			for _, hType := range hierarchyTypes {
				if string(rel.Type) == hType && rel.TargetID == parentID {
					children = append(children, f)
					break
				}
			}
		}
	}
	return children
}

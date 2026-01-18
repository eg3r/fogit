package features

import (
	"fmt"

	"github.com/eg3r/fogit/pkg/fogit"
)

// DetermineTreeRelationshipTypes determines which relationship types to use for tree hierarchy.
// It uses the following precedence:
// 1. Explicitly provided types (if any)
// 2. Default tree type from config
// 3. First non-cyclic relationship type found
// 4. First relationship type (if all allow cycles)
func DetermineTreeRelationshipTypes(cfg *fogit.Config, explicitTypes []string) ([]string, error) {
	// If explicit types provided, validate and use them
	if len(explicitTypes) > 0 {
		for _, hierarchyType := range explicitTypes {
			if _, exists := cfg.Relationships.Types[hierarchyType]; !exists {
				return nil, fmt.Errorf("relationship type '%s' not defined in config", hierarchyType)
			}
		}
		return explicitTypes, nil
	}

	// Use default from config if available
	if cfg.Relationships.Defaults.TreeType != "" {
		return []string{cfg.Relationships.Defaults.TreeType}, nil
	}

	// Fallback: find first relationship type in a non-cyclic category
	for typeName, typeConfig := range cfg.Relationships.Types {
		if cat, exists := cfg.Relationships.Categories[typeConfig.Category]; exists {
			if !cat.AllowCycles {
				return []string{typeName}, nil
			}
		}
	}

	// If all categories allow cycles, just use the first type
	if len(cfg.Relationships.Types) > 0 {
		for typeName := range cfg.Relationships.Types {
			return []string{typeName}, nil
		}
	}

	return nil, fmt.Errorf("no tree relationship type configured and no relationship types defined")
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

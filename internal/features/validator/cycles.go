package validator

import (
	"fmt"
)

// checkCycles detects cycles in categories where not allowed (E005)
// Optimized: builds graph once per category and detects all cycles in single pass
func (v *Validator) checkCycles(result *ValidationResult) {
	// Get categories that need strict cycle detection
	strictCategories := make(map[string]bool)
	for name, cat := range v.config.Relationships.Categories {
		if !cat.AllowCycles && cat.CycleDetection == "strict" {
			strictCategories[name] = true
		}
	}

	if len(strictCategories) == 0 {
		return
	}

	// Get forward relationship types (exclude inverse types to avoid false positives)
	// When A depends-on B, B gets required-by A - we only want to traverse depends-on
	forwardTypes := v.getForwardRelationshipTypes()

	// Check each strict category
	for category := range strictCategories {
		// Also get types that belong to this category
		categoryTypes := make(map[string]bool)
		for typeName, typeConfig := range v.config.Relationships.Types {
			if typeConfig.Category == category && forwardTypes[typeName] {
				categoryTypes[typeName] = true
			}
		}

		cycles := v.detectCyclesInCategory(category, categoryTypes)

		// Report each cycle once (using the first node as the reporter)
		reported := make(map[string]bool)
		for _, cycle := range cycles {
			if len(cycle) < 2 {
				continue
			}

			// Use smallest ID as cycle identifier to avoid duplicates
			cycleKey := v.getCycleKey(cycle)
			if reported[cycleKey] {
				continue
			}
			reported[cycleKey] = true

			// Report on the first feature in the cycle
			featureID := cycle[0]
			feature := v.featureMap[featureID]
			if feature == nil {
				continue
			}

			// Build readable cycle path
			cyclePath := v.formatCyclePath(cycle)

			result.Issues = append(result.Issues, ValidationIssue{
				Code:        CodeCycleViolation,
				Severity:    SeverityError,
				FeatureID:   featureID,
				FeatureName: feature.Name,
				FileName:    GetFeatureFileName(feature.Name),
				Message:     fmt.Sprintf("Cycle detected in %s category: %s", category, cyclePath),
				Fixable:     false,
				Context: map[string]string{
					"category": category,
					"cycle":    cyclePath,
				},
			})
		}
	}
}

// getForwardRelationshipTypes returns a set of relationship types that are "forward" types
// For relationship pairs (depends-on/required-by), we pick one direction to avoid counting
// the same edge twice. We use lexicographic ordering to consistently pick the "forward" one.
func (v *Validator) getForwardRelationshipTypes() map[string]bool {
	forward := make(map[string]bool)

	for name, rt := range v.config.Relationships.Types {
		if rt.Bidirectional {
			// Bidirectional types are always included
			forward[name] = true
		} else if rt.Inverse == "" {
			// No inverse defined - include it
			forward[name] = true
		} else {
			// Has an inverse - pick the lexicographically smaller one as "forward"
			if name < rt.Inverse {
				forward[name] = true
			}
			// Otherwise, the inverse type is the "forward" one
		}
	}

	return forward
}

// detectCyclesInCategory finds all cycles in a category using only forward relationship types
func (v *Validator) detectCyclesInCategory(_ string, categoryTypes map[string]bool) [][]string {
	// Build adjacency list for this category using only forward types
	adj := make(map[string][]string)
	for _, feature := range v.features {
		for _, rel := range feature.Relationships {
			// Only include relationship types in the categoryTypes set
			if !categoryTypes[string(rel.Type)] {
				continue
			}
			adj[feature.ID] = append(adj[feature.ID], rel.TargetID)
		}
	}

	// Find all cycles using DFS with path tracking
	// WHITE (0) = unvisited, GRAY (1) = in current path, BLACK (2) = fully processed
	color := make(map[string]int)
	var cycles [][]string

	var dfs func(node string, path []string)
	dfs = func(node string, path []string) {
		color[node] = 1 // GRAY - in current path
		path = append(path, node)

		for _, neighbor := range adj[node] {
			if color[neighbor] == 0 {
				dfs(neighbor, path)
			} else if color[neighbor] == 1 {
				// Back edge found - extract cycle from path
				// Find where neighbor appears in path
				cycleStart := -1
				for i, id := range path {
					if id == neighbor {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					// Extract the cycle portion of the path
					cycle := make([]string, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
			}
		}

		color[node] = 2 // BLACK - fully processed
	}

	// Run DFS from all unvisited nodes
	for _, feature := range v.features {
		if color[feature.ID] == 0 {
			dfs(feature.ID, nil)
		}
	}

	return cycles
}

// getCycleKey returns a unique key for a cycle (to avoid reporting same cycle multiple times)
func (v *Validator) getCycleKey(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}

	// Find the minimum ID and rotate cycle to start with it
	minIdx := 0
	for i, id := range cycle {
		if id < cycle[minIdx] {
			minIdx = i
		}
	}

	// Build key from rotated cycle
	key := ""
	for i := 0; i < len(cycle); i++ {
		key += cycle[(minIdx+i)%len(cycle)] + "->"
	}
	return key
}

// formatCyclePath creates a readable cycle path string
func (v *Validator) formatCyclePath(cycle []string) string {
	if len(cycle) == 0 {
		return ""
	}

	path := ""
	for i, id := range cycle {
		feature := v.featureMap[id]
		name := id
		if feature != nil {
			name = feature.Name
		}
		if i > 0 {
			path += " -> "
		}
		path += name
	}

	// Complete the cycle display
	if len(cycle) > 0 {
		feature := v.featureMap[cycle[0]]
		name := cycle[0]
		if feature != nil {
			name = feature.Name
		}
		path += " -> " + name
	}

	return path
}

// DetectAllCycles finds all cycles in the relationship graph for a given category
// Returns a list of cycle paths (each path is a slice of feature IDs)
func (v *Validator) DetectAllCycles(category string) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for _, feature := range v.features {
		if !visited[feature.ID] {
			path := []string{}
			v.findCyclesDFS(feature.ID, category, visited, recStack, path, &cycles)
		}
	}

	return cycles
}

// findCyclesDFS is a helper for DetectAllCycles
func (v *Validator) findCyclesDFS(current, category string, visited, recStack map[string]bool, path []string, cycles *[][]string) {
	visited[current] = true
	recStack[current] = true
	path = append(path, current)

	feature := v.featureMap[current]
	if feature == nil {
		recStack[current] = false
		return
	}

	for _, rel := range feature.Relationships {
		if rel.GetCategory(v.config) != category {
			continue
		}

		if !visited[rel.TargetID] {
			v.findCyclesDFS(rel.TargetID, category, visited, recStack, path, cycles)
		} else if recStack[rel.TargetID] {
			// Found a cycle - extract the cycle path
			cycleStart := -1
			for i, id := range path {
				if id == rel.TargetID {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cyclePath := make([]string, len(path)-cycleStart+1)
				copy(cyclePath, path[cycleStart:])
				cyclePath[len(cyclePath)-1] = rel.TargetID // Complete the cycle
				*cycles = append(*cycles, cyclePath)
			}
		}
	}

	recStack[current] = false
}

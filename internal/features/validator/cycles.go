package validator

import (
	"fmt"
)

// checkCycles detects cycles in categories where not allowed (E005)
func (v *Validator) checkCycles(result *ValidationResult) {
	for _, feature := range v.features {
		fileName := GetFeatureFileName(feature.Name)

		for _, rel := range feature.Relationships {
			category := rel.GetCategory(v.config)
			catConfig, exists := v.config.Relationships.Categories[category]
			if !exists {
				continue
			}

			if !catConfig.AllowCycles && catConfig.CycleDetection == "strict" {
				if v.hasCycle(feature.ID, rel.TargetID, category) {
					target := v.featureMap[rel.TargetID]
					targetName := rel.TargetID
					if target != nil {
						targetName = target.Name
					}

					result.Issues = append(result.Issues, ValidationIssue{
						Code:        CodeCycleViolation,
						Severity:    SeverityError,
						FeatureID:   feature.ID,
						FeatureName: feature.Name,
						FileName:    fileName,
						Message: fmt.Sprintf("Cycle detected in %s category: %s -> %s creates circular dependency",
							category, feature.Name, targetName),
						Fixable: false,
						Context: map[string]string{
							"category":   category,
							"targetID":   rel.TargetID,
							"targetName": targetName,
						},
					})
				}
			}
		}
	}
}

// hasCycle checks if adding edge sourceID->targetID creates a cycle
func (v *Validator) hasCycle(sourceID, targetID, category string) bool {
	visited := make(map[string]bool)
	return v.dfsCycleCheck(targetID, sourceID, category, visited)
}

// dfsCycleCheck performs DFS to find if target is reachable from current
func (v *Validator) dfsCycleCheck(current, target, category string, visited map[string]bool) bool {
	if current == target {
		return true
	}
	if visited[current] {
		return false
	}
	visited[current] = true

	feature := v.featureMap[current]
	if feature == nil {
		return false
	}

	for _, rel := range feature.Relationships {
		if rel.GetCategory(v.config) == category {
			if v.dfsCycleCheck(rel.TargetID, target, category, visited) {
				return true
			}
		}
	}

	return false
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

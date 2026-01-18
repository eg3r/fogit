package validator

import (
	"fmt"
)

// checkOrphanedRelationships finds relationships pointing to non-existent features (E001)
func (v *Validator) checkOrphanedRelationships(result *ValidationResult) {
	for _, feature := range v.features {
		fileName := GetFeatureFileName(feature.Name)

		for _, rel := range feature.Relationships {
			if _, exists := v.featureMap[rel.TargetID]; !exists {
				result.Issues = append(result.Issues, ValidationIssue{
					Code:        CodeOrphanedRelationship,
					Severity:    SeverityError,
					FeatureID:   feature.ID,
					FeatureName: feature.Name,
					FileName:    fileName,
					Message:     fmt.Sprintf("Orphaned relationship %s -> target '%s' not found", rel.Type, rel.TargetID),
					Fixable:     true,
					Context: map[string]string{
						"relationType": string(rel.Type),
						"targetID":     rel.TargetID,
						"relationID":   rel.ID,
					},
				})
			}
		}
	}
}

// checkMissingInverses finds relationships without required inverse (E002)
func (v *Validator) checkMissingInverses(result *ValidationResult) {
	for _, feature := range v.features {
		fileName := GetFeatureFileName(feature.Name)

		for _, rel := range feature.Relationships {
			target, exists := v.featureMap[rel.TargetID]
			if !exists {
				continue // Already caught by E001
			}

			inverseType := v.getInverseType(string(rel.Type))
			if inverseType == "" || !v.config.Relationships.System.AutoCreateInverse {
				continue
			}

			hasInverse := false
			for _, targetRel := range target.Relationships {
				if string(targetRel.Type) == inverseType && targetRel.TargetID == feature.ID {
					hasInverse = true
					break
				}
			}

			if !hasInverse {
				targetFileName := GetFeatureFileName(target.Name)
				result.Issues = append(result.Issues, ValidationIssue{
					Code:        CodeMissingInverse,
					Severity:    SeverityError,
					FeatureID:   feature.ID,
					FeatureName: feature.Name,
					FileName:    fileName,
					Message: fmt.Sprintf("Missing inverse - %s -> %s, but %s has no %s -> %s",
						rel.Type, target.Name, targetFileName, inverseType, feature.Name),
					Fixable: true,
					Context: map[string]string{
						"relationType": string(rel.Type),
						"inverseType":  inverseType,
						"targetID":     target.ID,
						"targetName":   target.Name,
					},
				})
			}
		}
	}
}

// checkDanglingInverses finds inverse relationships without forward relationship (E003)
func (v *Validator) checkDanglingInverses(result *ValidationResult) {
	for _, feature := range v.features {
		fileName := GetFeatureFileName(feature.Name)

		for _, rel := range feature.Relationships {
			// Check if this is an inverse type
			forwardType := v.getForwardType(string(rel.Type))
			if forwardType == "" {
				continue // Not an inverse type
			}

			target := v.featureMap[rel.TargetID]
			if target == nil {
				continue // Already caught by E001
			}

			// Check if the forward relationship exists in the target
			hasForward := false
			for _, targetRel := range target.Relationships {
				if string(targetRel.Type) == forwardType && targetRel.TargetID == feature.ID {
					hasForward = true
					break
				}
			}

			if !hasForward {
				result.Issues = append(result.Issues, ValidationIssue{
					Code:        CodeDanglingInverse,
					Severity:    SeverityError,
					FeatureID:   feature.ID,
					FeatureName: feature.Name,
					FileName:    fileName,
					Message: fmt.Sprintf("Dangling inverse - has %s -> %s, but target has no %s -> %s",
						rel.Type, target.Name, forwardType, feature.Name),
					Fixable: true,
					Context: map[string]string{
						"relationType": string(rel.Type),
						"forwardType":  forwardType,
						"targetID":     target.ID,
						"targetName":   target.Name,
						"relationID":   rel.ID,
					},
				})
			}
		}
	}
}

// checkSchemaViolations finds invalid relationship types (E004)
func (v *Validator) checkSchemaViolations(result *ValidationResult) {
	for _, feature := range v.features {
		fileName := GetFeatureFileName(feature.Name)

		for _, rel := range feature.Relationships {
			// Check if relationship type exists in config
			if _, exists := v.config.Relationships.Types[string(rel.Type)]; !exists {
				// Check if it's an alias
				isAlias := false
				for _, tc := range v.config.Relationships.Types {
					for _, alias := range tc.Aliases {
						if alias == string(rel.Type) {
							isAlias = true
							break
						}
					}
					if isAlias {
						break
					}
				}

				if !isAlias {
					result.Issues = append(result.Issues, ValidationIssue{
						Code:        CodeSchemaViolation,
						Severity:    SeverityError,
						FeatureID:   feature.ID,
						FeatureName: feature.Name,
						FileName:    fileName,
						Message:     fmt.Sprintf("Unknown relationship type: '%s'", rel.Type),
						Fixable:     false,
						Context: map[string]string{
							"relationType": string(rel.Type),
							"relationID":   rel.ID,
						},
					})
				}
			}
		}
	}
}

// checkVersionConstraints validates version constraint satisfaction (E006)
func (v *Validator) checkVersionConstraints(result *ValidationResult) {
	for _, feature := range v.features {
		fileName := GetFeatureFileName(feature.Name)

		for _, rel := range feature.Relationships {
			if rel.VersionConstraint == nil {
				continue
			}

			target := v.featureMap[rel.TargetID]
			if target == nil {
				continue // Already caught by E001
			}

			targetVersion := target.GetCurrentVersionKey()
			if targetVersion == "" {
				continue // No version to check
			}

			if !rel.VersionConstraint.IsSatisfiedBy(targetVersion) {
				result.Issues = append(result.Issues, ValidationIssue{
					Code:        CodeVersionConstraintViolation,
					Severity:    SeverityError,
					FeatureID:   feature.ID,
					FeatureName: feature.Name,
					FileName:    fileName,
					Message: fmt.Sprintf("Version constraint not satisfied: %s %s%s, but '%s' is at v%s",
						rel.Type,
						rel.VersionConstraint.Operator,
						rel.VersionConstraint.GetVersionString(),
						target.Name,
						targetVersion),
					Fixable: false,
					Context: map[string]string{
						"relationType":      string(rel.Type),
						"targetID":          target.ID,
						"targetName":        target.Name,
						"targetVersion":     targetVersion,
						"constraintOp":      rel.VersionConstraint.Operator,
						"constraintVersion": rel.VersionConstraint.GetVersionString(),
					},
				})
			}
		}
	}
}

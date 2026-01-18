package validator

import (
	"context"
	"fmt"

	"github.com/eg3r/fogit/pkg/fogit"
)

// AutoFixer attempts to automatically repair validation issues
type AutoFixer struct {
	repo       fogit.Repository
	config     *fogit.Config
	features   []*fogit.Feature
	featureMap map[string]*fogit.Feature
	dryRun     bool
}

// NewAutoFixer creates a new AutoFixer
func NewAutoFixer(repo fogit.Repository, config *fogit.Config, dryRun bool) *AutoFixer {
	return &AutoFixer{
		repo:   repo,
		config: config,
		dryRun: dryRun,
	}
}

// AttemptFixes tries to fix all fixable issues
func (af *AutoFixer) AttemptFixes(ctx context.Context, issues []ValidationIssue) (*FixResult, error) {
	// Load features
	features, err := af.repo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	af.features = features
	af.featureMap = make(map[string]*fogit.Feature)
	for _, f := range features {
		af.featureMap[f.ID] = f
	}

	result := &FixResult{}

	for _, issue := range issues {
		if !issue.Fixable {
			result.Skipped = append(result.Skipped,
				fmt.Sprintf("[%s] %s: not fixable", issue.Code, issue.FileName))
			continue
		}

		var fixed bool
		var fixErr error

		switch issue.Code {
		case CodeOrphanedRelationship:
			fixed, fixErr = af.fixOrphanedRelationship(ctx, issue)
		case CodeMissingInverse:
			fixed, fixErr = af.fixMissingInverse(ctx, issue)
		case CodeDanglingInverse:
			fixed, fixErr = af.fixDanglingInverse(ctx, issue)
		default:
			result.Skipped = append(result.Skipped,
				fmt.Sprintf("[%s] %s: no fix handler", issue.Code, issue.FileName))
			continue
		}

		if fixErr != nil {
			result.Failed = append(result.Failed,
				fmt.Sprintf("[%s] %s: %v", issue.Code, issue.FileName, fixErr))
		} else if fixed {
			result.Fixed = append(result.Fixed,
				fmt.Sprintf("[%s] %s", issue.Code, issue.FileName))
		}
	}

	return result, nil
}

// fixOrphanedRelationship removes relationships to non-existent targets
func (af *AutoFixer) fixOrphanedRelationship(ctx context.Context, issue ValidationIssue) (bool, error) {
	feature := af.featureMap[issue.FeatureID]
	if feature == nil {
		return false, fmt.Errorf("feature not found: %s", issue.FeatureID)
	}

	targetID := issue.Context["targetID"]
	if targetID == "" {
		return false, fmt.Errorf("missing targetID in issue context")
	}

	// Remove relationships pointing to the missing target
	var newRels []fogit.Relationship
	removed := false
	for _, rel := range feature.Relationships {
		if rel.TargetID != targetID {
			newRels = append(newRels, rel)
		} else {
			removed = true
		}
	}

	if !removed {
		return false, nil
	}

	if af.dryRun {
		return true, nil
	}

	feature.Relationships = newRels
	if err := af.repo.Update(ctx, feature); err != nil {
		return false, fmt.Errorf("failed to update feature: %w", err)
	}

	return true, nil
}

// fixMissingInverse creates the missing inverse relationship
func (af *AutoFixer) fixMissingInverse(ctx context.Context, issue ValidationIssue) (bool, error) {
	sourceFeature := af.featureMap[issue.FeatureID]
	if sourceFeature == nil {
		return false, fmt.Errorf("source feature not found: %s", issue.FeatureID)
	}

	targetID := issue.Context["targetID"]
	inverseType := issue.Context["inverseType"]
	if targetID == "" || inverseType == "" {
		return false, fmt.Errorf("missing context for fix")
	}

	targetFeature := af.featureMap[targetID]
	if targetFeature == nil {
		return false, fmt.Errorf("target feature not found: %s", targetID)
	}

	// Check if inverse already exists (might have been created in a previous fix)
	for _, rel := range targetFeature.Relationships {
		if string(rel.Type) == inverseType && rel.TargetID == sourceFeature.ID {
			return false, nil // Already exists
		}
	}

	// Create inverse relationship
	inverseRel := fogit.NewRelationship(
		fogit.RelationshipType(inverseType),
		sourceFeature.ID,
		sourceFeature.Name,
	)

	if err := targetFeature.AddRelationship(inverseRel); err != nil {
		if err == fogit.ErrDuplicateRelationship {
			return false, nil // Already exists
		}
		return false, fmt.Errorf("failed to add inverse relationship: %w", err)
	}

	if af.dryRun {
		return true, nil
	}

	if err := af.repo.Update(ctx, targetFeature); err != nil {
		return false, fmt.Errorf("failed to update target feature: %w", err)
	}

	return true, nil
}

// fixDanglingInverse removes inverse relationships without forward
func (af *AutoFixer) fixDanglingInverse(ctx context.Context, issue ValidationIssue) (bool, error) {
	feature := af.featureMap[issue.FeatureID]
	if feature == nil {
		return false, fmt.Errorf("feature not found: %s", issue.FeatureID)
	}

	relationID := issue.Context["relationID"]
	if relationID == "" {
		// Fallback: try to find by type and target
		relationType := issue.Context["relationType"]
		targetID := issue.Context["targetID"]
		if relationType == "" || targetID == "" {
			return false, fmt.Errorf("missing context for fix")
		}

		// Remove by type and target
		var newRels []fogit.Relationship
		removed := false
		for _, rel := range feature.Relationships {
			if string(rel.Type) == relationType && rel.TargetID == targetID {
				removed = true
			} else {
				newRels = append(newRels, rel)
			}
		}

		if !removed {
			return false, nil
		}

		if af.dryRun {
			return true, nil
		}

		feature.Relationships = newRels
		if err := af.repo.Update(ctx, feature); err != nil {
			return false, fmt.Errorf("failed to update feature: %w", err)
		}

		return true, nil
	}

	// Remove by relation ID
	var newRels []fogit.Relationship
	removed := false
	for _, rel := range feature.Relationships {
		if rel.ID != relationID {
			newRels = append(newRels, rel)
		} else {
			removed = true
		}
	}

	if !removed {
		return false, nil
	}

	if af.dryRun {
		return true, nil
	}

	feature.Relationships = newRels
	if err := af.repo.Update(ctx, feature); err != nil {
		return false, fmt.Errorf("failed to update feature: %w", err)
	}

	return true, nil
}

// IsDryRun returns true if the fixer is in dry-run mode
func (af *AutoFixer) IsDryRun() bool {
	return af.dryRun
}

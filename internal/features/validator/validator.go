package validator

import (
	"context"
	"fmt"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// Validator performs feature and relationship validation
type Validator struct {
	repo       fogit.Repository
	config     *fogit.Config
	features   []*fogit.Feature
	featureMap map[string]*fogit.Feature
}

// New creates a new Validator
func New(repo fogit.Repository, config *fogit.Config) *Validator {
	return &Validator{
		repo:   repo,
		config: config,
	}
}

// Validate performs all validation checks and returns results
func (v *Validator) Validate(ctx context.Context) (*ValidationResult, error) {
	// Load all features
	features, err := v.repo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}

	v.features = features
	v.featureMap = make(map[string]*fogit.Feature)
	for _, f := range features {
		v.featureMap[f.ID] = f
	}

	result := &ValidationResult{
		FeaturesCount: len(features),
	}

	// Count relationships
	for _, f := range features {
		result.RelCount += len(f.Relationships)
	}

	// Run all checks
	v.checkOrphanedRelationships(result) // E001
	v.checkMissingInverses(result)       // E002
	v.checkDanglingInverses(result)      // E003
	v.checkSchemaViolations(result)      // E004
	v.checkCycles(result)                // E005
	v.checkVersionConstraints(result)    // E006

	// Count by severity
	for _, issue := range result.Issues {
		switch issue.Severity {
		case SeverityError:
			result.Errors++
		case SeverityWarning:
			result.Warnings++
		}
	}

	return result, nil
}

// GetFeatures returns the loaded features (available after Validate is called)
func (v *Validator) GetFeatures() []*fogit.Feature {
	return v.features
}

// GetFeatureMap returns the feature ID map (available after Validate is called)
func (v *Validator) GetFeatureMap() map[string]*fogit.Feature {
	return v.featureMap
}

// GetFeatureFileName returns the YAML filename for a feature
// Uses storage.Slugify to ensure consistency
func GetFeatureFileName(name string) string {
	return storage.Slugify(name, storage.DefaultSlugifyOptions()) + ".yml"
}

// getInverseType returns the inverse relationship type for a given type
func (v *Validator) getInverseType(relType string) string {
	for name, rt := range v.config.Relationships.Types {
		if name == relType && rt.Inverse != "" {
			return rt.Inverse
		}
	}
	return ""
}

// getForwardType returns the forward relationship type for a given inverse type
func (v *Validator) getForwardType(relType string) string {
	for name, rt := range v.config.Relationships.Types {
		if rt.Inverse == relType {
			return name
		}
	}
	return ""
}

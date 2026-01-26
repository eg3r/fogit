package features

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/eg3r/fogit/internal/git"
	"github.com/eg3r/fogit/internal/logger"
	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// ParseVersionConstraint parses a version constraint string into a VersionConstraint struct
// Per spec 06-data-model.md (commit 0a355fc), supports both:
// - Simple versioning: ">=2", ">1", "=3" (integers)
// - Semantic versioning: ">=1.0.0", ">1.1.0", "=2.0.0" (MAJOR.MINOR.PATCH)
func ParseVersionConstraint(constraint string) (*fogit.VersionConstraint, error) {
	if constraint == "" {
		return nil, nil
	}

	// Pattern: operator followed by version (integer or semver)
	// Operators: =, >, <, >=, <=
	// Semver regex: matches x.y.z format
	re := regexp.MustCompile(`^(>=|<=|>|<|=)(.+)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(constraint))
	if matches == nil {
		return nil, fmt.Errorf("invalid version constraint format '%s', expected format like '>=2', '>1', '=3', '>=1.0.0'", constraint)
	}

	operator := matches[1]
	versionStr := strings.TrimSpace(matches[2])

	// Try parsing as integer first (simple versioning)
	if version, err := strconv.Atoi(versionStr); err == nil {
		vc := &fogit.VersionConstraint{
			Operator: operator,
			Version:  version,
		}
		if err := vc.IsValid(); err != nil {
			return nil, err
		}
		return vc, nil
	}

	// Try parsing as semantic version (x.y.z)
	semverRe := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if semverRe.MatchString(versionStr) {
		vc := &fogit.VersionConstraint{
			Operator: operator,
			Version:  versionStr, // Store as string for semver
		}
		if err := vc.IsValid(); err != nil {
			return nil, err
		}
		return vc, nil
	}

	return nil, fmt.Errorf("invalid version '%s', expected positive integer (e.g., 2) or semantic version (e.g., 1.0.0)", versionStr)
}

// RelationshipWithSource wraps a relationship with the source feature ID
type RelationshipWithSource struct {
	SourceID   string
	SourceName string
	Relation   fogit.Relationship
}

// FindIncomingRelationships finds all relationships pointing to the target feature.
// If relType is empty, all relationship types are included.
// This is a convenience wrapper around findIncomingRelationshipsFiltered.
func FindIncomingRelationships(repo fogit.Repository, ctx context.Context, targetID string, relType string) ([]RelationshipWithSource, error) {
	var types []string
	if relType != "" {
		types = []string{relType}
	}
	return findIncomingRelationshipsFiltered(repo, ctx, targetID, types)
}

// FindIncomingRelationshipsMultiType finds all relationships pointing to the target feature,
// filtering by multiple types (empty slice = all types).
// This is a convenience wrapper around findIncomingRelationshipsFiltered.
func FindIncomingRelationshipsMultiType(repo fogit.Repository, ctx context.Context, targetID string, relTypes []string) ([]RelationshipWithSource, error) {
	return findIncomingRelationshipsFiltered(repo, ctx, targetID, relTypes)
}

// findIncomingRelationshipsFiltered is the core implementation for finding incoming relationships.
// It consolidates the duplicate logic from FindIncomingRelationships and FindIncomingRelationshipsMultiType.
func findIncomingRelationshipsFiltered(repo fogit.Repository, ctx context.Context, targetID string, relTypes []string) ([]RelationshipWithSource, error) {
	allFeatures, err := repo.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	var incoming []RelationshipWithSource
	for _, f := range allFeatures {
		for _, rel := range f.Relationships {
			if rel.TargetID == targetID {
				if len(relTypes) == 0 || containsType(relTypes, string(rel.Type)) {
					incoming = append(incoming, RelationshipWithSource{
						SourceID:   f.ID,
						SourceName: f.Name,
						Relation:   rel,
					})
				}
			}
		}
	}
	return incoming, nil
}

// containsType checks if a type is in the list
func containsType(types []string, t string) bool {
	for _, typ := range types {
		if typ == t {
			return true
		}
	}
	return false
}

// LinkOptions contains options for cross-branch linking
type LinkOptions struct {
	GitRepo      *git.Repository // Git repository for cross-branch operations (optional)
	TargetBranch string          // Branch where target feature exists (for inverse relationship)
}

// Link creates a relationship between two features
func Link(ctx context.Context, repo fogit.Repository, source, target *fogit.Feature, relType fogit.RelationshipType, description string, versionConstraint string, cfg *fogit.Config, fogitDir string) (*fogit.Relationship, error) {
	return LinkWithOptions(ctx, repo, source, target, relType, description, versionConstraint, cfg, fogitDir, nil)
}

// LinkWithOptions creates a relationship between two features with cross-branch support
func LinkWithOptions(ctx context.Context, repo fogit.Repository, source, target *fogit.Feature, relType fogit.RelationshipType, description string, versionConstraint string, cfg *fogit.Config, fogitDir string, opts *LinkOptions) (*fogit.Relationship, error) {
	// Parse version constraint if provided
	var vc *fogit.VersionConstraint
	if versionConstraint != "" {
		var err error
		vc, err = ParseVersionConstraint(versionConstraint)
		if err != nil {
			return nil, fmt.Errorf("invalid version constraint: %w", err)
		}
	}

	// Create relationship object using NewRelationship to ensure ID and CreatedAt are set
	rel := fogit.NewRelationship(relType, target.ID, target.Name)
	rel.Description = description
	rel.VersionConstraint = vc

	// Validate relationship against config
	if err := rel.ValidateWithConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid relationship: %w", err)
	}

	// Check for cycles based on category settings
	if err := fogit.DetectCycleWithConfig(ctx, source, &rel, repo, cfg); err != nil {
		return nil, fmt.Errorf("cannot create relationship: %w", err)
	}

	// Add relationship
	if err := source.AddRelationship(rel); err != nil {
		return nil, fmt.Errorf("failed to add relationship: %w", err)
	}

	// Save updated source feature
	if err := repo.Update(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to save feature: %w", err)
	}

	// Auto-create inverse relationship if configured
	if cfg.Relationships.System.AutoCreateInverse {
		typeConfig, exists := cfg.Relationships.Types[string(relType)]
		if exists && typeConfig.Inverse != "" && !typeConfig.Bidirectional {
			inverseRel := fogit.NewRelationship(fogit.RelationshipType(typeConfig.Inverse), source.ID, source.Name)
			inverseRel.Description = description // Copy description to inverse

			if err := target.AddRelationship(inverseRel); err != nil {
				if err != fogit.ErrDuplicateRelationship {
					logger.Warn("failed to create inverse relationship", "error", err, "target", target.Name)
				}
			} else {
				// Try to save target - may need cross-branch save
				saveErr := repo.Update(ctx, target)
				if saveErr != nil {
					// If regular save fails and we have cross-branch options, try cross-branch save
					if opts != nil && opts.GitRepo != nil && opts.TargetBranch != "" {
						if cbErr := saveFeatureOnBranch(opts.GitRepo, target, opts.TargetBranch); cbErr != nil {
							logger.Warn("failed to save inverse relationship on branch", "error", cbErr, "target", target.Name, "branch", opts.TargetBranch)
						} else {
							fmt.Printf("Auto-created inverse relationship: %s -> %s (%s)\n", target.Name, source.Name, typeConfig.Inverse)
						}
					} else {
						logger.Warn("failed to save inverse relationship", "error", saveErr, "target", target.Name)
					}
				} else {
					fmt.Printf("Auto-created inverse relationship: %s -> %s (%s)\n", target.Name, source.Name, typeConfig.Inverse)
				}
			}
		}
	}

	return &rel, nil
}

// saveFeatureOnBranch saves a feature to a specific branch using git cross-branch operations
func saveFeatureOnBranch(gitRepo *git.Repository, feature *fogit.Feature, branch string) error {
	// Serialize feature to YAML
	data, err := storage.MarshalFeature(feature)
	if err != nil {
		return fmt.Errorf("failed to serialize feature: %w", err)
	}

	// Determine file path
	fileName := sanitizeFileName(feature.Name) + ".yml"
	filePath := ".fogit/features/" + fileName

	// Commit message
	commitMsg := fmt.Sprintf("fogit: add inverse relationship to %s", feature.Name)

	// Use git cross-branch write
	return gitRepo.UpdateFileOnBranch(branch, filePath, data, commitMsg)
}

// sanitizeFileName converts a feature name to a safe filename
func sanitizeFileName(name string) string {
	// Convert to lowercase
	result := strings.ToLower(name)
	// Replace spaces with hyphens
	result = strings.ReplaceAll(result, " ", "-")
	// Remove or replace invalid characters
	var cleaned strings.Builder
	for _, r := range result {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			cleaned.WriteRune(r)
		}
	}
	return cleaned.String()
}

// Unlink removes a relationship from a feature
func Unlink(ctx context.Context, repo fogit.Repository, source *fogit.Feature, relID string, fogitDir string, cfg *fogit.Config) (*fogit.Relationship, error) {
	// Find the relationship to display info before removing
	var foundRel *fogit.Relationship
	for _, rel := range source.Relationships {
		if strings.HasPrefix(rel.ID, relID) || rel.ID == relID {
			foundRel = &rel
			break
		}
	}

	if foundRel == nil {
		return nil, fmt.Errorf("relationship not found with ID: %s", relID)
	}

	// Remove by full ID
	if err := source.RemoveRelationshipByID(foundRel.ID); err != nil {
		return nil, fmt.Errorf("failed to remove relationship: %w", err)
	}

	// Save updated source feature
	if err := repo.Update(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to save feature: %w", err)
	}

	return foundRel, nil
}

// ClearAllRelationships removes all outgoing relationships from a feature
// Returns the list of removed relationships for reporting
func ClearAllRelationships(ctx context.Context, repo fogit.Repository, source *fogit.Feature, fogitDir string, cfg *fogit.Config) ([]fogit.Relationship, error) {
	if len(source.Relationships) == 0 {
		return nil, nil // No relationships to clear
	}

	// Build a feature map for looking up target names and inverse cleanup
	allFeatures, err := repo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list features: %w", err)
	}
	featureMap := make(map[string]*fogit.Feature)
	for _, f := range allFeatures {
		featureMap[f.ID] = f
	}

	// Copy relationships before clearing (for reporting)
	removedRels := make([]fogit.Relationship, len(source.Relationships))
	copy(removedRels, source.Relationships)

	// Process each relationship for inverse cleanup
	for i := range removedRels {
		rel := &removedRels[i]
		// Ensure target name is set
		if rel.TargetName == "" {
			if target, ok := featureMap[rel.TargetID]; ok {
				rel.TargetName = target.Name
			}
		}

		// Handle inverse relationship cleanup if auto-create-inverse was used
		target := featureMap[rel.TargetID]
		if cfg.Relationships.System.AutoCreateInverse && target != nil {
			typeConfig, exists := cfg.Relationships.Types[string(rel.Type)]
			if exists && typeConfig.Inverse != "" && !typeConfig.Bidirectional {
				// Try to remove the inverse relationship from target
				if err := target.RemoveRelationship(fogit.RelationshipType(typeConfig.Inverse), source.ID); err == nil {
					// Save target with removed inverse
					if err := repo.Update(ctx, target); err != nil {
						logger.Warn("failed to save inverse removal", "error", err, "target", target.Name)
					}
				}
			}
		}
	}

	// Clear all relationships from source
	source.Relationships = nil

	// Save updated source feature
	if err := repo.Update(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to save feature: %w", err)
	}

	return removedRels, nil
}

// CleanupIncomingRelationships removes all relationships from other features that point to the deleted feature
// Returns the number of relationships removed
func CleanupIncomingRelationships(ctx context.Context, repo fogit.Repository, deletedFeatureID string) (int, error) {
	allFeatures, err := repo.List(ctx, nil)
	if err != nil {
		return 0, err
	}

	removedCount := 0
	for _, f := range allFeatures {
		if f.ID == deletedFeatureID {
			continue
		}

		modified := false
		var remaining []fogit.Relationship
		for _, rel := range f.Relationships {
			if rel.TargetID != deletedFeatureID {
				remaining = append(remaining, rel)
			} else {
				modified = true
				removedCount++
			}
		}

		if modified {
			f.Relationships = remaining
			if err := repo.Update(ctx, f); err != nil {
				return removedCount, fmt.Errorf("failed to update %s: %w", f.Name, err)
			}
		}
	}

	return removedCount, nil
}

// UnlinkByTarget removes a relationship by target feature and optional type
func UnlinkByTarget(ctx context.Context, repo fogit.Repository, source, target *fogit.Feature, relType fogit.RelationshipType, fogitDir string, cfg *fogit.Config) (*fogit.Relationship, error) {
	// If no type specified, find first matching relationship
	if relType == "" {
		found := false
		for _, rel := range source.Relationships {
			if rel.TargetID == target.ID {
				relType = rel.Type
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("no relationship found between %s and %s", source.Name, target.Name)
		}
	}

	// Remove relationship from source
	if err := source.RemoveRelationship(relType, target.ID); err != nil {
		if err == fogit.ErrRelationshipNotFound {
			return nil, fmt.Errorf("relationship not found: %s -> %s (%s)", source.Name, target.Name, relType)
		}
		return nil, fmt.Errorf("failed to remove relationship: %w", err)
	}

	// Reconstruct removed relationship for return
	removedRel := &fogit.Relationship{
		Type:       relType,
		TargetID:   target.ID,
		TargetName: target.Name,
	}

	// Save updated source feature
	if err := repo.Update(ctx, source); err != nil {
		return nil, fmt.Errorf("failed to save feature: %w", err)
	}

	return removedRel, nil
}

package features

import (
	"context"
	"fmt"

	"github.com/eg3r/fogit/pkg/fogit"
)

// DeleteResult contains the result of a delete operation
type DeleteResult struct {
	Feature                *fogit.Feature `json:"feature" yaml:"feature"`
	CleanedUpRelationships int            `json:"cleaned_up_relationships" yaml:"cleaned_up_relationships"`
}

// DeleteOptions configures the delete operation
type DeleteOptions struct {
	// SkipRelationshipCleanup skips cleaning up incoming relationships from other features
	SkipRelationshipCleanup bool
}

// Delete removes a feature from the repository and cleans up all incoming relationships.
// This is a complete delete operation that:
// 1. Finds all features that have relationships pointing to this feature
// 2. Removes those incoming relationships
// 3. Deletes the feature from the repository
func Delete(ctx context.Context, repo fogit.Repository, feature *fogit.Feature, opts DeleteOptions) (*DeleteResult, error) {
	result := &DeleteResult{
		Feature: feature,
	}

	// Clean up incoming relationships unless skipped
	if !opts.SkipRelationshipCleanup {
		removedCount, err := CleanupIncomingRelationships(ctx, repo, feature.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to cleanup incoming relationships: %w", err)
		}
		result.CleanedUpRelationships = removedCount
	}

	// Delete the feature from the repository
	if err := repo.Delete(ctx, feature.ID); err != nil {
		return nil, fmt.Errorf("failed to delete feature: %w", err)
	}

	return result, nil
}

// GetIncomingRelationshipSummary returns information about features that have relationships
// pointing to the target feature. This is useful for confirmation prompts before deletion.
type IncomingRelationshipSummary struct {
	SourceID   string `json:"source_id" yaml:"source_id"`
	SourceName string `json:"source_name" yaml:"source_name"`
	Type       string `json:"type" yaml:"type"`
}

// GetIncomingRelationshipSummary returns a summary of all incoming relationships for a feature
func GetIncomingRelationshipSummary(ctx context.Context, repo fogit.Repository, featureID string) ([]IncomingRelationshipSummary, error) {
	incoming, err := FindIncomingRelationships(repo, ctx, featureID, "")
	if err != nil {
		return nil, err
	}

	summaries := make([]IncomingRelationshipSummary, len(incoming))
	for i, rel := range incoming {
		summaries[i] = IncomingRelationshipSummary{
			SourceID:   rel.SourceID,
			SourceName: rel.SourceName,
			Type:       string(rel.Relation.Type),
		}
	}

	return summaries, nil
}

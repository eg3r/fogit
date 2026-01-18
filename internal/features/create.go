package features

import (
	"context"
	"fmt"
	"time"

	"github.com/eg3r/fogit/pkg/fogit"
)

type CreateOptions struct {
	Name        string
	Description string
	Type        string
	Priority    string
	Category    string
	Domain      string
	Team        string
	Epic        string
	Module      string
	Tags        []string
	Metadata    map[string]interface{}
	ParentID    string

	// Git options
	SameBranch    bool
	IsolateBranch bool
}

func Create(ctx context.Context, repo fogit.Repository, opts CreateOptions, cfg *fogit.Config, fogitDir string) (*fogit.Feature, error) {
	// Create feature object
	feature := fogit.NewFeature(opts.Name)
	feature.Description = opts.Description
	feature.Tags = opts.Tags

	// Set organization fields via metadata accessors (per spec 06-data-model.md)
	if opts.Type != "" {
		feature.SetType(opts.Type)
	}
	if opts.Priority != "" {
		feature.SetPriority(fogit.Priority(opts.Priority))
	}
	if opts.Category != "" {
		feature.SetCategory(opts.Category)
	}
	if opts.Domain != "" {
		feature.SetDomain(opts.Domain)
	}
	if opts.Team != "" {
		feature.SetTeam(opts.Team)
	}
	if opts.Epic != "" {
		feature.SetEpic(opts.Epic)
	}
	if opts.Module != "" {
		feature.SetModule(opts.Module)
	}

	// Add any additional metadata
	if len(opts.Metadata) > 0 {
		for k, v := range opts.Metadata {
			feature.SetMetadata(k, v)
		}
	}

	// Handle parent relationship (if specified) - create contained-by relationship
	if opts.ParentID != "" {
		// Per spec: hierarchy is expressed via contains/contained-by relationships
		rel := fogit.Relationship{
			ID:        fmt.Sprintf("rel-%s", feature.ID[:8]),
			Type:      "contained-by",
			TargetID:  opts.ParentID,
			CreatedAt: time.Now().UTC(),
		}
		if err := feature.AddRelationship(rel); err != nil {
			return nil, fmt.Errorf("failed to add parent relationship: %w", err)
		}
	}

	// Validate feature
	if err := feature.Validate(); err != nil {
		return nil, fmt.Errorf("invalid feature: %w", err)
	}

	// Handle Git branch creation
	if err := HandleBranchCreation(opts.Name, cfg, opts.SameBranch, opts.IsolateBranch); err != nil {
		return nil, err
	}

	// Create feature in repository
	if err := repo.Create(ctx, feature); err != nil {
		return nil, fmt.Errorf("failed to create feature: %w", err)
	}

	return feature, nil
}

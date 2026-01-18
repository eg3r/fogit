package features

import (
	"context"
	"fmt"

	"github.com/eg3r/fogit/pkg/fogit"
)

type UpdateOptions struct {
	Name        *string
	Description *string
	State       *string
	Priority    *string
	Type        *string
	Category    *string
	Domain      *string
	Team        *string
	Epic        *string
	Module      *string
	Metadata    map[string]interface{}
}

func Update(ctx context.Context, repo fogit.Repository, feature *fogit.Feature, opts UpdateOptions) (bool, error) {
	changed := false

	if opts.Name != nil && *opts.Name != "" && *opts.Name != feature.Name {
		feature.Name = *opts.Name
		changed = true
	}

	if opts.Description != nil && *opts.Description != feature.Description {
		feature.Description = *opts.Description
		changed = true
	}

	if opts.State != nil && *opts.State != "" {
		newState := fogit.State(*opts.State)
		oldState := feature.DeriveState()
		if newState != oldState {
			if err := feature.UpdateState(newState); err != nil {
				return false, fmt.Errorf("failed to update state: %w", err)
			}
			// UpdateState already handles per-version timestamps
			changed = true
		}
	}

	// Use setter methods for organization fields (per spec 06-data-model.md)
	if opts.Priority != nil && *opts.Priority != "" {
		newPriority := fogit.Priority(*opts.Priority)
		if newPriority != feature.GetPriority() {
			if !newPriority.IsValid() {
				return false, fogit.ErrInvalidPriority
			}
			feature.SetPriority(newPriority)
			changed = true
		}
	}

	if opts.Type != nil && *opts.Type != feature.GetType() {
		feature.SetType(*opts.Type)
		changed = true
	}

	if opts.Category != nil && *opts.Category != feature.GetCategory() {
		feature.SetCategory(*opts.Category)
		changed = true
	}

	if opts.Domain != nil && *opts.Domain != feature.GetDomain() {
		feature.SetDomain(*opts.Domain)
		changed = true
	}

	if opts.Team != nil && *opts.Team != feature.GetTeam() {
		feature.SetTeam(*opts.Team)
		changed = true
	}

	if opts.Epic != nil && *opts.Epic != feature.GetEpic() {
		feature.SetEpic(*opts.Epic)
		changed = true
	}

	if opts.Module != nil && *opts.Module != feature.GetModule() {
		feature.SetModule(*opts.Module)
		changed = true
	}

	if len(opts.Metadata) > 0 {
		for k, v := range opts.Metadata {
			feature.SetMetadata(k, v)
		}
		changed = true
	}

	if changed {
		feature.UpdateModifiedAt()

		// Validate updated feature
		if err := feature.Validate(); err != nil {
			return false, fmt.Errorf("invalid feature after update: %w", err)
		}

		if err := repo.Update(ctx, feature); err != nil {
			return false, fmt.Errorf("failed to save feature: %w", err)
		}
	}

	return changed, nil
}

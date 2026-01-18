package features

import (
	"context"
	"fmt"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

func Reopen(ctx context.Context, repo fogit.Repository, feature *fogit.Feature, newVersion string, currentVersionStr string) (string, error) {
	// Slugify name for branch
	opts := storage.SlugifyOptions{
		MaxLength:        50,
		AllowSlashes:     false,
		NormalizeUnicode: true,
		EmptyFallback:    "unnamed",
	}
	slug := storage.Slugify(feature.Name, opts)
	branch := fmt.Sprintf("feature/%s-v%s", slug, newVersion)

	// Reopen feature with new version
	notes := fmt.Sprintf("Reopened from version %s", currentVersionStr)
	if err := feature.ReopenFeature(currentVersionStr, newVersion, branch, notes); err != nil {
		return "", fmt.Errorf("failed to reopen feature: %w", err)
	}

	// Update in repository
	if err := repo.Update(ctx, feature); err != nil {
		return "", fmt.Errorf("failed to save feature: %w", err)
	}

	return branch, nil
}

package fogit

import "context"

// Repository defines the interface for feature storage operations
type Repository interface {
	// Create creates a new feature in the repository
	Create(ctx context.Context, feature *Feature) error

	// Get retrieves a feature by ID
	Get(ctx context.Context, id string) (*Feature, error)

	// List retrieves features matching the given filter
	List(ctx context.Context, filter *Filter) ([]*Feature, error)

	// Update updates an existing feature
	Update(ctx context.Context, feature *Feature) error

	// Delete removes a feature from the repository
	Delete(ctx context.Context, id string) error
}

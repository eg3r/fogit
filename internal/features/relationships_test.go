package features

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

func TestParseVersionConstraint(t *testing.T) {
	tests := []struct {
		name         string
		constraint   string
		wantOperator string
		wantVersion  interface{}
		wantIsSemver bool
		wantIsSimple bool
		wantError    bool
	}{
		// Simple versioning (integers)
		{"simple >=2", ">=2", ">=", 2, false, true, false},
		{"simple >1", ">1", ">", 1, false, true, false},
		{"simple <3", "<3", "<", 3, false, true, false},
		{"simple <=4", "<=4", "<=", 4, false, true, false},
		{"simple =5", "=5", "=", 5, false, true, false},
		{"simple with spaces", "  >= 10  ", ">=", 10, false, true, false},

		// Semantic versioning
		{"semver >=1.0.0", ">=1.0.0", ">=", "1.0.0", true, false, false},
		{"semver >1.2.3", ">1.2.3", ">", "1.2.3", true, false, false},
		{"semver <2.0.0", "<2.0.0", "<", "2.0.0", true, false, false},
		{"semver <=1.5.0", "<=1.5.0", "<=", "1.5.0", true, false, false},
		{"semver =1.0.0", "=1.0.0", "=", "1.0.0", true, false, false},
		{"semver with spaces", "  >= 1.2.3  ", ">=", "1.2.3", true, false, false},

		// Empty constraint returns nil
		{"empty", "", "", nil, false, false, false},

		// Invalid constraints
		{"invalid operator", "!=2", "", nil, false, false, true},
		{"invalid format no operator", "2", "", nil, false, false, true},
		{"invalid semver format", ">=1.0", "", nil, false, false, true},
		{"invalid version 0", ">=0", "", nil, false, false, true},
		{"invalid version negative", ">=-1", "", nil, false, false, true},
		{"invalid version string", ">=abc", "", nil, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersionConstraint(tt.constraint)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseVersionConstraint() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return // Error expected, skip other checks
			}

			if tt.constraint == "" {
				if got != nil {
					t.Errorf("ParseVersionConstraint() = %v, want nil for empty constraint", got)
				}
				return
			}

			if got == nil {
				t.Fatal("ParseVersionConstraint() returned nil, want non-nil")
			}

			if got.Operator != tt.wantOperator {
				t.Errorf("ParseVersionConstraint().Operator = %v, want %v", got.Operator, tt.wantOperator)
			}

			if tt.wantIsSemver {
				if !got.IsSemanticVersion() {
					t.Errorf("ParseVersionConstraint().IsSemanticVersion() = false, want true")
				}
				if got.GetSemanticVersion() != tt.wantVersion {
					t.Errorf("ParseVersionConstraint().GetSemanticVersion() = %v, want %v", got.GetSemanticVersion(), tt.wantVersion)
				}
			}

			if tt.wantIsSimple {
				if !got.IsSimpleVersion() {
					t.Errorf("ParseVersionConstraint().IsSimpleVersion() = false, want true")
				}
				if got.GetSimpleVersion() != tt.wantVersion {
					t.Errorf("ParseVersionConstraint().GetSimpleVersion() = %v, want %v", got.GetSimpleVersion(), tt.wantVersion)
				}
			}
		})
	}
}

func TestContainsType(t *testing.T) {
	tests := []struct {
		name     string
		types    []string
		t        string
		expected bool
	}{
		{"empty types matches nothing", []string{}, "depends-on", false},
		{"single match", []string{"depends-on"}, "depends-on", true},
		{"single no match", []string{"implements"}, "depends-on", false},
		{"multiple match first", []string{"depends-on", "implements"}, "depends-on", true},
		{"multiple match second", []string{"depends-on", "implements"}, "implements", true},
		{"multiple no match", []string{"depends-on", "implements"}, "blocks", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsType(tt.types, tt.t); got != tt.expected {
				t.Errorf("containsType(%v, %s) = %v, want %v", tt.types, tt.t, got, tt.expected)
			}
		})
	}
}

func TestParseVersionConstraint_Integration(t *testing.T) {
	// Test that parsed constraints work correctly with IsSatisfiedBy

	tests := []struct {
		name          string
		constraint    string
		targetVersion string
		expected      bool
	}{
		// Simple constraints
		{">=2 satisfied by 3", ">=2", "3", true},
		{">=2 not satisfied by 1", ">=2", "1", false},
		{">1 satisfied by 2", ">1", "2", true},
		{">1 not satisfied by 1", ">1", "1", false},

		// Semver constraints
		{">=1.0.0 satisfied by 1.0.0", ">=1.0.0", "1.0.0", true},
		{">=1.0.0 satisfied by 1.1.0", ">=1.0.0", "1.1.0", true},
		{">=1.0.0 not satisfied by 0.9.9", ">=1.0.0", "0.9.9", false},
		{">1.0.0 satisfied by 1.0.1", ">1.0.0", "1.0.1", true},
		{">1.0.0 not satisfied by 1.0.0", ">1.0.0", "1.0.0", false},

		// Cross-version type constraints
		{">=2 against semver 2.0.0", ">=2", "2.0.0", true},
		{">=1.0.0 against simple 2", ">=1.0.0", "2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc, err := ParseVersionConstraint(tt.constraint)
			if err != nil {
				t.Fatalf("ParseVersionConstraint(%s) error = %v", tt.constraint, err)
			}

			if got := vc.IsSatisfiedBy(tt.targetVersion); got != tt.expected {
				t.Errorf("IsSatisfiedBy(%s) = %v, want %v", tt.targetVersion, got, tt.expected)
			}
		})
	}
}

// mockRepository implements fogit.Repository for testing
type mockRepository struct {
	features map[string]*fogit.Feature
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		features: make(map[string]*fogit.Feature),
	}
}

func (r *mockRepository) Create(ctx context.Context, feature *fogit.Feature) error {
	r.features[feature.ID] = feature
	return nil
}

func (r *mockRepository) Get(ctx context.Context, id string) (*fogit.Feature, error) {
	if f, ok := r.features[id]; ok {
		return f, nil
	}
	return nil, fogit.ErrNotFound
}

func (r *mockRepository) Update(ctx context.Context, feature *fogit.Feature) error {
	r.features[feature.ID] = feature
	return nil
}

func (r *mockRepository) Delete(ctx context.Context, id string) error {
	delete(r.features, id)
	return nil
}

func (r *mockRepository) List(ctx context.Context, filter *fogit.Filter) ([]*fogit.Feature, error) {
	var result []*fogit.Feature
	for _, f := range r.features {
		result = append(result, f)
	}
	return result, nil
}

func (r *mockRepository) GetByName(ctx context.Context, name string) (*fogit.Feature, error) {
	for _, f := range r.features {
		if f.Name == name {
			return f, nil
		}
	}
	return nil, fogit.ErrNotFound
}

// TestLink_ReturnsProperIDAndTimestamp verifies that Link() creates relationships with
// proper ID and CreatedAt values (regression test for bug where struct literal was used
// instead of NewRelationship, resulting in empty ID and zero timestamp)
func TestLink_ReturnsProperIDAndTimestamp(t *testing.T) {
	// Setup: create temp directory and repository
	tempDir := t.TempDir()
	fogitDir := filepath.Join(tempDir, ".fogit")
	if err := os.MkdirAll(filepath.Join(fogitDir, "features"), 0755); err != nil {
		t.Fatalf("Failed to create fogit directory: %v", err)
	}

	repo := storage.NewFileRepository(fogitDir)

	ctx := context.Background()
	cfg := fogit.DefaultConfig()

	// Create source and target features
	source := fogit.NewFeature("Source Feature")
	target := fogit.NewFeature("Target Feature")
	if err := repo.Create(ctx, source); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	if err := repo.Create(ctx, target); err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	// Create relationship using Link
	rel, err := Link(ctx, repo, source, target, "related-to", "Test relationship", "", cfg, fogitDir)
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	// CRITICAL: Verify ID is set (was empty before fix)
	if rel.ID == "" {
		t.Error("Link() returned relationship with empty ID - this is a regression!")
	}

	// CRITICAL: Verify CreatedAt is set (was zero before fix)
	if rel.CreatedAt.IsZero() {
		t.Error("Link() returned relationship with zero CreatedAt - this is a regression!")
	}

	// Verify ID looks like a UUID (36 chars with dashes)
	if len(rel.ID) != 36 {
		t.Errorf("Link() returned relationship with invalid ID length: got %d, want 36", len(rel.ID))
	}

	// Verify the relationship was properly stored in the feature
	source, err = repo.Get(ctx, source.ID)
	if err != nil {
		t.Fatalf("Failed to reload source: %v", err)
	}

	if len(source.Relationships) == 0 {
		t.Fatal("No relationships stored on source feature")
	}

	storedRel := source.Relationships[0]
	if storedRel.ID == "" {
		t.Error("Stored relationship has empty ID")
	}
	if storedRel.CreatedAt.IsZero() {
		t.Error("Stored relationship has zero CreatedAt")
	}
	if storedRel.ID != rel.ID {
		t.Errorf("Stored relationship ID mismatch: got %s, want %s", storedRel.ID, rel.ID)
	}
}

// TestFindIncomingRelationships tests the basic function that finds incoming relationships
func TestFindIncomingRelationships(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	// Create target feature (will be deleted)
	targetFeature := fogit.NewFeature("Database")
	repo.Create(ctx, targetFeature)

	// Create source feature with relationship TO target
	sourceFeature := fogit.NewFeature("API Layer")
	sourceFeature.Relationships = []fogit.Relationship{
		{
			ID:         "rel-1",
			Type:       "depends-on",
			TargetID:   targetFeature.ID,
			TargetName: targetFeature.Name,
		},
	}
	repo.Create(ctx, sourceFeature)

	// Create another source feature with relationship TO target
	anotherSource := fogit.NewFeature("Web UI")
	anotherSource.Relationships = []fogit.Relationship{
		{
			ID:         "rel-2",
			Type:       "depends-on",
			TargetID:   targetFeature.ID,
			TargetName: targetFeature.Name,
		},
	}
	repo.Create(ctx, anotherSource)

	// Create unrelated feature
	unrelated := fogit.NewFeature("Unrelated Feature")
	repo.Create(ctx, unrelated)

	// Find incoming relationships
	incoming, err := FindIncomingRelationships(repo, ctx, targetFeature.ID, "")
	if err != nil {
		t.Fatalf("FindIncomingRelationships() error = %v", err)
	}

	// Should find 2 incoming relationships
	if len(incoming) != 2 {
		t.Errorf("expected 2 incoming relationships, got %d", len(incoming))
	}

	// Verify source names
	sourceNames := make(map[string]bool)
	for _, ir := range incoming {
		sourceNames[ir.SourceName] = true
	}

	if !sourceNames["API Layer"] {
		t.Error("expected incoming relationship from 'API Layer'")
	}
	if !sourceNames["Web UI"] {
		t.Error("expected incoming relationship from 'Web UI'")
	}
}

// TestCleanupIncomingRelationships tests cleanup of relationships when a feature is deleted
func TestCleanupIncomingRelationships(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fogit-cleanup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fogitDir := filepath.Join(tempDir, ".fogit")
	os.MkdirAll(filepath.Join(fogitDir, "features"), 0755)

	repo := storage.NewFileRepository(fogitDir)
	ctx := context.Background()

	// Create target feature (will be deleted)
	targetFeature := fogit.NewFeature("Database")
	if err := repo.Create(ctx, targetFeature); err != nil {
		t.Fatalf("failed to create target feature: %v", err)
	}

	// Create source feature with multiple relationships (one to target, one to something else)
	sourceFeature := fogit.NewFeature("API Layer")
	sourceFeature.Relationships = []fogit.Relationship{
		{
			ID:         "rel-1",
			Type:       "depends-on",
			TargetID:   targetFeature.ID,
			TargetName: targetFeature.Name,
		},
		{
			ID:         "rel-2",
			Type:       "relates-to",
			TargetID:   "some-other-id",
			TargetName: "Other Feature",
		},
	}
	if err := repo.Create(ctx, sourceFeature); err != nil {
		t.Fatalf("failed to create source feature: %v", err)
	}

	// Cleanup incoming relationships
	removedCount, err := CleanupIncomingRelationships(ctx, repo, targetFeature.ID)
	if err != nil {
		t.Fatalf("CleanupIncomingRelationships() error = %v", err)
	}

	if removedCount != 1 {
		t.Errorf("expected 1 removed relationship, got %d", removedCount)
	}

	// Verify source feature still has the other relationship but not the one to target
	updatedSource, err := repo.Get(ctx, sourceFeature.ID)
	if err != nil {
		t.Fatalf("failed to get updated source feature: %v", err)
	}

	if len(updatedSource.Relationships) != 1 {
		t.Errorf("expected 1 remaining relationship, got %d", len(updatedSource.Relationships))
	}

	if updatedSource.Relationships[0].TargetID == targetFeature.ID {
		t.Error("relationship to deleted feature should have been removed")
	}

	if updatedSource.Relationships[0].ID != "rel-2" {
		t.Errorf("expected rel-2 to remain, got %s", updatedSource.Relationships[0].ID)
	}
}

// TestCleanupIncomingRelationships_NoRelationships tests cleanup when there are no incoming relationships
func TestCleanupIncomingRelationships_NoRelationships(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fogit-cleanup-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fogitDir := filepath.Join(tempDir, ".fogit")
	os.MkdirAll(filepath.Join(fogitDir, "features"), 0755)

	repo := storage.NewFileRepository(fogitDir)
	ctx := context.Background()

	// Create target feature with no incoming relationships
	targetFeature := fogit.NewFeature("Standalone Feature")
	if err := repo.Create(ctx, targetFeature); err != nil {
		t.Fatalf("failed to create target feature: %v", err)
	}

	// Create another feature with no relationship to target
	otherFeature := fogit.NewFeature("Other Feature")
	otherFeature.Relationships = []fogit.Relationship{
		{
			ID:         "rel-1",
			Type:       "depends-on",
			TargetID:   "different-id",
			TargetName: "Different Feature",
		},
	}
	if err := repo.Create(ctx, otherFeature); err != nil {
		t.Fatalf("failed to create other feature: %v", err)
	}

	// Cleanup should find nothing to remove
	removedCount, err := CleanupIncomingRelationships(ctx, repo, targetFeature.ID)
	if err != nil {
		t.Fatalf("CleanupIncomingRelationships() error = %v", err)
	}

	if removedCount != 0 {
		t.Errorf("expected 0 removed relationships, got %d", removedCount)
	}

	// Other feature's relationships should be unchanged
	updatedOther, err := repo.Get(ctx, otherFeature.ID)
	if err != nil {
		t.Fatalf("failed to get other feature: %v", err)
	}

	if len(updatedOther.Relationships) != 1 {
		t.Errorf("other feature should still have 1 relationship, got %d", len(updatedOther.Relationships))
	}
}

func TestFindIncomingRelationshipsMultiType(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	// Create test features
	featureA := fogit.NewFeature("Feature A")
	featureB := fogit.NewFeature("Feature B")
	featureC := fogit.NewFeature("Feature C")

	// B depends-on A
	featureB.Relationships = []fogit.Relationship{
		{Type: "depends-on", TargetID: featureA.ID, TargetName: featureA.Name},
	}

	// C implements A and depends-on A
	featureC.Relationships = []fogit.Relationship{
		{Type: "implements", TargetID: featureA.ID, TargetName: featureA.Name},
		{Type: "depends-on", TargetID: featureA.ID, TargetName: featureA.Name},
	}

	repo.Create(ctx, featureA)
	repo.Create(ctx, featureB)
	repo.Create(ctx, featureC)

	tests := []struct {
		name      string
		targetID  string
		types     []string
		wantCount int
	}{
		{"all types", featureA.ID, nil, 3},
		{"empty slice all types", featureA.ID, []string{}, 3},
		{"depends-on only", featureA.ID, []string{"depends-on"}, 2},
		{"implements only", featureA.ID, []string{"implements"}, 1},
		{"both types", featureA.ID, []string{"depends-on", "implements"}, 3},
		{"non-existent type", featureA.ID, []string{"blocks"}, 0},
		{"no incoming", featureB.ID, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incoming, err := FindIncomingRelationshipsMultiType(repo, ctx, tt.targetID, tt.types)
			if err != nil {
				t.Fatalf("FindIncomingRelationshipsMultiType() error = %v", err)
			}
			if len(incoming) != tt.wantCount {
				t.Errorf("FindIncomingRelationshipsMultiType() returned %d, want %d", len(incoming), tt.wantCount)
			}
		})
	}
}

func TestClearAllRelationships(t *testing.T) {
	ctx := context.Background()

	// Create a temp directory for fogit
	tempDir, err := os.MkdirTemp("", "fogit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fogitDir := filepath.Join(tempDir, ".fogit")
	os.MkdirAll(fogitDir, 0755)

	// Create default config
	cfg := fogit.DefaultConfig()

	t.Run("clear relationships from feature", func(t *testing.T) {
		repo := newMockRepository()

		// Create test features
		featureA := fogit.NewFeature("Feature A")
		featureB := fogit.NewFeature("Feature B")
		featureC := fogit.NewFeature("Feature C")

		// A has relationships to B and C
		featureA.Relationships = []fogit.Relationship{
			{Type: "depends-on", TargetID: featureB.ID, TargetName: featureB.Name},
			{Type: "implements", TargetID: featureC.ID, TargetName: featureC.Name},
		}

		repo.Create(ctx, featureA)
		repo.Create(ctx, featureB)
		repo.Create(ctx, featureC)

		// Clear relationships
		removed, err := ClearAllRelationships(ctx, repo, featureA, fogitDir, cfg)
		if err != nil {
			t.Fatalf("ClearAllRelationships() error = %v", err)
		}

		// Check removed count
		if len(removed) != 2 {
			t.Errorf("ClearAllRelationships() removed %d, want 2", len(removed))
		}

		// Check feature has no relationships
		if len(featureA.Relationships) != 0 {
			t.Errorf("Feature still has %d relationships, want 0", len(featureA.Relationships))
		}

		// Verify in repo
		updated, _ := repo.Get(ctx, featureA.ID)
		if len(updated.Relationships) != 0 {
			t.Errorf("Repo feature still has %d relationships, want 0", len(updated.Relationships))
		}
	})

	t.Run("clear empty relationships", func(t *testing.T) {
		repo := newMockRepository()

		featureA := fogit.NewFeature("Feature A")
		featureA.Relationships = nil

		repo.Create(ctx, featureA)

		removed, err := ClearAllRelationships(ctx, repo, featureA, fogitDir, cfg)
		if err != nil {
			t.Fatalf("ClearAllRelationships() error = %v", err)
		}

		if removed != nil {
			t.Errorf("ClearAllRelationships() returned %v, want nil for empty", removed)
		}
	})

	t.Run("removed relationships have target names", func(t *testing.T) {
		repo := newMockRepository()

		featureA := fogit.NewFeature("Feature A")
		featureB := fogit.NewFeature("Feature B")

		// Relationship without target name (will be looked up)
		featureA.Relationships = []fogit.Relationship{
			{Type: "depends-on", TargetID: featureB.ID, TargetName: ""},
		}

		repo.Create(ctx, featureA)
		repo.Create(ctx, featureB)

		removed, err := ClearAllRelationships(ctx, repo, featureA, fogitDir, cfg)
		if err != nil {
			t.Fatalf("ClearAllRelationships() error = %v", err)
		}

		if len(removed) != 1 {
			t.Fatalf("ClearAllRelationships() removed %d, want 1", len(removed))
		}

		// Check that target name was filled in
		if removed[0].TargetName != featureB.Name {
			t.Errorf("Removed relationship TargetName = %q, want %q", removed[0].TargetName, featureB.Name)
		}
	})
}

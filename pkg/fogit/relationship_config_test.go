package fogit

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestNewRelationship verifies that the factory function properly initializes all fields
func TestNewRelationship(t *testing.T) {
	beforeCreate := time.Now().UTC().Add(-time.Second) // Allow for clock skew

	rel := NewRelationship("depends-on", "target-123", "Target Feature")

	afterCreate := time.Now().UTC().Add(time.Second)

	// Verify ID is set and looks like a UUID
	if rel.ID == "" {
		t.Error("NewRelationship() returned empty ID")
	}
	if len(rel.ID) != 36 {
		t.Errorf("NewRelationship() ID length = %d, want 36 (UUID format)", len(rel.ID))
	}

	// Verify CreatedAt is set and reasonable
	if rel.CreatedAt.IsZero() {
		t.Error("NewRelationship() returned zero CreatedAt")
	}
	if rel.CreatedAt.Before(beforeCreate) || rel.CreatedAt.After(afterCreate) {
		t.Errorf("NewRelationship() CreatedAt = %v, want between %v and %v",
			rel.CreatedAt, beforeCreate, afterCreate)
	}

	// Verify other fields are set correctly
	if rel.Type != "depends-on" {
		t.Errorf("NewRelationship() Type = %v, want depends-on", rel.Type)
	}
	if rel.TargetID != "target-123" {
		t.Errorf("NewRelationship() TargetID = %v, want target-123", rel.TargetID)
	}
	if rel.TargetName != "Target Feature" {
		t.Errorf("NewRelationship() TargetName = %v, want Target Feature", rel.TargetName)
	}

	// Verify optional fields are zero/nil
	if rel.Description != "" {
		t.Errorf("NewRelationship() Description = %v, want empty", rel.Description)
	}
	if rel.VersionConstraint != nil {
		t.Errorf("NewRelationship() VersionConstraint = %v, want nil", rel.VersionConstraint)
	}
}

// TestNewRelationship_UniqueIDs verifies each call generates a unique ID
func TestNewRelationship_UniqueIDs(t *testing.T) {
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		rel := NewRelationship("depends-on", "target", "Target")
		if ids[rel.ID] {
			t.Errorf("NewRelationship() generated duplicate ID: %s", rel.ID)
		}
		ids[rel.ID] = true
	}
}

func TestRelationship_ValidateWithConfig(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name    string
		rel     Relationship
		wantErr string
	}{
		{
			name: "valid depends-on",
			rel: Relationship{
				Type:     "depends-on",
				TargetID: "target-123",
			},
			wantErr: "",
		},
		{
			name: "valid with alias",
			rel: Relationship{
				Type:     "requires", // alias for depends-on
				TargetID: "target-123",
			},
			wantErr: "",
		},
		{
			name: "unknown type",
			rel: Relationship{
				Type:     "unknown-type",
				TargetID: "target-123",
			},
			wantErr: "not defined in config",
		},
		{
			name: "empty target ID",
			rel: Relationship{
				Type:     "depends-on",
				TargetID: "",
			},
			wantErr: "target ID cannot be empty",
		},
		{
			name: "valid workflow type",
			rel: Relationship{
				Type:     "blocks",
				TargetID: "target-123",
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rel.ValidateWithConfig(cfg)

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateWithConfig() unexpected error = %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateWithConfig() expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateWithConfig() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestRelationship_ValidateWithConfig_AliasResolution(t *testing.T) {
	cfg := DefaultConfig()

	rel := Relationship{
		Type:     "requires", // alias for depends-on
		TargetID: "target-123",
	}

	err := rel.ValidateWithConfig(cfg)
	if err != nil {
		t.Fatalf("ValidateWithConfig() error = %v", err)
	}

	// Check that the type was updated to canonical name
	if rel.Type != "depends-on" {
		t.Errorf("Expected type to be updated to 'depends-on', got %q", rel.Type)
	}
}

func TestRelationship_GetCategory(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name    string
		relType RelationshipType
		wantCat string
	}{
		{"depends-on is structural", "depends-on", "structural"},
		{"blocks is workflow", "blocks", "workflow"},
		{"related-to is informational", "related-to", "informational"},
		{"references is informational", "references", "informational"},
		{"contains is structural", "contains", "structural"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel := Relationship{Type: tt.relType}
			cat := rel.GetCategory(cfg)
			if cat != tt.wantCat {
				t.Errorf("GetCategory() = %q, want %q", cat, tt.wantCat)
			}
		})
	}
}

func TestRelationship_GetCategory_UnknownType(t *testing.T) {
	cfg := DefaultConfig()

	rel := Relationship{Type: "unknown-type"}
	cat := rel.GetCategory(cfg)

	// Should return default category
	if cat != cfg.Relationships.Defaults.Category {
		t.Errorf("GetCategory() = %q, want default category %q", cat, cfg.Relationships.Defaults.Category)
	}
}

func TestDetectCycleWithConfig_SelfReference(t *testing.T) {
	cfg := DefaultConfig()
	repo := &mockRepository{features: make(map[string]*Feature)}

	source := &Feature{ID: "feat-1", Name: "Feature 1"}
	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-1", // Same as source
	}

	err := DetectCycleWithConfig(context.Background(), source, rel, repo, cfg)
	if err == nil {
		t.Error("Expected error for self-reference, got nil")
	}
	if !strings.Contains(err.Error(), "cannot create relationship to self") {
		t.Errorf("Expected self-reference error, got: %v", err)
	}
}

func TestDetectCycleWithConfig_StructuralCycle(t *testing.T) {
	cfg := DefaultConfig()

	// Create a mock repository with features
	feat1 := &Feature{
		ID:   "feat-1",
		Name: "Feature 1",
	}
	feat2 := &Feature{
		ID:   "feat-2",
		Name: "Feature 2",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-1"},
		},
	}

	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
			"feat-2": feat2,
		},
	}

	// Try to create feat-1 depends-on feat-2 (would create a cycle)
	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-2",
	}

	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err == nil {
		t.Error("Expected cycle detection error, got nil")
		return // Don't continue if err is nil
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("Expected cycle detection error, got: %v", err)
	}
}

func TestDetectCycleWithConfig_InformationalAllowsCycles(t *testing.T) {
	cfg := DefaultConfig()

	// Create circular informational relationships (should be allowed)
	feat1 := &Feature{
		ID:   "feat-1",
		Name: "Feature 1",
	}
	feat2 := &Feature{
		ID:   "feat-2",
		Name: "Feature 2",
		Relationships: []Relationship{
			{Type: "related-to", TargetID: "feat-1"},
		},
	}

	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
			"feat-2": feat2,
		},
	}

	// Try to create feat-1 related-to feat-2 (informational allows cycles)
	rel := &Relationship{
		Type:     "related-to",
		TargetID: "feat-2",
	}

	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err != nil {
		t.Errorf("Informational relationships should allow cycles, got error: %v", err)
	}
}

func TestDetectCycleWithConfig_WarnMode(t *testing.T) {
	cfg := DefaultConfig()

	// Workflow category has cycle_detection: warn
	feat1 := &Feature{
		ID:   "feat-1",
		Name: "Feature 1",
	}
	feat2 := &Feature{
		ID:   "feat-2",
		Name: "Feature 2",
		Relationships: []Relationship{
			{Type: "blocks", TargetID: "feat-1"},
		},
	}

	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
			"feat-2": feat2,
		},
	}

	// Try to create feat-1 blocks feat-2 (would create cycle in workflow)
	rel := &Relationship{
		Type:     "blocks",
		TargetID: "feat-2",
	}

	// Should not error (warn mode just logs)
	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err != nil {
		t.Errorf("Warn mode should not error, got: %v", err)
	}
}

func TestDetectCycleWithConfig_DifferentCategories(t *testing.T) {
	cfg := DefaultConfig()

	// Create features with relationships in different categories
	feat1 := &Feature{
		ID:   "feat-1",
		Name: "Feature 1",
	}
	feat2 := &Feature{
		ID:   "feat-2",
		Name: "Feature 2",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-3"}, // structural
		},
	}
	feat3 := &Feature{
		ID:   "feat-3",
		Name: "Feature 3",
		Relationships: []Relationship{
			{Type: "related-to", TargetID: "feat-1"}, // informational
		},
	}

	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
			"feat-2": feat2,
			"feat-3": feat3,
		},
	}

	// feat-1 depends-on feat-2 (structural)
	// feat-2 depends-on feat-3 (structural)
	// feat-3 related-to feat-1 (informational)
	// No cycle in structural category alone
	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-2",
	}

	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err != nil {
		t.Errorf("Different categories should not interfere, got error: %v", err)
	}
}

// TestRelationship_ValidateWithConfig_AllTypes tests all 17 relationship types
func TestRelationship_ValidateWithConfig_AllTypes(t *testing.T) {
	cfg := DefaultConfig()

	// All 17 types from config (9 primary + 8 inverse)
	allTypes := []struct {
		typ      string
		category string
	}{
		// Structural (8 types)
		{"depends-on", "structural"},
		{"required-by", "structural"},
		{"contains", "structural"},
		{"contained-by", "structural"},
		{"implements", "structural"},
		{"implemented-by", "structural"},
		{"replaces", "structural"},
		{"replaced-by", "structural"},
		// Informational (6 types)
		{"references", "informational"},
		{"referenced-by", "informational"},
		{"related-to", "informational"},
		{"conflicts-with", "informational"},
		{"tested-by", "informational"},
		{"tests", "informational"},
		// Workflow (2 types)
		{"blocks", "workflow"},
		{"blocked-by", "workflow"},
	}

	for _, tt := range allTypes {
		t.Run(tt.typ, func(t *testing.T) {
			rel := Relationship{
				Type:     RelationshipType(tt.typ),
				TargetID: "target-123",
			}

			err := rel.ValidateWithConfig(cfg)
			if err != nil {
				t.Errorf("ValidateWithConfig() for type %q failed: %v", tt.typ, err)
			}

			// Verify category
			cat := rel.GetCategory(cfg)
			if cat != tt.category {
				t.Errorf("GetCategory() = %q, want %q for type %q", cat, tt.category, tt.typ)
			}
		})
	}
}

// TestRelationship_ValidateWithConfig_AllAliases tests all defined aliases
func TestRelationship_ValidateWithConfig_AllAliases(t *testing.T) {
	cfg := DefaultConfig()

	aliases := []struct {
		alias     string
		canonical string
	}{
		{"requires", "depends-on"},
		{"needs", "depends-on"},
		{"has", "contains"},
		{"includes", "contains"},
		{"uses", "references"},
		{"mentions", "references"},
		{"relates", "related-to"},
		{"associated-with", "related-to"},
		{"incompatible-with", "conflicts-with"},
	}

	for _, tt := range aliases {
		t.Run(tt.alias, func(t *testing.T) {
			rel := Relationship{
				Type:     RelationshipType(tt.alias),
				TargetID: "target-123",
			}

			err := rel.ValidateWithConfig(cfg)
			if err != nil {
				t.Errorf("ValidateWithConfig() failed for alias %q: %v", tt.alias, err)
			}

			// Verify it was resolved to canonical name
			if rel.Type != RelationshipType(tt.canonical) {
				t.Errorf("Alias %q should resolve to %q, got %q", tt.alias, tt.canonical, rel.Type)
			}
		})
	}
}

// TestRelationship_ValidateWithConfig_InvalidCategory tests error when type references unknown category
func TestRelationship_ValidateWithConfig_InvalidCategory(t *testing.T) {
	// Create config with invalid category reference
	cfg := DefaultConfig()
	cfg.Relationships.Types["bad-type"] = RelationshipTypeConfig{
		Category:    "non-existent-category",
		Description: "Type with bad category",
	}

	rel := Relationship{
		Type:     "bad-type",
		TargetID: "target-123",
	}

	err := rel.ValidateWithConfig(cfg)
	if err == nil {
		t.Error("Expected error for type with unknown category, got nil")
	}
	if !strings.Contains(err.Error(), "unknown category") {
		t.Errorf("Expected unknown category error, got: %v", err)
	}
}

// TestDetectCycleWithConfig_TransitiveCycle tests cycle detection through multiple hops
func TestDetectCycleWithConfig_TransitiveCycle(t *testing.T) {
	cfg := DefaultConfig()

	// Create A → B → C → A cycle
	feat1 := &Feature{
		ID:   "feat-1",
		Name: "Feature 1",
	}
	feat2 := &Feature{
		ID:   "feat-2",
		Name: "Feature 2",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-3"},
		},
	}
	feat3 := &Feature{
		ID:   "feat-3",
		Name: "Feature 3",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-1"},
		},
	}

	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
			"feat-2": feat2,
			"feat-3": feat3,
		},
	}

	// Try to create feat-1 depends-on feat-2 (completes the cycle)
	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-2",
	}

	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err == nil {
		t.Error("Expected cycle detection error for transitive cycle, got nil")
		return
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("Expected cycle detection error, got: %v", err)
	}
}

// TestDetectCycleWithConfig_LongChain tests deep dependency chains
func TestDetectCycleWithConfig_LongChain(t *testing.T) {
	cfg := DefaultConfig()

	// Create A → B → C → D → E → A (5-node cycle)
	features := make(map[string]*Feature)
	ids := []string{"feat-1", "feat-2", "feat-3", "feat-4", "feat-5"}

	for i, id := range ids {
		feat := &Feature{
			ID:   id,
			Name: "Feature " + id,
		}
		// Link to next (except last one)
		if i < len(ids)-1 {
			feat.Relationships = []Relationship{
				{Type: "depends-on", TargetID: ids[i+1]},
			}
		} else {
			// Last one links back to first
			feat.Relationships = []Relationship{
				{Type: "depends-on", TargetID: ids[0]},
			}
		}
		features[id] = feat
	}

	repo := &mockRepository{features: features}

	// Try to add another relationship that would extend the cycle
	source := features["feat-1"]
	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-2",
	}

	// Should detect the existing cycle
	err := DetectCycleWithConfig(context.Background(), source, rel, repo, cfg)
	if err == nil {
		t.Error("Expected cycle detection in long chain, got nil")
		return
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("Expected cycle detection error, got: %v", err)
	}
}

// TestDetectCycleWithConfig_ComplexGraph tests branching dependency graph
func TestDetectCycleWithConfig_ComplexGraph(t *testing.T) {
	cfg := DefaultConfig()

	// Create complex graph:
	//     feat-1
	//    /      \
	// feat-2  feat-3
	//    \      /
	//     feat-4
	feat1 := &Feature{ID: "feat-1", Name: "F1"}
	feat2 := &Feature{
		ID:   "feat-2",
		Name: "F2",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-4"},
		},
	}
	feat3 := &Feature{
		ID:   "feat-3",
		Name: "F3",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-4"},
		},
	}
	feat4 := &Feature{ID: "feat-4", Name: "F4"}

	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
			"feat-2": feat2,
			"feat-3": feat3,
			"feat-4": feat4,
		},
	}

	// feat-1 → feat-2 (no cycle)
	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-2",
	}
	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err != nil {
		t.Errorf("No cycle should exist, got error: %v", err)
	}

	// feat-1 → feat-3 (no cycle)
	rel2 := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-3",
	}
	err = DetectCycleWithConfig(context.Background(), feat1, rel2, repo, cfg)
	if err != nil {
		t.Errorf("No cycle should exist, got error: %v", err)
	}

	// Now try feat-4 → feat-1 (would create cycle)
	rel3 := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-1",
	}

	// First add the relationships to feat-1 to complete the graph
	feat1.Relationships = []Relationship{
		{Type: "depends-on", TargetID: "feat-2"},
		{Type: "depends-on", TargetID: "feat-3"},
	}

	err = DetectCycleWithConfig(context.Background(), feat4, rel3, repo, cfg)
	if err == nil {
		t.Error("Expected cycle detection for feat-4 → feat-1, got nil")
		return
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("Expected cycle detection error, got: %v", err)
	}
}

// TestDetectCycleWithConfig_TargetNotFound tests behavior when target feature doesn't exist
func TestDetectCycleWithConfig_TargetNotFound(t *testing.T) {
	cfg := DefaultConfig()

	feat1 := &Feature{ID: "feat-1", Name: "F1"}
	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
		},
	}

	// Try to create relationship to non-existent feature
	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "non-existent",
	}

	// Should not error - cycle detection can't verify non-existent targets
	// This is intentional - existence validation happens elsewhere
	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err != nil {
		t.Errorf("Should not error for non-existent target, got: %v", err)
	}
}

// TestDetectCycleWithConfig_EmptyRepository tests behavior with empty repo
func TestDetectCycleWithConfig_EmptyRepository(t *testing.T) {
	cfg := DefaultConfig()

	feat1 := &Feature{ID: "feat-1", Name: "F1"}
	repo := &mockRepository{features: make(map[string]*Feature)}

	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-2",
	}

	// Should not error - no cycles in empty repo
	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err != nil {
		t.Errorf("Should not error for empty repo, got: %v", err)
	}
}

// TestDetectCycleWithConfig_MixedInverseTypes tests inverse relationship handling
func TestDetectCycleWithConfig_MixedInverseTypes(t *testing.T) {
	cfg := DefaultConfig()

	// feat-1 depends-on feat-2
	// feat-2 required-by should not create cycle (different type in same category)
	feat1 := &Feature{ID: "feat-1", Name: "F1"}
	feat2 := &Feature{
		ID:   "feat-2",
		Name: "F2",
		Relationships: []Relationship{
			{Type: "required-by", TargetID: "feat-1"},
		},
	}

	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
			"feat-2": feat2,
		},
	}

	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-2",
	}

	// Both are structural category, so should detect the cycle
	// feat-1 depends-on feat-2, feat-2 required-by feat-1 is essentially a cycle
	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err == nil {
		t.Error("Expected cycle detection with inverse types, got nil")
		return
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("Expected cycle detection error, got: %v", err)
	}
}

// TestDetectCycleWithConfig_BidirectionalRelationships tests bidirectional type handling
func TestDetectCycleWithConfig_BidirectionalRelationships(t *testing.T) {
	cfg := DefaultConfig()

	// related-to is bidirectional and allows cycles
	feat1 := &Feature{ID: "feat-1", Name: "F1"}
	feat2 := &Feature{
		ID:   "feat-2",
		Name: "F2",
		Relationships: []Relationship{
			{Type: "related-to", TargetID: "feat-1"},
		},
	}

	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
			"feat-2": feat2,
		},
	}

	rel := &Relationship{
		Type:     "related-to",
		TargetID: "feat-2",
	}

	// Should not error - informational category allows cycles
	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err != nil {
		t.Errorf("Bidirectional informational should allow cycles, got error: %v", err)
	}
}

// TestDetectCycleWithConfig_UnknownCategory tests error handling for unknown category
func TestDetectCycleWithConfig_UnknownCategory(t *testing.T) {
	cfg := DefaultConfig()

	// Add a type with unknown category
	cfg.Relationships.Types["bad-type"] = RelationshipTypeConfig{
		Category:    "unknown-category",
		Description: "Bad type",
	}

	feat1 := &Feature{ID: "feat-1", Name: "F1"}
	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
		},
	}

	rel := &Relationship{
		Type:     "bad-type",
		TargetID: "feat-2",
	}

	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err == nil {
		t.Error("Expected error for unknown category, got nil")
		return
	}
	if !strings.Contains(err.Error(), "unknown category") {
		t.Errorf("Expected unknown category error, got: %v", err)
	}
}

// TestRelationship_GetCategory_AllCategories tests all 4 categories
func TestRelationship_GetCategory_AllCategories(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		relType  string
		category string
	}{
		{"depends-on", "structural"},
		{"references", "informational"},
		{"blocks", "workflow"},
		// Note: No compliance types in default config, but we can test the mechanism
	}

	for _, tt := range tests {
		t.Run(tt.relType+"/"+tt.category, func(t *testing.T) {
			rel := Relationship{Type: RelationshipType(tt.relType)}
			cat := rel.GetCategory(cfg)
			if cat != tt.category {
				t.Errorf("GetCategory() = %q, want %q", cat, tt.category)
			}
		})
	}
}

// TestRelationship_GetCategory_Compliance tests compliance category if added
func TestRelationship_GetCategory_Compliance(t *testing.T) {
	cfg := DefaultConfig()

	// Add a compliance type
	cfg.Relationships.Types["complies-with"] = RelationshipTypeConfig{
		Category:    "compliance",
		Description: "Feature complies with requirement",
	}
	cfg.Relationships.Categories["compliance"] = RelationshipCategory{
		Description:     "Regulatory and compliance tracking",
		AllowCycles:     false,
		CycleDetection:  "strict",
		IncludeInImpact: true,
	}

	rel := Relationship{Type: "complies-with"}
	cat := rel.GetCategory(cfg)

	if cat != "compliance" {
		t.Errorf("GetCategory() = %q, want 'compliance'", cat)
	}
}

// TestDetectCycleWithConfig_VisitedTracking tests that visited nodes are properly tracked
func TestDetectCycleWithConfig_VisitedTracking(t *testing.T) {
	cfg := DefaultConfig()

	// Create diamond structure:
	//      feat-1
	//      /    \
	//  feat-2  feat-3
	//      \    /
	//      feat-4
	// Then feat-4 → feat-5 → feat-1 (cycle)

	feat1 := &Feature{ID: "feat-1", Name: "F1"}
	feat2 := &Feature{
		ID:   "feat-2",
		Name: "F2",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-4"},
		},
	}
	feat3 := &Feature{
		ID:   "feat-3",
		Name: "F3",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-4"},
		},
	}
	feat4 := &Feature{
		ID:   "feat-4",
		Name: "F4",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-5"},
		},
	}
	feat5 := &Feature{
		ID:   "feat-5",
		Name: "F5",
		Relationships: []Relationship{
			{Type: "depends-on", TargetID: "feat-1"},
		},
	}

	repo := &mockRepository{
		features: map[string]*Feature{
			"feat-1": feat1,
			"feat-2": feat2,
			"feat-3": feat3,
			"feat-4": feat4,
			"feat-5": feat5,
		},
	}

	// Try feat-1 → feat-2 (would complete cycle through multiple paths)
	rel := &Relationship{
		Type:     "depends-on",
		TargetID: "feat-2",
	}

	err := DetectCycleWithConfig(context.Background(), feat1, rel, repo, cfg)
	if err == nil {
		t.Error("Expected cycle detection, got nil")
		return
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("Expected cycle detection error, got: %v", err)
	}
}

// mockRepository for testing
type mockRepository struct {
	features map[string]*Feature
}

func (m *mockRepository) Create(ctx context.Context, f *Feature) error {
	m.features[f.ID] = f
	return nil
}

func (m *mockRepository) Get(ctx context.Context, id string) (*Feature, error) {
	f, ok := m.features[id]
	if !ok {
		return nil, ErrNotFound
	}
	return f, nil
}

func (m *mockRepository) Update(ctx context.Context, f *Feature) error {
	m.features[f.ID] = f
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	delete(m.features, id)
	return nil
}

func (m *mockRepository) List(ctx context.Context, filter *Filter) ([]*Feature, error) {
	var features []*Feature
	for _, f := range m.features {
		features = append(features, f)
	}
	return features, nil
}

func (m *mockRepository) FindBySlug(ctx context.Context, slug string) (*Feature, error) {
	return nil, ErrNotFound
}

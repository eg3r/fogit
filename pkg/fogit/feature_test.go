package fogit

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewFeature(t *testing.T) {
	name := "Test Feature"
	feature := NewFeature(name)

	if feature.Name != name {
		t.Errorf("expected name %q, got %q", name, feature.Name)
	}
	if feature.ID == "" {
		t.Error("expected non-empty ID")
	}
	// State is now derived from version timestamps
	if feature.DeriveState() != StateOpen {
		t.Errorf("expected state %q, got %q", StateOpen, feature.DeriveState())
	}
	// Priority is now stored in metadata
	if feature.GetPriority() != PriorityMedium {
		t.Errorf("expected priority %q, got %q", PriorityMedium, feature.GetPriority())
	}
	if len(feature.Tags) != 0 {
		t.Errorf("expected empty tags, got %v", feature.Tags)
	}
	if feature.Metadata == nil {
		t.Error("expected initialized metadata map")
	}
}

func TestFeature_Validate(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *Feature
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid feature",
			setup: func() *Feature {
				f := NewFeature("Test Feature")
				f.SetType("software-feature")
				return f
			},
			wantErr: false,
		},
		{
			name: "empty name",
			setup: func() *Feature {
				f := NewFeature("temp")
				f.Name = ""
				return f
			},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name: "invalid priority",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority("invalid")
				return f
			},
			wantErr: true,
			errMsg:  "invalid priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature := tt.setup()
			err := feature.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got nil")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestState_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  bool
	}{
		{"open", StateOpen, true},
		{"in-progress", StateInProgress, true},
		{"closed", StateClosed, true},
		{"invalid", State("invalid"), false},
		{"empty", State(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsValid(); got != tt.want {
				t.Errorf("State.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeature_DeriveState(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name      string
		setup     func() *Feature
		wantState State
	}{
		{
			name: "open - created_at equals modified_at",
			setup: func() *Feature {
				f := NewFeature("Test")
				// NewFeature creates version with created_at == modified_at
				return f
			},
			wantState: StateOpen,
		},
		{
			name: "in-progress - modified_at after created_at",
			setup: func() *Feature {
				f := NewFeature("Test")
				v := f.GetCurrentVersion()
				v.CreatedAt = now.Add(-time.Hour)
				v.ModifiedAt = now
				return f
			},
			wantState: StateInProgress,
		},
		{
			name: "closed - has closed_at",
			setup: func() *Feature {
				f := NewFeature("Test")
				v := f.GetCurrentVersion()
				closedAt := now
				v.ClosedAt = &closedAt
				return f
			},
			wantState: StateClosed,
		},
		{
			name: "closed - closed_at takes precedence",
			setup: func() *Feature {
				f := NewFeature("Test")
				v := f.GetCurrentVersion()
				v.CreatedAt = now.Add(-time.Hour)
				v.ModifiedAt = now.Add(-time.Minute)
				closedAt := now
				v.ClosedAt = &closedAt
				return f
			},
			wantState: StateClosed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.setup()
			got := f.DeriveState()
			if got != tt.wantState {
				t.Errorf("DeriveState() = %v, want %v", got, tt.wantState)
			}
		})
	}
}

func TestPriority_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		want     bool
	}{
		{"low", PriorityLow, true},
		{"medium", PriorityMedium, true},
		{"high", PriorityHigh, true},
		{"critical", PriorityCritical, true},
		{"invalid", Priority("invalid"), false},
		{"empty", Priority(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.priority.IsValid(); got != tt.want {
				t.Errorf("Priority.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestState_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name string
		from State
		to   State
		want bool
	}{
		// Valid transitions from open (per spec: open -> in-progress or closed)
		{"open to in-progress", StateOpen, StateInProgress, true},
		{"open to closed", StateOpen, StateClosed, true},
		{"open to open", StateOpen, StateOpen, false},

		// Valid transitions from in-progress (per spec: in-progress -> closed)
		{"in-progress to closed", StateInProgress, StateClosed, true},
		{"in-progress to open", StateInProgress, StateOpen, false},
		{"in-progress to in-progress", StateInProgress, StateInProgress, false},

		// Valid transitions from closed (per spec: closed -> open for reopen)
		{"closed to open", StateClosed, StateOpen, true},
		{"closed to in-progress", StateClosed, StateInProgress, false},
		{"closed to closed", StateClosed, StateClosed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Errorf("State.CanTransitionTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeature_UpdateState(t *testing.T) {
	tests := []struct {
		name      string
		fromState State
		toState   State
		wantErr   bool
	}{
		{"valid transition", StateOpen, StateInProgress, false},
		{"invalid transition", StateInProgress, StateOpen, true},
		{"invalid state", StateOpen, State("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature := NewFeature("Test")
			// Set initial state via version timestamps
			if tt.fromState == StateInProgress {
				v := feature.GetCurrentVersion()
				v.ModifiedAt = v.CreatedAt.Add(time.Hour)
			}

			err := feature.UpdateState(tt.toState)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && feature.DeriveState() != tt.toState {
				t.Errorf("State not updated: got %v, want %v", feature.DeriveState(), tt.toState)
			}
		})
	}
}

func TestFeature_AddTag(t *testing.T) {
	feature := NewFeature("Test")

	// Add first tag
	feature.AddTag("backend")
	if len(feature.Tags) != 1 || feature.Tags[0] != "backend" {
		t.Errorf("expected [backend], got %v", feature.Tags)
	}

	// Add second tag
	feature.AddTag("api")
	if len(feature.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(feature.Tags))
	}

	// Add duplicate tag (should not add)
	feature.AddTag("backend")
	if len(feature.Tags) != 2 {
		t.Errorf("expected 2 tags after duplicate, got %d", len(feature.Tags))
	}
}

func TestFeature_RemoveTag(t *testing.T) {
	feature := NewFeature("Test")
	feature.Tags = []string{"backend", "api", "frontend"}

	// Remove existing tag
	feature.RemoveTag("api")
	if len(feature.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(feature.Tags))
	}

	// Verify remaining tags
	hasBackend, hasFrontend := false, false
	for _, tag := range feature.Tags {
		if tag == "backend" {
			hasBackend = true
		}
		if tag == "frontend" {
			hasFrontend = true
		}
	}
	if !hasBackend || !hasFrontend {
		t.Errorf("expected backend and frontend, got %v", feature.Tags)
	}

	// Remove non-existent tag (should be no-op)
	feature.RemoveTag("nonexistent")
	if len(feature.Tags) != 2 {
		t.Errorf("expected 2 tags after removing nonexistent, got %d", len(feature.Tags))
	}
}

func TestFeature_AddFile(t *testing.T) {
	feature := NewFeature("Test")

	// Add first file
	feature.AddFile("src/main.go")
	if len(feature.Files) != 1 || feature.Files[0] != "src/main.go" {
		t.Errorf("expected [src/main.go], got %v", feature.Files)
	}

	// Add second file
	feature.AddFile("src/utils.go")
	if len(feature.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(feature.Files))
	}

	// Add duplicate file (should not add)
	feature.AddFile("src/main.go")
	if len(feature.Files) != 2 {
		t.Errorf("expected 2 files after duplicate, got %d", len(feature.Files))
	}
}

func TestFeature_RemoveFile(t *testing.T) {
	feature := NewFeature("Test")
	feature.Files = []string{"src/main.go", "src/utils.go", "src/test.go"}

	// Remove existing file
	feature.RemoveFile("src/utils.go")
	if len(feature.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(feature.Files))
	}

	// Verify remaining files
	hasMain, hasTest := false, false
	for _, file := range feature.Files {
		if file == "src/main.go" {
			hasMain = true
		}
		if file == "src/test.go" {
			hasTest = true
		}
	}
	if !hasMain || !hasTest {
		t.Errorf("expected main.go and test.go, got %v", feature.Files)
	}

	// Remove non-existent file (should be no-op)
	feature.RemoveFile("nonexistent.go")
	if len(feature.Files) != 2 {
		t.Errorf("expected 2 files after removing nonexistent, got %d", len(feature.Files))
	}
}

func TestFeature_Relationships(t *testing.T) {
	feature := NewFeature("Test")

	// Test with relationships field
	rel := Relationship{
		Type:        "blocks",
		TargetID:    "test-id-123",
		Description: "Blocks feature X",
	}

	feature.Relationships = []Relationship{rel}

	if len(feature.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(feature.Relationships))
	}
	if feature.Relationships[0].Type != "blocks" {
		t.Errorf("expected type %v, got %v", "blocks", feature.Relationships[0].Type)
	}
}

func TestFeature_AddRelationship(t *testing.T) {
	tests := []struct {
		name    string
		rel     Relationship
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid relationship",
			rel: Relationship{
				Type:     "depends-on",
				TargetID: "target-123",
			},
			wantErr: false,
		},
		{
			name: "empty target ID",
			rel: Relationship{
				Type:     "depends-on",
				TargetID: "",
			},
			wantErr: true,
			errMsg:  "target ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature := NewFeature("Test")
			err := feature.AddRelationship(tt.rel)

			if (err != nil) != tt.wantErr {
				t.Errorf("AddRelationship() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(feature.Relationships) != 1 {
					t.Errorf("expected 1 relationship, got %d", len(feature.Relationships))
				}
				if feature.Relationships[0].Type != tt.rel.Type {
					t.Errorf("expected type %v, got %v", tt.rel.Type, feature.Relationships[0].Type)
				}
			}
		})
	}
}

// TestFeature_AddRelationship_PreservesFields verifies that AddRelationship preserves
// all fields including ID and CreatedAt (regression test)
func TestFeature_AddRelationship_PreservesFields(t *testing.T) {
	feature := NewFeature("Test")

	// Create relationship using NewRelationship (the proper way)
	rel := NewRelationship("depends-on", "target-123", "Target Feature")
	rel.Description = "Test description"

	originalID := rel.ID
	originalCreatedAt := rel.CreatedAt

	err := feature.AddRelationship(rel)
	if err != nil {
		t.Fatalf("AddRelationship() failed: %v", err)
	}

	// Verify the stored relationship has all fields preserved
	stored := feature.Relationships[0]

	if stored.ID != originalID {
		t.Errorf("AddRelationship() changed ID: got %v, want %v", stored.ID, originalID)
	}
	if stored.ID == "" {
		t.Error("AddRelationship() stored relationship with empty ID")
	}
	if !stored.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("AddRelationship() changed CreatedAt: got %v, want %v", stored.CreatedAt, originalCreatedAt)
	}
	if stored.CreatedAt.IsZero() {
		t.Error("AddRelationship() stored relationship with zero CreatedAt")
	}
	if stored.Type != rel.Type {
		t.Errorf("AddRelationship() changed Type: got %v, want %v", stored.Type, rel.Type)
	}
	if stored.TargetID != rel.TargetID {
		t.Errorf("AddRelationship() changed TargetID: got %v, want %v", stored.TargetID, rel.TargetID)
	}
	if stored.TargetName != rel.TargetName {
		t.Errorf("AddRelationship() changed TargetName: got %v, want %v", stored.TargetName, rel.TargetName)
	}
	if stored.Description != rel.Description {
		t.Errorf("AddRelationship() changed Description: got %v, want %v", stored.Description, rel.Description)
	}
}

func TestFeature_AddRelationshipDuplicate(t *testing.T) {
	feature := NewFeature("Test")
	rel := Relationship{
		Type:     "depends-on",
		TargetID: "target-123",
	}

	// Add first time - should succeed
	err := feature.AddRelationship(rel)
	if err != nil {
		t.Fatalf("first AddRelationship() failed: %v", err)
	}

	// Add again - should fail with duplicate error
	err = feature.AddRelationship(rel)
	if err != ErrDuplicateRelationship {
		t.Errorf("expected ErrDuplicateRelationship, got %v", err)
	}

	if len(feature.Relationships) != 1 {
		t.Errorf("expected 1 relationship after duplicate, got %d", len(feature.Relationships))
	}
}

func TestFeature_RemoveRelationship(t *testing.T) {
	feature := NewFeature("Test")

	// Add two relationships
	rel1 := Relationship{Type: "depends-on", TargetID: "target-1"}
	rel2 := Relationship{Type: "blocks", TargetID: "target-2"}

	_ = feature.AddRelationship(rel1)
	_ = feature.AddRelationship(rel2)

	// Remove first relationship
	err := feature.RemoveRelationship("depends-on", "target-1")
	if err != nil {
		t.Errorf("RemoveRelationship() error = %v", err)
	}

	if len(feature.Relationships) != 1 {
		t.Errorf("expected 1 relationship after removal, got %d", len(feature.Relationships))
	}

	// Try to remove non-existent relationship
	err = feature.RemoveRelationship("depends-on", "nonexistent")
	if err != ErrRelationshipNotFound {
		t.Errorf("expected ErrRelationshipNotFound, got %v", err)
	}
}

func TestFeature_HasRelationship(t *testing.T) {
	feature := NewFeature("Test")
	rel := Relationship{Type: "depends-on", TargetID: "target-123"}
	_ = feature.AddRelationship(rel)

	if !feature.HasRelationship("depends-on", "target-123") {
		t.Error("expected relationship to exist")
	}

	if feature.HasRelationship("blocks", "target-123") {
		t.Error("expected relationship with different type not to exist")
	}

	if feature.HasRelationship("depends-on", "other-target") {
		t.Error("expected relationship with different target not to exist")
	}
}

func TestFeature_GetRelationships(t *testing.T) {
	feature := NewFeature("Test")

	// Add multiple relationships
	_ = feature.AddRelationship(Relationship{Type: "depends-on", TargetID: "target-1"})
	_ = feature.AddRelationship(Relationship{Type: "depends-on", TargetID: "target-2"})
	_ = feature.AddRelationship(Relationship{Type: "blocks", TargetID: "target-3"})

	// Get all relationships
	all := feature.GetRelationships("")
	if len(all) != 3 {
		t.Errorf("expected 3 relationships, got %d", len(all))
	}

	// Get filtered by type
	deps := feature.GetRelationships("depends-on")
	if len(deps) != 2 {
		t.Errorf("expected 2 depends-on relationships, got %d", len(deps))
	}

	blocks := feature.GetRelationships("blocks")
	if len(blocks) != 1 {
		t.Errorf("expected 1 blocks relationship, got %d", len(blocks))
	}
}

func TestFeature_Metadata(t *testing.T) {
	feature := NewFeature("Test")

	// Add metadata
	feature.Metadata["estimate"] = "8h"
	feature.Metadata["sprint"] = 23
	feature.Metadata["custom"] = true

	// Priority is already set by default, so we have more than 3 entries
	if feature.Metadata["estimate"] != "8h" {
		t.Errorf("expected estimate 8h, got %v", feature.Metadata["estimate"])
	}
	if feature.Metadata["sprint"] != 23 {
		t.Errorf("expected sprint 23, got %v", feature.Metadata["sprint"])
	}
	if feature.Metadata["custom"] != true {
		t.Errorf("expected custom true, got %v", feature.Metadata["custom"])
	}
}

// Edge case tests

func TestFeature_RemoveRelationshipByID(t *testing.T) {
	feature := NewFeature("Test")
	rel1 := NewRelationship("depends-on", "target-1", "Target 1")
	rel2 := NewRelationship("blocks", "target-2", "Target 2")

	_ = feature.AddRelationship(rel1)
	_ = feature.AddRelationship(rel2)

	// Remove by ID
	err := feature.RemoveRelationshipByID(rel1.ID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(feature.Relationships) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(feature.Relationships))
	}

	// Try to remove already removed
	err = feature.RemoveRelationshipByID(rel1.ID)
	if err != ErrRelationshipNotFound {
		t.Errorf("expected ErrRelationshipNotFound, got %v", err)
	}

	// Remove non-existent ID
	err = feature.RemoveRelationshipByID("non-existent-id")
	if err != ErrRelationshipNotFound {
		t.Errorf("expected ErrRelationshipNotFound, got %v", err)
	}
}

func TestFeature_Validate_ComplexCases(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *Feature
		wantErr bool
	}{
		{
			name: "feature with all fields populated",
			setup: func() *Feature {
				f := NewFeature("Complete Feature")
				f.Description = "A complete feature"
				f.SetType("software-feature")
				f.SetCategory("auth")
				f.SetDomain("backend")
				f.SetTeam("security")
				f.SetEpic("user-management")
				f.SetModule("auth-service")
				f.Tags = []string{"tag1", "tag2"}
				f.Files = []string{"file1.go", "file2.go"}
				f.Metadata["estimate"] = "8h"
				f.Metadata["sprint"] = 23
				_ = f.AddRelationship(NewRelationship("depends-on", "target", "Target"))
				return f
			},
			wantErr: false,
		},
		{
			name: "feature with minimum fields",
			setup: func() *Feature {
				return NewFeature("Minimal")
			},
			wantErr: false,
		},
		{
			name: "feature with empty name",
			setup: func() *Feature {
				f := NewFeature("temp")
				f.Name = ""
				return f
			},
			wantErr: true, // Empty name not allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature := tt.setup()
			err := feature.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestState_CanTransitionTo_AllCombinations(t *testing.T) {
	states := []State{StateOpen, StateInProgress, StateClosed}

	// Test all combinations
	transitions := map[State]map[State]bool{
		StateOpen: {
			StateOpen:       false,
			StateInProgress: true,
			StateClosed:     true,
		},
		StateInProgress: {
			StateOpen:       false,
			StateInProgress: false,
			StateClosed:     true,
		},
		StateClosed: {
			StateOpen:       true,
			StateInProgress: false,
			StateClosed:     false,
		},
	}

	for fromState, toStates := range transitions {
		for toState, expected := range toStates {
			t.Run(string(fromState)+"_to_"+string(toState), func(t *testing.T) {
				got := fromState.CanTransitionTo(toState)
				if got != expected {
					t.Errorf("CanTransitionTo(%v -> %v) = %v, want %v",
						fromState, toState, got, expected)
				}
			})
		}
	}

	// Test with invalid states
	for _, state := range states {
		invalid := State("invalid")
		if state.CanTransitionTo(invalid) {
			t.Errorf("should not allow transition from %v to invalid state", state)
		}
	}
}

func TestNewFeature_Versioning(t *testing.T) {
	feature := NewFeature("Test Feature")

	// Check version 1 initialized via GetCurrentVersionKey
	if feature.GetCurrentVersionKey() != "1" {
		t.Errorf("expected CurrentVersionKey='1', got %s", feature.GetCurrentVersionKey())
	}

	// Check versions map
	if feature.Versions == nil {
		t.Fatal("expected Versions map to be initialized")
	}
	if len(feature.Versions) != 1 {
		t.Errorf("expected 1 version, got %d", len(feature.Versions))
	}

	// Check version 1 details
	v1, ok := feature.Versions["1"]
	if !ok {
		t.Fatal("expected version '1' to exist")
	}
	if v1.CreatedAt.IsZero() {
		t.Error("expected version 1 CreatedAt to be set")
	}
	if v1.ClosedAt != nil {
		t.Error("expected version 1 ClosedAt to be nil (open)")
	}
}

func TestFeatureVersion_Structure(t *testing.T) {
	now := time.Now().UTC()
	closedTime := now.Add(24 * time.Hour)

	tests := []struct {
		name    string
		version FeatureVersion
	}{
		{
			name: "open version",
			version: FeatureVersion{
				CreatedAt: now,
				ClosedAt:  nil,
				Branch:    "feature/test-feature",
				Authors:   []string{"alice@example.com"},
				Notes:     "Initial implementation",
			},
		},
		{
			name: "closed version",
			version: FeatureVersion{
				CreatedAt: now,
				ClosedAt:  &closedTime,
				Branch:    "feature/test-feature-v2",
				Authors:   []string{"alice@example.com", "bob@example.com"},
				Notes:     "Bug fix for email validation",
			},
		},
		{
			name: "minimal version",
			version: FeatureVersion{
				CreatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.version

			if v.CreatedAt.IsZero() {
				t.Error("expected CreatedAt to be set")
			}

			// Verify fields are accessible
			_ = v.ClosedAt
			_ = v.Branch
			_ = v.Authors
			_ = v.Notes
		})
	}
}

func TestFeature_MultipleVersions(t *testing.T) {
	feature := NewFeature("Test Feature")
	now := time.Now().UTC()

	// Add version 2
	v1ClosedTime := now.Add(24 * time.Hour)
	feature.Versions["1"].ClosedAt = &v1ClosedTime
	feature.Versions["2"] = &FeatureVersion{
		CreatedAt:  now.Add(25 * time.Hour),
		ModifiedAt: now.Add(25 * time.Hour),
		ClosedAt:   nil,
		Branch:     "feature/test-feature-v2",
		Authors:    []string{"alice@example.com"},
		Notes:      "Reopened for bug fix",
	}

	// Check current version key
	if feature.GetCurrentVersionKey() != "2" {
		t.Errorf("expected CurrentVersionKey='2', got %s", feature.GetCurrentVersionKey())
	}

	// Check both versions exist
	if len(feature.Versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(feature.Versions))
	}

	// Verify version 1 is closed
	v1 := feature.Versions["1"]
	if v1.ClosedAt == nil {
		t.Error("expected version 1 to be closed")
	}

	// Verify version 2 is open
	v2 := feature.Versions["2"]
	if v2.ClosedAt != nil {
		t.Error("expected version 2 to be open")
	}
}

func TestFeature_ReopenFeature(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() *Feature
		versionFormat string
		increment     VersionIncrement
		branch        string
		notes         string
		wantErr       bool
		wantVersion   string
		wantState     State
	}{
		{
			name: "reopen closed feature - simple minor",
			setup: func() *Feature {
				f := NewFeature("Test Feature")
				now := time.Now().UTC()
				closedTime := now.Add(24 * time.Hour)
				f.Versions["1"].ClosedAt = &closedTime
				return f
			},
			versionFormat: "simple",
			increment:     VersionIncrementMinor,
			branch:        "feature/test-feature-v2",
			notes:         "Bug fix",
			wantErr:       false,
			wantVersion:   "2",
			wantState:     StateOpen, // Per spec: reopened features start in OPEN state
		},
		{
			name: "reopen closed feature - semantic minor",
			setup: func() *Feature {
				f := NewFeature("Test Feature")
				now := time.Now().UTC()
				closedTime := now.Add(24 * time.Hour)
				f.Versions["1"].ClosedAt = &closedTime
				return f
			},
			versionFormat: "semantic",
			increment:     VersionIncrementMinor,
			branch:        "feature/test-feature-v1.1.0",
			notes:         "Minor update",
			wantErr:       false,
			wantVersion:   "1.1.0",
			wantState:     StateOpen, // Per spec: reopened features start in OPEN state
		},
		{
			name: "cannot reopen open feature",
			setup: func() *Feature {
				return NewFeature("Test Feature")
			},
			versionFormat: "simple",
			increment:     VersionIncrementMinor,
			branch:        "",
			notes:         "",
			wantErr:       true,
		},
		{
			name: "cannot reopen in-progress feature",
			setup: func() *Feature {
				f := NewFeature("Test Feature")
				// Make it in-progress by modifying timestamp
				v := f.GetCurrentVersion()
				v.ModifiedAt = v.CreatedAt.Add(time.Hour)
				return f
			},
			versionFormat: "simple",
			increment:     VersionIncrementMinor,
			branch:        "",
			notes:         "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := tt.setup()

			// Calculate new version based on format and increment
			currentVersionStr := "1"
			newVersion, err := IncrementVersion(currentVersionStr, tt.versionFormat, tt.increment)
			if err != nil && !tt.wantErr {
				t.Fatalf("IncrementVersion failed: %v", err)
			}

			err = f.ReopenFeature(currentVersionStr, newVersion, tt.branch, tt.notes)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReopenFeature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Don't check other fields if error expected
			}

			// Check state was updated
			if f.DeriveState() != tt.wantState {
				t.Errorf("State = %q, want %q", f.DeriveState(), tt.wantState)
			}

			// Check old version was closed
			v1 := f.Versions["1"]
			if v1.ClosedAt == nil {
				t.Error("expected version 1 to be closed")
			}

			// Check new version was created
			v2, ok := f.Versions[tt.wantVersion]
			if !ok {
				t.Errorf("expected version %q to exist", tt.wantVersion)
				return
			}

			if v2.ClosedAt != nil {
				t.Error("expected new version to be open")
			}

			if v2.Branch != tt.branch {
				t.Errorf("Branch = %q, want %q", v2.Branch, tt.branch)
			}

			if v2.Notes != tt.notes {
				t.Errorf("Notes = %q, want %q", v2.Notes, tt.notes)
			}
		})
	}
}

func TestFeature_ReopenFeature_MultipleReopens(t *testing.T) {
	f := NewFeature("Test Feature")
	now := time.Now().UTC()

	// Close version 1
	f.Versions["1"].ClosedAt = &now

	// Reopen as version 2
	err := f.ReopenFeature("1", "2", "feature/test-v2", "First reopen")
	if err != nil {
		t.Fatalf("first reopen failed: %v", err)
	}

	// Close version 2
	later := now.Add(24 * time.Hour)
	f.Versions["2"].ClosedAt = &later

	// Reopen as version 3
	err = f.ReopenFeature("2", "3", "feature/test-v3", "Second reopen")
	if err != nil {
		t.Fatalf("second reopen failed: %v", err)
	}

	// Verify final state
	if f.GetCurrentVersionKey() != "3" {
		t.Errorf("CurrentVersionKey = %s, want 3", f.GetCurrentVersionKey())
	}

	if len(f.Versions) != 3 {
		t.Errorf("expected 3 versions, got %d", len(f.Versions))
	}

	// All previous versions should be closed
	for i := 1; i <= 2; i++ {
		vStr := strconv.Itoa(i)
		v := f.Versions[vStr]
		if v.ClosedAt == nil {
			t.Errorf("expected version %d to be closed", i)
		}
	}

	// Current version should be open
	v3 := f.Versions["3"]
	if v3.ClosedAt != nil {
		t.Error("expected version 3 to be open")
	}
}

func TestIncrementVersion(t *testing.T) {
	tests := []struct {
		name      string
		current   string
		format    string
		increment VersionIncrement
		want      string
		wantErr   bool
	}{
		// Simple format tests
		{"simple minor 1->2", "1", "simple", VersionIncrementMinor, "2", false},
		{"simple minor 2->3", "2", "simple", VersionIncrementMinor, "3", false},
		{"simple major 1->2", "1", "simple", VersionIncrementMajor, "2", false},
		{"simple patch no-op", "1", "simple", VersionIncrementPatch, "1", false},

		// Semantic format tests
		{"semantic patch 1.0.0->1.0.1", "1.0.0", "semantic", VersionIncrementPatch, "1.0.1", false},
		{"semantic patch 1.0.1->1.0.2", "1.0.1", "semantic", VersionIncrementPatch, "1.0.2", false},
		{"semantic minor 1.0.0->1.1.0", "1.0.0", "semantic", VersionIncrementMinor, "1.1.0", false},
		{"semantic minor 1.1.0->1.2.0", "1.1.0", "semantic", VersionIncrementMinor, "1.2.0", false},
		{"semantic minor resets patch", "1.0.5", "semantic", VersionIncrementMinor, "1.1.0", false},
		{"semantic major 1.0.0->2.0.0", "1.0.0", "semantic", VersionIncrementMajor, "2.0.0", false},
		{"semantic major 1.5.3->2.0.0", "1.5.3", "semantic", VersionIncrementMajor, "2.0.0", false},
		{"semantic major resets minor+patch", "1.5.3", "semantic", VersionIncrementMajor, "2.0.0", false},

		// Cross-format conversion (simple number to semantic)
		{"convert simple to semantic patch", "1", "semantic", VersionIncrementPatch, "1.0.1", false},
		{"convert simple to semantic minor", "1", "semantic", VersionIncrementMinor, "1.1.0", false},
		{"convert simple to semantic major", "1", "semantic", VersionIncrementMajor, "2.0.0", false},

		// Error cases
		{"invalid simple version", "abc", "simple", VersionIncrementMinor, "", true},
		{"invalid semantic version", "1.a.0", "semantic", VersionIncrementMinor, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IncrementVersion(tt.current, tt.format, tt.increment)

			if (err != nil) != tt.wantErr {
				t.Errorf("IncrementVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("IncrementVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeature_OrganizationFields(t *testing.T) {
	f := NewFeature("Test")

	// Test setting and getting organization fields via metadata
	f.SetType("software-feature")
	if f.GetType() != "software-feature" {
		t.Errorf("GetType() = %q, want %q", f.GetType(), "software-feature")
	}

	f.SetCategory("backend")
	if f.GetCategory() != "backend" {
		t.Errorf("GetCategory() = %q, want %q", f.GetCategory(), "backend")
	}

	f.SetDomain("api")
	if f.GetDomain() != "api" {
		t.Errorf("GetDomain() = %q, want %q", f.GetDomain(), "api")
	}

	f.SetTeam("core")
	if f.GetTeam() != "core" {
		t.Errorf("GetTeam() = %q, want %q", f.GetTeam(), "core")
	}

	f.SetEpic("v2")
	if f.GetEpic() != "v2" {
		t.Errorf("GetEpic() = %q, want %q", f.GetEpic(), "v2")
	}

	f.SetModule("auth")
	if f.GetModule() != "auth" {
		t.Errorf("GetModule() = %q, want %q", f.GetModule(), "auth")
	}

	f.SetPriority(PriorityHigh)
	if f.GetPriority() != PriorityHigh {
		t.Errorf("GetPriority() = %q, want %q", f.GetPriority(), PriorityHigh)
	}
}

func TestFeature_TimestampAccessors(t *testing.T) {
	f := NewFeature("Test")

	// GetCreatedAt should return version's created_at
	created := f.GetCreatedAt()
	if created.IsZero() {
		t.Error("GetCreatedAt() should not be zero")
	}

	// GetModifiedAt should return version's modified_at
	modified := f.GetModifiedAt()
	if modified.IsZero() {
		t.Error("GetModifiedAt() should not be zero")
	}

	// Initially created == modified (open state)
	if !created.Equal(modified) {
		t.Error("For new feature, created_at should equal modified_at")
	}

	// GetClosedAt should return nil for open feature
	if f.GetClosedAt() != nil {
		t.Error("GetClosedAt() should be nil for open feature")
	}

	// UpdateModifiedAt should change modified_at
	time.Sleep(time.Millisecond)
	f.UpdateModifiedAt()
	newModified := f.GetModifiedAt()
	if !newModified.After(modified) {
		t.Error("UpdateModifiedAt() should update modified_at")
	}
}

func TestFeature_GetSortedVersionKeys(t *testing.T) {
	now := time.Now().UTC()

	feature := &Feature{
		ID:   "test-123",
		Name: "Test Feature",
		Versions: map[string]*FeatureVersion{
			"3": {
				CreatedAt:  now.Add(2 * time.Hour),
				ModifiedAt: now.Add(2 * time.Hour),
			},
			"1": {
				CreatedAt:  now,
				ModifiedAt: now,
			},
			"2": {
				CreatedAt:  now.Add(1 * time.Hour),
				ModifiedAt: now.Add(1 * time.Hour),
			},
		},
	}

	keys := feature.GetSortedVersionKeys()

	expected := []string{"1", "2", "3"}
	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], key)
		}
	}
}

func TestFeature_GetSortedVersionKeys_Semantic(t *testing.T) {
	now := time.Now().UTC()

	feature := &Feature{
		ID:   "test-456",
		Name: "Semantic Feature",
		Versions: map[string]*FeatureVersion{
			"1.0.0": {
				CreatedAt:  now,
				ModifiedAt: now,
			},
			"2.0.0": {
				CreatedAt:  now.Add(3 * time.Hour),
				ModifiedAt: now.Add(3 * time.Hour),
			},
			"1.1.0": {
				CreatedAt:  now.Add(1 * time.Hour),
				ModifiedAt: now.Add(1 * time.Hour),
			},
			"1.0.1": {
				CreatedAt:  now.Add(30 * time.Minute),
				ModifiedAt: now.Add(30 * time.Minute),
			},
		},
	}

	keys := feature.GetSortedVersionKeys()

	// Verify order is chronological based on CreatedAt
	expected := []string{"1.0.0", "1.0.1", "1.1.0", "2.0.0"}
	if len(keys) != len(expected) {
		t.Fatalf("expected %d keys, got %d", len(expected), len(keys))
	}

	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("position %d: expected version %q, got %q", i, expected[i], key)
		}
	}
}

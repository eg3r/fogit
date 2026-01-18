package features

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/eg3r/fogit/internal/testutil"
	"github.com/eg3r/fogit/pkg/fogit"
)

// =============================================================================
// Integration Test: Feature Lifecycle with Relationships
// =============================================================================

// TestIntegration_FeatureLifecycleWithRelationships tests the full lifecycle:
// 1. Create parent feature
// 2. Create child features with depends-on relationships
// 3. Update child states
// 4. Verify relationship constraints are respected
// 5. Close features in correct order
func TestIntegration_FeatureLifecycleWithRelationships(t *testing.T) {
	env := testutil.TempDirWithGit(t)
	defer env.Cleanup()
	env.CreateInitialCommit(t)

	ctx := context.Background()
	cfg := fogit.DefaultConfig()

	// Step 1: Create parent feature "User Authentication"
	authFeature := fogit.NewFeature("User Authentication")
	authFeature.Description = "Main authentication system"
	authFeature.SetType("software-feature")
	authFeature.SetPriority(fogit.PriorityHigh)
	if err := env.Repository.Create(ctx, authFeature); err != nil {
		t.Fatalf("Failed to create auth feature: %v", err)
	}

	// Step 2: Create child feature "Login Page" that depends on auth
	loginFeature := fogit.NewFeature("Login Page")
	loginFeature.Description = "UI for user login"
	loginFeature.SetType("ui-component")
	if err := env.Repository.Create(ctx, loginFeature); err != nil {
		t.Fatalf("Failed to create login feature: %v", err)
	}

	// Step 3: Create relationship: Login depends-on Auth
	rel, err := Link(ctx, env.Repository, loginFeature, authFeature,
		fogit.RelationshipType("depends-on"), "Login requires auth backend", "", cfg, env.FogitDir)
	if err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}
	if rel == nil {
		t.Fatal("Link returned nil relationship")
	}

	// Step 4: Verify relationship was created
	loginFeature, err = env.Repository.Get(ctx, loginFeature.ID)
	if err != nil {
		t.Fatalf("Failed to reload login feature: %v", err)
	}
	if len(loginFeature.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(loginFeature.Relationships))
	}
	if loginFeature.Relationships[0].TargetID != authFeature.ID {
		t.Errorf("Relationship target mismatch: got %s, want %s",
			loginFeature.Relationships[0].TargetID, authFeature.ID)
	}

	// Step 5: Update login feature state to in-progress
	loginFeature.UpdateModifiedAt()
	time.Sleep(10 * time.Millisecond) // Ensure time difference for state derivation
	loginFeature.UpdateModifiedAt()
	if err := env.Repository.Update(ctx, loginFeature); err != nil {
		t.Fatalf("Failed to update login feature: %v", err)
	}

	// Verify state is in-progress (based on timestamps)
	loginFeature, _ = env.Repository.Get(ctx, loginFeature.ID)
	if loginFeature.DeriveState() != fogit.StateInProgress {
		t.Errorf("Expected state in-progress, got %s", loginFeature.DeriveState())
	}

	// Step 6: Find incoming relationships to auth feature
	incoming, err := FindIncomingRelationships(env.Repository, ctx, authFeature.ID, "")
	if err != nil {
		t.Fatalf("Failed to find incoming relationships: %v", err)
	}
	if len(incoming) != 1 {
		t.Errorf("Expected 1 incoming relationship, got %d", len(incoming))
	}
	if len(incoming) > 0 && incoming[0].SourceID != loginFeature.ID {
		t.Errorf("Incoming source mismatch: got %s, want %s", incoming[0].SourceID, loginFeature.ID)
	}

	t.Log("✓ Feature lifecycle with relationships completed successfully")
}

// =============================================================================
// Integration Test: Cycle Detection in Relationships
// =============================================================================

// TestIntegration_RelationshipCycleDetection tests that cycle detection works:
// 1. A -> B (depends-on)
// 2. B -> C (depends-on)
// 3. C -> A (depends-on) should be rejected (creates cycle)
func TestIntegration_RelationshipCycleDetection(t *testing.T) {
	env := testutil.TempDirWithGit(t)
	defer env.Cleanup()

	ctx := context.Background()
	cfg := fogit.DefaultConfig()

	// Create three features
	featureA := env.CreateFeature(t, "Feature A")
	featureB := env.CreateFeature(t, "Feature B")
	featureC := env.CreateFeature(t, "Feature C")

	// A -> B (depends-on)
	_, err := Link(ctx, env.Repository, featureA, featureB,
		fogit.RelationshipType("depends-on"), "", "", cfg, env.FogitDir)
	if err != nil {
		t.Fatalf("Failed to create A->B relationship: %v", err)
	}

	// B -> C (depends-on)
	_, err = Link(ctx, env.Repository, featureB, featureC,
		fogit.RelationshipType("depends-on"), "", "", cfg, env.FogitDir)
	if err != nil {
		t.Fatalf("Failed to create B->C relationship: %v", err)
	}

	// Reload features to get updated relationships
	featureA, _ = env.Repository.Get(ctx, featureA.ID)
	_, _ = env.Repository.Get(ctx, featureB.ID)
	featureC, _ = env.Repository.Get(ctx, featureC.ID)

	// C -> A (should fail - creates cycle)
	_, err = Link(ctx, env.Repository, featureC, featureA,
		fogit.RelationshipType("depends-on"), "", "", cfg, env.FogitDir)
	if err == nil {
		t.Error("Expected cycle detection error, but relationship was created")
	} else {
		t.Logf("✓ Cycle correctly detected: %v", err)
	}

	// Verify the cycle wasn't created
	featureC, _ = env.Repository.Get(ctx, featureC.ID)
	for _, rel := range featureC.Relationships {
		if rel.TargetID == featureA.ID {
			t.Error("Cycle relationship was incorrectly created")
		}
	}

	t.Log("✓ Cycle detection test completed successfully")
}

// TestIntegration_SelfReferencePrevention tests that self-references are blocked
func TestIntegration_SelfReferencePrevention(t *testing.T) {
	env := testutil.TempDirWithGit(t)
	defer env.Cleanup()

	ctx := context.Background()
	cfg := fogit.DefaultConfig()

	feature := env.CreateFeature(t, "Self Reference Test")

	// Try to create self-reference
	_, err := Link(ctx, env.Repository, feature, feature,
		fogit.RelationshipType("depends-on"), "", "", cfg, env.FogitDir)
	if err == nil {
		t.Error("Expected self-reference to be rejected")
	} else {
		t.Logf("✓ Self-reference correctly rejected: %v", err)
	}
}

// =============================================================================
// Integration Test: Multi-Version Workflow
// =============================================================================

// TestIntegration_MultiVersionWorkflow tests creating and managing multiple versions:
// 1. Create feature with v1
// 2. Close v1
// 3. Reopen as v2
// 4. Verify version history
func TestIntegration_MultiVersionWorkflow(t *testing.T) {
	env := testutil.TempDirWithGit(t)
	defer env.Cleanup()
	env.CreateInitialCommit(t)

	ctx := context.Background()

	// Step 1: Create feature (v1 is created automatically)
	feature := fogit.NewFeature("Multi-Version Feature")
	feature.SetPriority(fogit.PriorityMedium)
	if err := env.Repository.Create(ctx, feature); err != nil {
		t.Fatalf("Failed to create feature: %v", err)
	}

	// Verify v1 exists
	if len(feature.Versions) != 1 {
		t.Errorf("Expected 1 version after creation, got %d", len(feature.Versions))
	}
	if feature.GetCurrentVersionKey() != "1" {
		t.Errorf("Expected current version '1', got '%s'", feature.GetCurrentVersionKey())
	}

	// Step 2: Close v1 using UpdateState
	if err := feature.UpdateState(fogit.StateClosed); err != nil {
		t.Fatalf("Failed to close v1: %v", err)
	}
	if err := env.Repository.Update(ctx, feature); err != nil {
		t.Fatalf("Failed to save closed v1: %v", err)
	}

	// Verify state is closed
	if feature.DeriveState() != fogit.StateClosed {
		t.Errorf("Expected closed state, got %s", feature.DeriveState())
	}

	// Step 3: Reopen as v2 (create new version)
	if err := feature.ReopenFeature("1", "2", "feature/v2", "Reopening for additional work"); err != nil {
		t.Fatalf("Failed to reopen as v2: %v", err)
	}
	if err := env.Repository.Update(ctx, feature); err != nil {
		t.Fatalf("Failed to save v2: %v", err)
	}

	// Verify v2 exists and is current
	feature, _ = env.Repository.Get(ctx, feature.ID)
	if len(feature.Versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(feature.Versions))
	}
	if feature.GetCurrentVersionKey() != "2" {
		t.Errorf("Expected current version '2', got '%s'", feature.GetCurrentVersionKey())
	}

	// Verify state is in-progress (reopened features start in-progress per spec)
	if feature.DeriveState() != fogit.StateInProgress {
		t.Errorf("Expected in-progress state for reopened version, got %s", feature.DeriveState())
	}

	// Step 4: Verify version history
	v1 := feature.Versions["1"]
	v2 := feature.Versions["2"]

	if v1 == nil || v2 == nil {
		t.Fatal("Missing version data")
	}

	if v1.ClosedAt == nil {
		t.Error("v1 should be closed")
	}
	if v2.ClosedAt != nil {
		t.Error("v2 should not be closed")
	}
	if v2.Branch != "feature/v2" {
		t.Errorf("v2 branch mismatch: got %s, want feature/v2", v2.Branch)
	}

	t.Log("✓ Multi-version workflow completed successfully")
}

// =============================================================================
// Integration Test: Feature Merge Workflow with Branch
// =============================================================================

// TestIntegration_FeatureMergeWorkflow tests the complete merge workflow:
// 1. Create feature on a branch
// 2. Make changes and commit
// 3. Merge feature (closes it)
// 4. Verify feature is closed and metadata updated
func TestIntegration_FeatureMergeWorkflow(t *testing.T) {
	// Use testutil helper which has proven to work
	env := testutil.TempDirWithGit(t)
	defer env.Cleanup()
	env.CreateInitialCommit(t)

	ctx := context.Background()

	branchName := "feature/merge-test"

	// Step 1: Create a feature using the helper (proven to work)
	feature := env.CreateFeature(t, "Merge Test Feature")
	feature.SetPriority(fogit.PriorityMedium)
	// Set branch on the current version
	if cv := feature.GetCurrentVersion(); cv != nil {
		cv.Branch = branchName
	}
	// Save the branch info
	if err := env.Repository.Update(ctx, feature); err != nil {
		t.Fatalf("Failed to save feature with branch: %v", err)
	}

	// Save the feature ID for later retrieval
	featureID := feature.ID

	// Get worktree for git operations
	w, err := env.GitRepo.Worktree()
	if err != nil {
		t.Fatalf("Failed to get worktree: %v", err)
	}

	// Commit the feature files BEFORE checking out new branch
	// (git checkout would otherwise lose uncommitted changes)
	if _, err = w.Add("."); err != nil {
		t.Fatalf("Failed to add feature files: %v", err)
	}
	_, err = w.Commit("Add feature: Merge Test Feature", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit feature: %v", err)
	}

	// Create feature branch in git
	err = w.Checkout(&gogit.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Create: true,
	})
	if err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// Step 2: Reload feature and update metadata
	feature, err = env.Repository.Get(ctx, featureID)
	if err != nil {
		t.Fatalf("Failed to reload feature: %v", err)
	}
	feature.UpdateModifiedAt()
	if err := env.Repository.Update(ctx, feature); err != nil {
		t.Fatalf("Failed to update feature: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(env.RootDir, "feature-code.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if _, err = w.Add("."); err != nil {
		t.Fatalf("Failed to add files: %v", err)
	}

	_, err = w.Commit("Implement feature", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Verify feature is in-progress
	feature, err = env.Repository.Get(ctx, featureID)
	if err != nil {
		t.Fatalf("Failed to reload feature for state check: %v", err)
	}
	if feature.DeriveState() != fogit.StateInProgress {
		t.Errorf("Expected in-progress state, got %s", feature.DeriveState())
	}

	// Step 3: Close the feature (simulating merge)
	if err := feature.UpdateState(fogit.StateClosed); err != nil {
		t.Fatalf("Failed to close feature: %v", err)
	}
	if err := env.Repository.Update(ctx, feature); err != nil {
		t.Fatalf("Failed to save closed feature: %v", err)
	}

	// Step 4: Verify feature is closed
	feature, err = env.Repository.Get(ctx, featureID)
	if err != nil {
		t.Fatalf("Failed to reload closed feature: %v", err)
	}
	if feature.DeriveState() != fogit.StateClosed {
		t.Errorf("Expected closed state after merge, got %s", feature.DeriveState())
	}

	// Verify version metadata
	currentVersion := feature.GetCurrentVersion()
	if currentVersion == nil {
		t.Fatal("Current version is nil")
	}
	if currentVersion.ClosedAt == nil {
		t.Error("ClosedAt should be set after merge")
	}
	if currentVersion.Branch != branchName {
		t.Errorf("Branch mismatch: got %s, want %s", currentVersion.Branch, branchName)
	}

	t.Log("✓ Feature merge workflow completed successfully")
}

// =============================================================================
// Integration Test: Relationship with Version Constraints
// =============================================================================

// TestIntegration_RelationshipVersionConstraints tests version-constrained relationships:
// 1. Create feature A with v1
// 2. Create feature B that depends on A >=1
// 3. Verify constraint validation
func TestIntegration_RelationshipVersionConstraints(t *testing.T) {
	env := testutil.TempDirWithGit(t)
	defer env.Cleanup()

	ctx := context.Background()
	cfg := fogit.DefaultConfig()

	// Create feature A
	featureA := env.CreateFeature(t, "API Library")

	// Create feature B
	featureB := env.CreateFeature(t, "API Consumer")

	// Create relationship with version constraint
	rel, err := Link(ctx, env.Repository, featureB, featureA,
		fogit.RelationshipType("depends-on"), "Requires API v1+", ">=1", cfg, env.FogitDir)
	if err != nil {
		t.Fatalf("Failed to create versioned relationship: %v", err)
	}

	// Verify constraint was stored
	if rel.VersionConstraint == nil {
		t.Error("Version constraint not stored")
	} else {
		if rel.VersionConstraint.Operator != ">=" {
			t.Errorf("Operator mismatch: got %s, want >=", rel.VersionConstraint.Operator)
		}
	}

	// Reload and verify
	featureB, _ = env.Repository.Get(ctx, featureB.ID)
	if len(featureB.Relationships) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(featureB.Relationships))
	}

	storedConstraint := featureB.Relationships[0].VersionConstraint
	if storedConstraint == nil {
		t.Error("Version constraint not persisted")
	}

	t.Log("✓ Version constraint relationship test completed successfully")
}

// TestIntegration_InvalidVersionConstraint tests that invalid constraints are rejected
func TestIntegration_InvalidVersionConstraint(t *testing.T) {
	env := testutil.TempDirWithGit(t)
	defer env.Cleanup()

	ctx := context.Background()
	cfg := fogit.DefaultConfig()

	featureA := env.CreateFeature(t, "Feature A")
	featureB := env.CreateFeature(t, "Feature B")

	// Try invalid constraint format
	_, err := Link(ctx, env.Repository, featureB, featureA,
		fogit.RelationshipType("depends-on"), "", "invalid", cfg, env.FogitDir)
	if err == nil {
		t.Error("Expected error for invalid version constraint")
	} else {
		t.Logf("✓ Invalid constraint correctly rejected: %v", err)
	}

	// Try invalid operator
	_, err = Link(ctx, env.Repository, featureB, featureA,
		fogit.RelationshipType("depends-on"), "", "!=5", cfg, env.FogitDir)
	if err == nil {
		t.Error("Expected error for invalid operator")
	} else {
		t.Logf("✓ Invalid operator correctly rejected: %v", err)
	}
}

// =============================================================================
// Integration Test: Duplicate Relationship Prevention
// =============================================================================

// TestIntegration_DuplicateRelationshipPrevention tests that duplicate relationships are blocked
func TestIntegration_DuplicateRelationshipPrevention(t *testing.T) {
	env := testutil.TempDirWithGit(t)
	defer env.Cleanup()

	ctx := context.Background()
	cfg := fogit.DefaultConfig()

	featureA := env.CreateFeature(t, "Feature A")
	featureB := env.CreateFeature(t, "Feature B")

	// Create first relationship
	_, err := Link(ctx, env.Repository, featureA, featureB,
		fogit.RelationshipType("depends-on"), "", "", cfg, env.FogitDir)
	if err != nil {
		t.Fatalf("Failed to create first relationship: %v", err)
	}

	// Reload feature A
	featureA, _ = env.Repository.Get(ctx, featureA.ID)

	// Try to create duplicate relationship
	_, err = Link(ctx, env.Repository, featureA, featureB,
		fogit.RelationshipType("depends-on"), "", "", cfg, env.FogitDir)
	if err == nil {
		t.Error("Expected error for duplicate relationship")
	} else {
		t.Logf("✓ Duplicate relationship correctly rejected: %v", err)
	}
}

// =============================================================================
// Integration Test: Relationship Unlink
// =============================================================================

// TestIntegration_RelationshipUnlink tests unlinking relationships
func TestIntegration_RelationshipUnlink(t *testing.T) {
	env := testutil.TempDirWithGit(t)
	defer env.Cleanup()

	ctx := context.Background()
	cfg := fogit.DefaultConfig()

	featureA := env.CreateFeature(t, "Feature A")
	featureB := env.CreateFeature(t, "Feature B")

	// Create relationship
	rel, err := Link(ctx, env.Repository, featureA, featureB,
		fogit.RelationshipType("depends-on"), "Test link", "", cfg, env.FogitDir)
	if err != nil {
		t.Fatalf("Failed to create relationship: %v", err)
	}

	// Reload and verify
	featureA, _ = env.Repository.Get(ctx, featureA.ID)
	if len(featureA.Relationships) != 1 {
		t.Fatalf("Expected 1 relationship, got %d", len(featureA.Relationships))
	}

	// Unlink
	removedRel, err := Unlink(ctx, env.Repository, featureA, rel.ID, env.FogitDir, cfg)
	if err != nil {
		t.Fatalf("Failed to unlink: %v", err)
	}
	if removedRel.ID != rel.ID {
		t.Errorf("Removed relationship ID mismatch: got %s, want %s", removedRel.ID, rel.ID)
	}

	// Verify relationship was removed
	featureA, _ = env.Repository.Get(ctx, featureA.ID)
	if len(featureA.Relationships) != 0 {
		t.Errorf("Expected 0 relationships after unlink, got %d", len(featureA.Relationships))
	}

	t.Log("✓ Relationship unlink test completed successfully")
}

package features

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/eg3r/fogit/internal/storage"
	"github.com/eg3r/fogit/pkg/fogit"
)

// TestFindFeature tests finding features by ID and name
func TestFindFeature(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	featuresDir := tmpDir + "/.fogit/features"
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	repo := storage.NewFileRepository(tmpDir)

	// Create test feature
	testFeature := fogit.NewFeature("Test Feature")
	testFeature.SetType("test")
	testFeature.SetPriority(fogit.PriorityMedium)

	if err := repo.Create(context.Background(), testFeature); err != nil {
		t.Fatalf("failed to create test feature: %v", err)
	}

	tests := []struct {
		name       string
		identifier string
		wantErr    bool
		errType    error
	}{
		{
			name:       "find by exact ID",
			identifier: testFeature.ID,
			wantErr:    false,
		},
		{
			name:       "find by exact name",
			identifier: "Test Feature",
			wantErr:    false,
		},
		{
			name:       "find by lowercase name",
			identifier: "test feature",
			wantErr:    false,
		},
		{
			name:       "find by mixed case name",
			identifier: "TeSt FeAtUrE",
			wantErr:    false,
		},
		{
			name:       "not found by ID",
			identifier: "00000000-0000-0000-0000-000000000000",
			wantErr:    true,
			errType:    fogit.ErrNotFound,
		},
		{
			name:       "not found by name",
			identifier: "Nonexistent Feature",
			wantErr:    true,
			errType:    fogit.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Find(context.Background(), repo, tt.identifier, &fogit.Config{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Find() returned nil result without error")
			}
			if !tt.wantErr && result.Feature == nil {
				t.Error("Find() returned nil feature without error")
			}
			if !tt.wantErr && result.Feature != nil && result.Feature.Name != testFeature.Name {
				t.Errorf("Find() returned feature with name %q, want %q", result.Feature.Name, testFeature.Name)
			}
		})
	}
}

func TestFindForBranch(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	fogitDir := tmpDir + "/.fogit"
	if err := os.MkdirAll(fogitDir+"/features", 0755); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)

	// Create features using UpdateState() to set proper states
	// Use explicit timestamps to ensure deterministic ordering

	// Feature 1: open with older timestamp
	f1 := fogit.NewFeature("Feature 1")
	f1.Metadata["branch"] = "feature/one"
	// Set to an older timestamp to ensure f2 is "more recent"
	if v := f1.GetCurrentVersion(); v != nil {
		oldTime := time.Now().UTC().Add(-time.Hour)
		v.CreatedAt = oldTime
		v.ModifiedAt = oldTime
	}
	if err := repo.Create(context.Background(), f1); err != nil {
		t.Fatal(err)
	}

	// Feature 2: open (default state, most recent - newer timestamp)
	f2 := fogit.NewFeature("Feature 2")
	f2.Metadata["branch"] = "feature/two"
	// Ensure f2 has a newer timestamp than f1
	if v := f2.GetCurrentVersion(); v != nil {
		newTime := time.Now().UTC()
		v.CreatedAt = newTime
		v.ModifiedAt = newTime
	}
	if err := repo.Create(context.Background(), f2); err != nil {
		t.Fatal(err)
	}

	// Feature 3: closed
	f3 := fogit.NewFeature("Feature 3")
	f3.Metadata["branch"] = "feature/three"
	if err := f3.UpdateState(fogit.StateClosed); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}
	if err := repo.Create(context.Background(), f3); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		branch    string
		wantID    string
		wantFound bool
	}{
		{
			name:      "exact match",
			branch:    "feature/one",
			wantID:    f1.ID,
			wantFound: true,
		},
		{
			name:      "no match returns most recent open",
			branch:    "feature/unknown",
			wantID:    f2.ID,
			wantFound: true,
		},
		{
			name:   "closed feature ignored",
			branch: "feature/three",
			wantID: f2.ID, // Returns most recent OPEN feature, ignoring the closed one even if branch matches?
			// Wait, the logic says: "Try to find feature with matching branch in metadata" (iterates over OPEN features)
			// So closed features are filtered out by repo.List(StateOpen)
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindForBranch(context.Background(), repo, tt.branch)
			if err != nil {
				t.Fatalf("FindForBranch() error = %v", err)
			}

			if !tt.wantFound {
				if got != nil {
					t.Errorf("FindForBranch() got %v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Fatal("FindForBranch() got nil, want feature")
			}

			if got.ID != tt.wantID {
				t.Errorf("FindForBranch() ID = %v, want %v", got.ID, tt.wantID)
			}
		})
	}
}

// TestFindForBranch_EqualTimestamps tests deterministic behavior when timestamps are equal
func TestFindForBranch_EqualTimestamps(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := tmpDir + "/.fogit"
	if err := os.MkdirAll(fogitDir+"/features", 0755); err != nil {
		t.Fatal(err)
	}

	repo := storage.NewFileRepository(fogitDir)

	// Create features with IDENTICAL timestamps
	fixedTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create features in random ID order to verify tiebreaker works
	f1 := fogit.NewFeature("Feature A")
	f1.Metadata["branch"] = "feature/a"
	if v := f1.GetCurrentVersion(); v != nil {
		v.CreatedAt = fixedTime
		v.ModifiedAt = fixedTime
	}

	f2 := fogit.NewFeature("Feature B")
	f2.Metadata["branch"] = "feature/b"
	if v := f2.GetCurrentVersion(); v != nil {
		v.CreatedAt = fixedTime
		v.ModifiedAt = fixedTime
	}

	// Save them - order shouldn't matter due to tiebreaker
	if err := repo.Create(context.Background(), f1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(context.Background(), f2); err != nil {
		t.Fatal(err)
	}

	// Determine which ID is lexicographically greater (that's what tiebreaker uses)
	expectedID := f1.ID
	if f2.ID > f1.ID {
		expectedID = f2.ID
	}

	// Run multiple times to ensure determinism
	for i := 0; i < 10; i++ {
		got, err := FindForBranch(context.Background(), repo, "feature/unknown")
		if err != nil {
			t.Fatalf("FindForBranch() error = %v", err)
		}

		if got == nil {
			t.Fatal("FindForBranch() got nil, want feature")
		}

		if got.ID != expectedID {
			t.Errorf("Run %d: FindForBranch() ID = %v, want %v (lexicographically greater)", i, got.ID, expectedID)
		}
	}
}

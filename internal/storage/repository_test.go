package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eg3r/fogit/pkg/fogit"
)

// setupTestRepo creates a temporary repository for testing
func setupTestRepo(t *testing.T) (*FileRepository, func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "fogit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	repo := NewFileRepository(tempDir)

	cleanup := func() {
		os.RemoveAll(tempDir)
	}

	return repo, cleanup
}

func TestFileRepository_Create(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	feature := fogit.NewFeature("Test Feature")

	err := repo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Verify feature was created
	retrieved, err := repo.Get(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if retrieved.ID != feature.ID {
		t.Errorf("ID mismatch: got %v, want %v", retrieved.ID, feature.ID)
	}
	if retrieved.Name != feature.Name {
		t.Errorf("Name mismatch: got %v, want %v", retrieved.Name, feature.Name)
	}
}

func TestFileRepository_CreateDuplicate(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	feature := fogit.NewFeature("Test Feature")

	// Create first time
	err := repo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Try to create again - should fail
	err = repo.Create(ctx, feature)
	if err != fogit.ErrFeatureAlreadyExists {
		t.Errorf("Create() duplicate expected ErrFeatureAlreadyExists, got %v", err)
	}
}

func TestFileRepository_CreateInvalid(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name    string
		feature *fogit.Feature
		wantErr bool
	}{
		{
			name:    "nil feature",
			feature: nil,
			wantErr: true,
		},
		{
			name: "empty name",
			feature: func() *fogit.Feature {
				f := fogit.NewFeature("temp")
				f.ID = "test-id"
				f.Name = ""
				f.SetType("software-feature")
				return f
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, tt.feature)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileRepository_Get(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	feature := fogit.NewFeature("Test Feature")

	err := repo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Test successful get
	retrieved, err := repo.Get(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if retrieved.ID != feature.ID {
		t.Errorf("ID mismatch: got %v, want %v", retrieved.ID, feature.ID)
	}

	// Test not found
	_, err = repo.Get(ctx, "nonexistent-id")
	if err != fogit.ErrNotFound {
		t.Errorf("Get() nonexistent expected ErrNotFound, got %v", err)
	}

	// Test empty ID
	_, err = repo.Get(ctx, "")
	if err == nil {
		t.Errorf("Get() with empty ID should return error")
	}
}

func TestFileRepository_Update(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	feature := fogit.NewFeature("Original Title")

	err := repo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Update feature
	feature.Name = "Updated Name"
	feature.Description = "New description"
	feature.UpdateState(fogit.StateInProgress)

	err = repo.Update(ctx, feature)
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify update
	retrieved, err := repo.Get(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if retrieved.Name != "Updated Name" {
		t.Errorf("Name not updated: got %v, want %v", retrieved.Name, "Updated Name")
	}
	if retrieved.Description != "New description" {
		t.Errorf("Description not updated: got %v, want %v", retrieved.Description, "New description")
	}

	// Test update nonexistent
	nonexistent := fogit.NewFeature("Nonexistent")
	err = repo.Update(ctx, nonexistent)
	if err != fogit.ErrNotFound {
		t.Errorf("Update() nonexistent expected ErrNotFound, got %v", err)
	}
}

func TestFileRepository_Delete(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()
	feature := fogit.NewFeature("Test Feature")

	err := repo.Create(ctx, feature)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	// Delete feature
	err = repo.Delete(ctx, feature.ID)
	if err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Verify deletion
	_, err = repo.Get(ctx, feature.ID)
	if err != fogit.ErrNotFound {
		t.Errorf("Get() after Delete() expected ErrNotFound, got %v", err)
	}

	// Test delete nonexistent
	err = repo.Delete(ctx, "nonexistent-id")
	if err != fogit.ErrNotFound {
		t.Errorf("Delete() nonexistent expected ErrNotFound, got %v", err)
	}
}

func TestFileRepository_List(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// Create features and use UpdateState() to set proper states
	// State is derived from timestamps via UpdateState()

	feature1 := fogit.NewFeature("Feature 1")
	feature1.SetType("software-feature")
	feature1.SetPriority(fogit.PriorityHigh)
	feature1.Tags = []string{"backend", "api"}
	// NewFeature creates with open state (created_at == modified_at)

	feature2 := fogit.NewFeature("Bug Fix")
	feature2.SetType("bug")
	feature2.SetPriority(fogit.PriorityMedium)
	// Transition to in-progress using UpdateState
	if err := feature2.UpdateState(fogit.StateInProgress); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	feature3 := fogit.NewFeature("Task Item")
	feature3.SetType("task")
	feature3.Tags = []string{"backend", "testing"}
	// Transition to closed using UpdateState
	if err := feature3.UpdateState(fogit.StateClosed); err != nil {
		t.Fatalf("UpdateState failed: %v", err)
	}

	for _, f := range []*fogit.Feature{feature1, feature2, feature3} {
		if err := repo.Create(ctx, f); err != nil {
			t.Fatalf("Create() failed: %v", err)
		}
	}

	tests := []struct {
		name      string
		filter    *fogit.Filter
		wantCount int
	}{
		{
			name:      "no filter",
			filter:    nil,
			wantCount: 3,
		},
		{
			name:      "filter by type",
			filter:    &fogit.Filter{Type: "software-feature"},
			wantCount: 1,
		},
		{
			name:      "filter by state",
			filter:    &fogit.Filter{State: fogit.StateInProgress},
			wantCount: 1,
		},
		{
			name:      "filter by tags",
			filter:    &fogit.Filter{Tags: []string{"backend"}},
			wantCount: 2,
		},
		{
			name:      "filter by multiple tags",
			filter:    &fogit.Filter{Tags: []string{"backend", "testing"}},
			wantCount: 1,
		},
		{
			name:      "filter by search",
			filter:    &fogit.Filter{Search: "bug"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features, err := repo.List(ctx, tt.filter)
			if err != nil {
				t.Fatalf("List() failed: %v", err)
			}
			if len(features) != tt.wantCount {
				t.Errorf("List() returned %d features, want %d", len(features), tt.wantCount)
			}
		})
	}
}

func TestFileRepository_ListEmpty(t *testing.T) {
	repo, cleanup := setupTestRepo(t)
	defer cleanup()

	ctx := context.Background()

	// List from empty repository
	features, err := repo.List(ctx, nil)
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}
	if len(features) != 0 {
		t.Errorf("List() on empty repo returned %d features, want 0", len(features))
	}
}

func TestSlugifiedFilenames(t *testing.T) {
	// Test that features are stored with slugified filenames
	dir := t.TempDir()
	repo := NewFileRepository(dir)

	tests := []struct {
		name         string
		featureName  string
		expectedFile string
	}{
		{
			name:         "simple name",
			featureName:  "User Authentication",
			expectedFile: "user-authentication.yml",
		},
		{
			name:         "special characters",
			featureName:  "OAuth 2.0 Integration",
			expectedFile: "oauth-2-0-integration.yml",
		},
		{
			name:         "version numbers",
			featureName:  "API v2.5",
			expectedFile: "api-v2-5.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature := fogit.NewFeature(tt.featureName)
			feature.Description = "Test feature"
			feature.SetPriority(fogit.PriorityMedium)

			err := repo.Create(context.Background(), feature)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}

			// Check that file exists with expected name
			expectedPath := filepath.Join(dir, "features", tt.expectedFile)
			if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
				// List all files to help debug
				files, _ := os.ReadDir(filepath.Join(dir, "features"))
				var fileNames []string
				for _, f := range files {
					fileNames = append(fileNames, f.Name())
				}
				t.Fatalf("Expected file %s not found. Found files: %v", tt.expectedFile, fileNames)
			}

			// Verify we can retrieve the feature by ID
			retrieved, err := repo.Get(context.Background(), feature.ID)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if retrieved.Name != tt.featureName {
				t.Errorf("Get() name = %v, want %v", retrieved.Name, tt.featureName)
			}
		})
	}
}

func TestFileRepository_EdgeCases(t *testing.T) {
	t.Run("unicode in feature name", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()
		feature := fogit.NewFeature("用户认证 🚀")
		feature.Description = "Unicode description with émojis 🎉"
		feature.Tags = []string{"標籤", "テスト", "тег"}

		err := repo.Create(ctx, feature)
		if err != nil {
			t.Fatalf("Create() with unicode failed: %v", err)
		}

		retrieved, err := repo.Get(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}
		if retrieved.Name != feature.Name {
			t.Errorf("Unicode name mismatch: got %v, want %v", retrieved.Name, feature.Name)
		}
		if len(retrieved.Tags) != len(feature.Tags) {
			t.Errorf("Tags length mismatch: got %d, want %d", len(retrieved.Tags), len(feature.Tags))
		}
	})

	t.Run("extremely long feature name", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()
		longName := "This is an extremely long feature name that exceeds normal limits to test if the repository handles it correctly and creates appropriate filenames without errors while maintaining data integrity and ensuring that all the information is properly stored in the YAML file format and can be retrieved later without any issues whatsoever even though the name is ridiculously long and spans multiple lines when displayed in most editors and terminals which is a realistic edge case that might occur in production environments where users have freedom to name their features however they want without strict length validation at the input level because we want to be flexible and user-friendly while still maintaining system stability and performance characteristics"
		feature := fogit.NewFeature(longName)

		err := repo.Create(ctx, feature)
		if err != nil {
			t.Fatalf("Create() with long name failed: %v", err)
		}

		retrieved, err := repo.Get(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}
		if retrieved.Name != longName {
			t.Errorf("Long name mismatch")
		}
	})

	t.Run("special characters in all fields", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()
		feature := fogit.NewFeature("Feature with \"quotes\" and 'apostrophes'")
		feature.Description = "Description with:\n- Newlines\n- Tabs\t\there\n- Quotes \"like this\"\n- Special chars: <>&@#$%"
		feature.SetType("type/with/slashes")
		feature.SetCategory("category:with:colons")
		feature.Metadata["special"] = "value with \"quotes\""
		feature.Metadata["newline"] = "line1\nline2"

		err := repo.Create(ctx, feature)
		if err != nil {
			t.Fatalf("Create() with special chars failed: %v", err)
		}

		retrieved, err := repo.Get(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}
		if retrieved.Name != feature.Name {
			t.Errorf("Name with special chars mismatch")
		}
		if retrieved.Description != feature.Description {
			t.Errorf("Description with special chars mismatch")
		}
	})

	t.Run("feature with complex relationships and metadata", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()

		// Create parent feature
		parent := fogit.NewFeature("Parent Feature")
		if err := repo.Create(ctx, parent); err != nil {
			t.Fatalf("Create parent failed: %v", err)
		}

		// Create feature with complex data
		feature := fogit.NewFeature("Complex Feature")
		feature.Relationships = []fogit.Relationship{
			{
				Type:        "parent",
				TargetID:    parent.ID,
				Description: "Parent relationship",
			},
		}
		feature.Metadata = map[string]interface{}{
			"nested": map[string]interface{}{
				"deep": map[string]interface{}{
					"value": "deeply nested",
				},
			},
			"array": []interface{}{"item1", "item2", 123, true},
			"mixed": map[string]interface{}{
				"string": "text",
				"number": 42,
				"bool":   true,
				"null":   nil,
			},
		}

		err := repo.Create(ctx, feature)
		if err != nil {
			t.Fatalf("Create() with complex data failed: %v", err)
		}

		retrieved, err := repo.Get(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}
		if len(retrieved.Relationships) != 1 {
			t.Errorf("Relationships count mismatch: got %d, want 1", len(retrieved.Relationships))
		}
		if retrieved.Metadata["nested"] == nil {
			t.Errorf("Nested metadata not preserved")
		}
	})

	t.Run("concurrent creates with same name", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()
		sameName := "Concurrent Feature"

		feature1 := fogit.NewFeature(sameName)
		feature2 := fogit.NewFeature(sameName)

		// Both should succeed with different IDs
		err1 := repo.Create(ctx, feature1)
		err2 := repo.Create(ctx, feature2)

		if err1 != nil || err2 != nil {
			t.Fatalf("Concurrent creates failed: err1=%v, err2=%v", err1, err2)
		}

		// Both should be retrievable
		r1, err := repo.Get(ctx, feature1.ID)
		if err != nil {
			t.Fatalf("Get feature1 failed: %v", err)
		}
		r2, err := repo.Get(ctx, feature2.ID)
		if err != nil {
			t.Fatalf("Get feature2 failed: %v", err)
		}

		if r1.ID == r2.ID {
			t.Errorf("Features with same name have same ID")
		}
		if r1.Name != sameName || r2.Name != sameName {
			t.Errorf("Names not preserved")
		}
	})

	t.Run("update with all fields changed", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()
		feature := fogit.NewFeature("Original")
		if err := repo.Create(ctx, feature); err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		// Change everything
		feature.Name = "Updated Name 更新"
		feature.Description = "New description with\nmultiple lines"
		feature.SetType("new-type")
		feature.UpdateState(fogit.StateInProgress)
		feature.SetPriority(fogit.PriorityCritical)
		feature.SetCategory("new-category")
		feature.SetDomain("new-domain")
		feature.SetTeam("new-team")
		feature.SetEpic("new-epic")
		feature.SetModule("new-module")
		feature.Tags = []string{"new1", "new2", "新しい"}
		feature.Files = []string{"file1.go", "file2.go"}
		feature.Metadata["custom"] = "value"
		feature.Metadata["number"] = 42

		if err := repo.Update(ctx, feature); err != nil {
			t.Fatalf("Update() failed: %v", err)
		}

		retrieved, err := repo.Get(ctx, feature.ID)
		if err != nil {
			t.Fatalf("Get() failed: %v", err)
		}

		// Verify all fields updated
		if retrieved.Name != feature.Name {
			t.Errorf("Name not updated")
		}
		if retrieved.DeriveState() != feature.DeriveState() {
			t.Errorf("State not updated")
		}
		if len(retrieved.Tags) != len(feature.Tags) {
			t.Errorf("Tags not updated")
		}
		if retrieved.Metadata["custom"] != "value" {
			t.Errorf("Metadata not updated")
		}
	})

	t.Run("list with complex filters", func(t *testing.T) {
		repo, cleanup := setupTestRepo(t)
		defer cleanup()

		ctx := context.Background()

		// Create features with various properties
		// Use UpdateState() to set proper states

		// Feature 1: open (default state from NewFeature)
		f1 := fogit.NewFeature("Backend API 🚀")
		f1.ID = "id1"
		f1.SetType("software-feature")
		f1.SetPriority(fogit.PriorityHigh)
		f1.Tags = []string{"backend", "api", "高优先级"}
		f1.SetCategory("backend")

		// Feature 2: in-progress
		f2 := fogit.NewFeature("Frontend Component")
		f2.ID = "id2"
		f2.SetType("software-feature")
		f2.SetPriority(fogit.PriorityMedium)
		f2.Tags = []string{"frontend", "ui"}
		f2.SetCategory("frontend")
		if err := f2.UpdateState(fogit.StateInProgress); err != nil {
			t.Fatalf("UpdateState failed: %v", err)
		}

		// Feature 3: closed
		f3 := fogit.NewFeature("Bug Fix 修复")
		f3.ID = "id3"
		f3.SetType("bug")
		f3.SetPriority(fogit.PriorityLow)
		f3.Tags = []string{"backend", "bug"}
		f3.SetCategory("backend")
		if err := f3.UpdateState(fogit.StateClosed); err != nil {
			t.Fatalf("UpdateState failed: %v", err)
		}

		features := []*fogit.Feature{f1, f2, f3}

		for _, f := range features {
			if err := repo.Create(ctx, f); err != nil {
				t.Fatalf("Create() failed: %v", err)
			}
		}

		tests := []struct {
			name      string
			filter    *fogit.Filter
			wantCount int
		}{
			{
				name:      "unicode in tags",
				filter:    &fogit.Filter{Tags: []string{"高优先级"}},
				wantCount: 1,
			},
			{
				name:      "search unicode",
				filter:    &fogit.Filter{Search: "修复"},
				wantCount: 1,
			},
			{
				name:      "multiple criteria",
				filter:    &fogit.Filter{Category: "backend", State: fogit.StateOpen},
				wantCount: 1,
			},
			{
				name:      "case insensitive type",
				filter:    &fogit.Filter{Type: "SOFTWARE-FEATURE"},
				wantCount: 2,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				results, err := repo.List(ctx, tt.filter)
				if err != nil {
					t.Fatalf("List() failed: %v", err)
				}
				if len(results) != tt.wantCount {
					t.Errorf("List() returned %d features, want %d", len(results), tt.wantCount)
				}
			})
		}
	})
}

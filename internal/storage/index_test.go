package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestIDIndex_SetGet(t *testing.T) {
	tmpDir := t.TempDir()
	idx := NewIDIndex(tmpDir)

	// Test Set and Get
	idx.Set("id-123", "feature-a.yml")
	idx.Set("id-456", "feature-b.yml")

	if got := idx.Get("id-123"); got != "feature-a.yml" {
		t.Errorf("Get(id-123) = %q, want %q", got, "feature-a.yml")
	}

	if got := idx.Get("id-456"); got != "feature-b.yml" {
		t.Errorf("Get(id-456) = %q, want %q", got, "feature-b.yml")
	}

	// Test non-existent
	if got := idx.Get("nonexistent"); got != "" {
		t.Errorf("Get(nonexistent) = %q, want empty", got)
	}
}

func TestIDIndex_Has(t *testing.T) {
	tmpDir := t.TempDir()
	idx := NewIDIndex(tmpDir)

	idx.Set("id-123", "feature-a.yml")

	if !idx.Has("id-123") {
		t.Error("Has(id-123) = false, want true")
	}

	if idx.Has("nonexistent") {
		t.Error("Has(nonexistent) = true, want false")
	}
}

func TestIDIndex_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	idx := NewIDIndex(tmpDir)

	idx.Set("id-123", "feature-a.yml")
	idx.Delete("id-123")

	if idx.Has("id-123") {
		t.Error("Has(id-123) = true after delete, want false")
	}

	if got := idx.Get("id-123"); got != "" {
		t.Errorf("Get(id-123) after delete = %q, want empty", got)
	}
}

func TestIDIndex_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	idx := NewIDIndex(tmpDir)

	// Add entries and save
	idx.Set("id-123", "feature-a.yml")
	idx.Set("id-456", "feature-b.yml")

	if err := idx.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file was created
	indexPath := filepath.Join(tmpDir, "metadata", "id_index.json")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		t.Error("Index file was not created")
	}

	// Create new index and load
	idx2 := NewIDIndex(tmpDir)
	if err := idx2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := idx2.Get("id-123"); got != "feature-a.yml" {
		t.Errorf("After load, Get(id-123) = %q, want %q", got, "feature-a.yml")
	}

	if got := idx2.Get("id-456"); got != "feature-b.yml" {
		t.Errorf("After load, Get(id-456) = %q, want %q", got, "feature-b.yml")
	}
}

func TestIDIndex_LoadNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	idx := NewIDIndex(tmpDir)

	// Load should succeed with empty index when file doesn't exist
	if err := idx.Load(); err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if len(idx.Entries) != 0 {
		t.Errorf("Entries length = %d, want 0", len(idx.Entries))
	}
}

func TestIDIndex_Rebuild(t *testing.T) {
	tmpDir := t.TempDir()
	featuresDir := filepath.Join(tmpDir, "features")

	// Create features directory
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("Failed to create features dir: %v", err)
	}

	// Create some feature files
	f1 := fogit.NewFeature("Feature A")
	f2 := fogit.NewFeature("Feature B")

	if err := WriteFeatureFile(filepath.Join(featuresDir, "feature-a.yml"), f1); err != nil {
		t.Fatalf("Failed to write feature A: %v", err)
	}
	if err := WriteFeatureFile(filepath.Join(featuresDir, "feature-b.yml"), f2); err != nil {
		t.Fatalf("Failed to write feature B: %v", err)
	}

	// Create index and rebuild
	idx := NewIDIndex(tmpDir)
	if err := idx.Rebuild(featuresDir); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	// Check entries
	if len(idx.Entries) != 2 {
		t.Errorf("Entries length = %d, want 2", len(idx.Entries))
	}

	if got := idx.Get(f1.ID); got != "feature-a.yml" {
		t.Errorf("Get(%s) = %q, want %q", f1.ID, got, "feature-a.yml")
	}

	if got := idx.Get(f2.ID); got != "feature-b.yml" {
		t.Errorf("Get(%s) = %q, want %q", f2.ID, got, "feature-b.yml")
	}
}

func TestIDIndex_RebuildEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	idx := NewIDIndex(tmpDir)

	// Rebuild with non-existent features dir should succeed
	if err := idx.Rebuild(filepath.Join(tmpDir, "features")); err != nil {
		t.Fatalf("Rebuild() error = %v, want nil", err)
	}

	if len(idx.Entries) != 0 {
		t.Errorf("Entries length = %d, want 0", len(idx.Entries))
	}
}

func TestFileRepository_IndexIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")

	// Create features directory
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("Failed to create features dir: %v", err)
	}

	repo := NewFileRepository(fogitDir)
	ctx := t.Context()

	// Create a feature
	f1 := fogit.NewFeature("Test Feature")
	if err := repo.Create(ctx, f1); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get should use index (O(1) lookup)
	got, err := repo.Get(ctx, f1.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name != f1.Name {
		t.Errorf("Get() name = %q, want %q", got.Name, f1.Name)
	}

	// Update should update index
	f1.Name = "Updated Feature"
	if err := repo.Update(ctx, f1); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify we can still get it
	got, err = repo.Get(ctx, f1.ID)
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if got.Name != "Updated Feature" {
		t.Errorf("Get() after update name = %q, want %q", got.Name, "Updated Feature")
	}

	// Delete should remove from index
	if err := repo.Delete(ctx, f1.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Should not be found anymore
	_, err = repo.Get(ctx, f1.ID)
	if err != fogit.ErrNotFound {
		t.Errorf("Get() after delete error = %v, want ErrNotFound", err)
	}
}

func TestFileRepository_IndexFallback(t *testing.T) {
	tmpDir := t.TempDir()
	fogitDir := filepath.Join(tmpDir, ".fogit")
	featuresDir := filepath.Join(fogitDir, "features")

	// Create features directory
	if err := os.MkdirAll(featuresDir, 0755); err != nil {
		t.Fatalf("Failed to create features dir: %v", err)
	}

	// Create a feature file directly (bypassing repository/index)
	f1 := fogit.NewFeature("Direct Feature")
	if err := WriteFeatureFile(filepath.Join(featuresDir, "direct-feature.yml"), f1); err != nil {
		t.Fatalf("WriteFeatureFile() error = %v", err)
	}

	// Create repository - should auto-rebuild index
	repo := NewFileRepository(fogitDir)
	ctx := t.Context()

	// Get should find it via fallback scan and update index
	got, err := repo.Get(ctx, f1.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name != f1.Name {
		t.Errorf("Get() name = %q, want %q", got.Name, f1.Name)
	}
}

package storage

import (
	"testing"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestMarshalFeature(t *testing.T) {
	tests := []struct {
		name    string
		feature *fogit.Feature
		wantErr bool
	}{
		{
			name:    "valid feature",
			feature: fogit.NewFeature("Test Feature"),
			wantErr: false,
		},
		{
			name:    "nil feature",
			feature: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := MarshalFeature(tt.feature)
			if tt.wantErr {
				if err == nil {
					t.Errorf("MarshalFeature() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("MarshalFeature() unexpected error: %v", err)
				}
				if len(data) == 0 {
					t.Errorf("MarshalFeature() returned empty data")
				}
			}
		})
	}
}

func TestUnmarshalFeature(t *testing.T) {
	// Create a valid feature for testing
	validFeature := fogit.NewFeature("Test Feature")
	validFeature.Description = "Test description"
	validData, _ := MarshalFeature(validFeature)

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "valid data",
			data:    validData,
			wantErr: false,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "invalid yaml",
			data:    []byte("invalid: yaml: data: ["),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feature, err := UnmarshalFeature(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Errorf("UnmarshalFeature() expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("UnmarshalFeature() unexpected error: %v", err)
				}
				if feature == nil {
					t.Errorf("UnmarshalFeature() returned nil feature")
				}
			}
		})
	}
}

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	original := fogit.NewFeature("Round Trip Test")
	original.Description = "Testing serialization round trip"
	original.SetType("bug")
	original.UpdateState(fogit.StateInProgress)
	original.SetPriority(fogit.PriorityHigh)
	original.Tags = []string{"test", "serialization"}

	// Marshal
	data, err := MarshalFeature(original)
	if err != nil {
		t.Fatalf("MarshalFeature() failed: %v", err)
	}

	// Unmarshal
	restored, err := UnmarshalFeature(data)
	if err != nil {
		t.Fatalf("UnmarshalFeature() failed: %v", err)
	}

	// Verify key fields
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: got %v, want %v", restored.ID, original.ID)
	}
	if restored.Name != original.Name {
		t.Errorf("Name mismatch: got %v, want %v", restored.Name, original.Name)
	}
	if restored.Description != original.Description {
		t.Errorf("Description mismatch: got %v, want %v", restored.Description, original.Description)
	}
	if restored.GetType() != original.GetType() {
		t.Errorf("Type mismatch: got %v, want %v", restored.GetType(), original.GetType())
	}
	if restored.DeriveState() != original.DeriveState() {
		t.Errorf("State mismatch: got %v, want %v", restored.DeriveState(), original.DeriveState())
	}
}

func TestYAML_EdgeCases(t *testing.T) {
	t.Run("unicode in all text fields", func(t *testing.T) {
		feature := fogit.NewFeature("用户认证 🚀")
		feature.Description = "描述：Unicode测试\n日本語テスト\n한국어 테스트"
		feature.SetType("软件功能")
		feature.SetCategory("категория")
		feature.SetDomain("ドメイン")
		feature.SetTeam("команда")
		feature.SetEpic("史诗")
		feature.SetModule("モジュール")
		feature.Tags = []string{"標籤", "タグ", "тег", "🏷️"}
		feature.Files = []string{"файл.go", "ファイル.ts", "文件.py"}

		data, err := MarshalFeature(feature)
		if err != nil {
			t.Fatalf("MarshalFeature() with unicode failed: %v", err)
		}

		restored, err := UnmarshalFeature(data)
		if err != nil {
			t.Fatalf("UnmarshalFeature() failed: %v", err)
		}

		if restored.Name != feature.Name {
			t.Errorf("Unicode name not preserved: got %v, want %v", restored.Name, feature.Name)
		}
		if restored.Description != feature.Description {
			t.Errorf("Unicode description not preserved")
		}
		if len(restored.Tags) != len(feature.Tags) {
			t.Errorf("Unicode tags not preserved")
		}
	})

	t.Run("special yaml characters", func(t *testing.T) {
		feature := fogit.NewFeature("Feature: with YAML special chars")
		feature.Description = "Description with colons: like this, dashes - and more -, quotes \"double\" and 'single', percent 100% complete, ampersand Tom & Jerry, asterisk *important*, hash #hashtag, at user@email.com"
		feature.Metadata = map[string]interface{}{
			"key:with:colons":  "value",
			"key-with-dashes":  "value",
			"key with spaces":  "value",
			"key.with.dots":    "value",
			"key/with/slashes": "value",
		}

		data, err := MarshalFeature(feature)
		if err != nil {
			t.Fatalf("MarshalFeature() with special chars failed: %v", err)
		}

		restored, err := UnmarshalFeature(data)
		if err != nil {
			t.Fatalf("UnmarshalFeature() failed: %v", err)
		}

		if restored.Name != feature.Name {
			t.Errorf("Name with special chars not preserved")
		}
		if restored.Description != feature.Description {
			t.Errorf("Description with special chars not preserved: got %q, want %q", restored.Description, feature.Description)
		}
	})

	t.Run("extremely long strings", func(t *testing.T) {
		longString := ""
		for i := 0; i < 1000; i++ {
			longString += "This is line " + string(rune('0'+i%10)) + " of a very long description. "
		}

		feature := fogit.NewFeature("Feature with long content")
		feature.Description = longString

		data, err := MarshalFeature(feature)
		if err != nil {
			t.Fatalf("MarshalFeature() with long string failed: %v", err)
		}

		restored, err := UnmarshalFeature(data)
		if err != nil {
			t.Fatalf("UnmarshalFeature() failed: %v", err)
		}

		if restored.Description != feature.Description {
			t.Errorf("Long description not preserved")
		}
	})

	t.Run("complex nested metadata", func(t *testing.T) {
		feature := fogit.NewFeature("Feature with complex metadata")
		feature.Metadata = map[string]interface{}{
			"level1": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": map[string]interface{}{
						"level4": "deep value",
						"array":  []interface{}{1, 2, 3, "four", true, nil},
					},
				},
			},
			"mixed_array": []interface{}{
				"string",
				42,
				3.14,
				true,
				nil,
				map[string]interface{}{"nested": "object"},
				[]interface{}{"nested", "array"},
			},
			"empty_map":   map[string]interface{}{},
			"empty_array": []interface{}{},
			"null_value":  nil,
		}

		data, err := MarshalFeature(feature)
		if err != nil {
			t.Fatalf("MarshalFeature() with complex metadata failed: %v", err)
		}

		restored, err := UnmarshalFeature(data)
		if err != nil {
			t.Fatalf("UnmarshalFeature() failed: %v", err)
		}

		if restored.Metadata == nil {
			t.Fatalf("Metadata is nil")
		}
		if restored.Metadata["level1"] == nil {
			t.Errorf("Nested metadata not preserved")
		}
	})

	t.Run("all relationship types", func(t *testing.T) {
		feature := fogit.NewFeature("Feature with all relationships")
		feature.Relationships = []fogit.Relationship{
			{Type: "blocks", TargetID: "id1", Description: "Blocks feature 1"},
			{Type: "blocked-by", TargetID: "id2", Description: "Blocked by feature 2"},
			{Type: "relates-to", TargetID: "id3", Description: "Related to feature 3"},
			{Type: "parent", TargetID: "id4", Description: "Parent feature"},
			{Type: "child", TargetID: "id5", Description: "Child feature"},
			{Type: "depends-on", TargetID: "id6", Description: "Depends on feature 6"},
			{Type: "duplicate-of", TargetID: "id7", Description: "Duplicate of feature 7"},
			{Type: "contains", TargetID: "id8", Description: "Contains feature 8"},
			{Type: "contained-by", TargetID: "id9", Description: "Contained by feature 9"},
			{Type: "implements", TargetID: "id10", Description: "Implements spec 10"},
			{Type: "implemented-by", TargetID: "id11", Description: "Implemented by feature 11"},
		}

		data, err := MarshalFeature(feature)
		if err != nil {
			t.Fatalf("MarshalFeature() with all relationships failed: %v", err)
		}

		restored, err := UnmarshalFeature(data)
		if err != nil {
			t.Fatalf("UnmarshalFeature() failed: %v", err)
		}

		if len(restored.Relationships) != 11 {
			t.Errorf("Relationships count mismatch: got %d, want 11", len(restored.Relationships))
		}
	})

	t.Run("empty arrays and slices", func(t *testing.T) {
		feature := fogit.NewFeature("Feature with empty collections")
		feature.Tags = []string{}
		feature.Files = []string{}
		feature.Relationships = []fogit.Relationship{}

		data, err := MarshalFeature(feature)
		if err != nil {
			t.Fatalf("MarshalFeature() failed: %v", err)
		}

		restored, err := UnmarshalFeature(data)
		if err != nil {
			t.Fatalf("UnmarshalFeature() failed: %v", err)
		}

		// Empty slices get unmarshaled as nil, which is acceptable
		if len(restored.Tags) != 0 {
			t.Errorf("Empty tags should have length 0, got %d", len(restored.Tags))
		}
		if len(restored.Files) != 0 {
			t.Errorf("Empty files should have length 0, got %d", len(restored.Files))
		}
		if len(restored.Relationships) != 0 {
			t.Errorf("Empty relationships should have length 0, got %d", len(restored.Relationships))
		}
	})

	t.Run("whitespace variations", func(t *testing.T) {
		feature := fogit.NewFeature("  Feature with whitespace  ")
		feature.Description = "\n\n  Multiple newlines\n  and spaces  \n\n"
		feature.Tags = []string{"tag1", "tag2", "tag3"}

		data, err := MarshalFeature(feature)
		if err != nil {
			t.Fatalf("MarshalFeature() failed: %v", err)
		}

		restored, err := UnmarshalFeature(data)
		if err != nil {
			t.Fatalf("UnmarshalFeature() failed: %v", err)
		}

		if restored.Name != feature.Name {
			t.Errorf("Whitespace in name not preserved: got %q, want %q", restored.Name, feature.Name)
		}
		if restored.Description != feature.Description {
			t.Errorf("Whitespace in description not preserved: got %q, want %q", restored.Description, feature.Description)
		}
	})
}

package commands

import (
	"testing"
)

// TestListCommandFlags tests flag configuration for the list command
func TestListCommandFlags(t *testing.T) {
	// Verify important flags exist
	flags := []string{"state", "priority", "type", "category", "domain", "team", "epic", "tag", "format", "sort"}

	for _, flag := range flags {
		t.Run("has "+flag+" flag", func(t *testing.T) {
			f := listCmd.Flag(flag)
			if f == nil {
				t.Errorf("listCmd should have --%s flag", flag)
			}
		})
	}
}

// Note: Storage and filtering tests are covered in:
// - internal/storage/repository_test.go (TestFileRepository_List, TestFileRepository_ListEmpty)
// - pkg/fogit/filter_test.go (TestFilter_Matches, TestFilter_Matches_EdgeCases)
// - pkg/fogit/sort_test.go (TestSortFeatures)
// - internal/printer/list_test.go (TestIsValidFormat)

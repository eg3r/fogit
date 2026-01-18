package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/pkg/fogit"
)

// OutputTree prints a hierarchical tree of features
func OutputTree(w io.Writer, root *fogit.Feature, allFeatures []*fogit.Feature, hierarchyTypes []string, maxDepth int) error {
	return displayTreeRecursive(w, root, allFeatures, hierarchyTypes, 0, maxDepth)
}

func displayTreeRecursive(w io.Writer, feature *fogit.Feature, allFeatures []*fogit.Feature, hierarchyTypes []string, currentDepth int, maxDepth int) error {
	// Check depth limit
	if maxDepth >= 0 && currentDepth > maxDepth {
		return nil
	}

	// Display current feature
	indent := strings.Repeat("  ", currentDepth)
	prefix := "├─"
	if currentDepth == 0 {
		prefix = ""
	}

	// Format feature info
	info := fmt.Sprintf("%s %s", feature.Name, feature.DeriveState())
	if priority := feature.GetPriority(); priority != "" && priority != fogit.PriorityMedium {
		info += fmt.Sprintf(" [%s]", priority)
	}
	if category := feature.GetCategory(); category != "" {
		info += fmt.Sprintf(" (%s)", category)
	}

	fmt.Fprintf(w, "%s%s %s\n", indent, prefix, info)

	// Find children
	children := features.FindChildren(feature.ID, allFeatures, hierarchyTypes)

	// Display children recursively
	for _, child := range children {
		if err := displayTreeRecursive(w, child, allFeatures, hierarchyTypes, currentDepth+1, maxDepth); err != nil {
			return err
		}
	}

	return nil
}

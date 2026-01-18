package printer

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/eg3r/fogit/internal/features"
)

// OutputStats prints repository statistics
func OutputStats(w io.Writer, stats *features.RepoStats, details bool) error {
	fmt.Fprintf(w, "Repository Statistics\n")
	fmt.Fprintf(w, "=====================\n\n")

	fmt.Fprintf(w, "Total Features: %d\n", stats.TotalFeatures)
	fmt.Fprintf(w, "Total Files:    %d\n", stats.TotalFiles)
	fmt.Fprintf(w, "Total Tags:     %d (%d unique)\n", stats.TotalTags, len(stats.UniqueTags))
	fmt.Fprintf(w, "Average Age:    %s\n", roundDuration(stats.AvgAge))
	fmt.Fprintf(w, "\n")

	// By State
	fmt.Fprintf(w, "By State:\n")
	printMapStats(w, stats.ByState, stats.TotalFeatures)
	fmt.Fprintf(w, "\n")

	// By Priority
	fmt.Fprintf(w, "By Priority:\n")
	printMapStats(w, stats.ByPriority, stats.TotalFeatures)
	fmt.Fprintf(w, "\n")

	// Relationships
	fmt.Fprintf(w, "Relationships:\n")
	fmt.Fprintf(w, "  Total:   %d\n", stats.TotalRelations)
	fmt.Fprintf(w, "  Average: %.1f per feature\n", stats.AvgRelations)
	fmt.Fprintf(w, "  Impact:  %d (blocking/critical)\n", stats.ImpactRelations)

	if details {
		fmt.Fprintf(w, "\nBy Type:\n")
		printStringMapStats(w, stats.ByType, stats.TotalFeatures)

		fmt.Fprintf(w, "\nBy Category:\n")
		printStringMapStats(w, stats.ByCategory, stats.TotalFeatures)

		fmt.Fprintf(w, "\nBy Domain:\n")
		printStringMapStats(w, stats.ByDomain, stats.TotalFeatures)

		fmt.Fprintf(w, "\nBy Team:\n")
		printStringMapStats(w, stats.ByTeam, stats.TotalFeatures)

		fmt.Fprintf(w, "\nBy Epic:\n")
		printStringMapStats(w, stats.ByEpic, stats.TotalFeatures)

		fmt.Fprintf(w, "\nRelationship Types:\n")
		printStringMapStats(w, stats.ByRelationType, stats.TotalRelations)

		fmt.Fprintf(w, "\nRelationship Categories:\n")
		printStringMapStats(w, stats.ByRelationCat, stats.TotalRelations)

		fmt.Fprintf(w, "\nTop Tags:\n")
		printTopTags(w, stats.UniqueTags, 5)
	}

	return nil
}

func roundDuration(d time.Duration) time.Duration {
	return d.Round(time.Hour)
}

func printMapStats[K comparable](w io.Writer, m map[K]int, total int) {
	if total == 0 {
		return
	}

	// Convert to slice for sorting
	type kv struct {
		Key   K
		Value int
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}

	// Sort by value (descending)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	for _, item := range sorted {
		percent := float64(item.Value) / float64(total) * 100
		fmt.Fprintf(w, "  %-12v %d (%.1f%%)\n", item.Key, item.Value, percent)
	}
}

func printStringMapStats(w io.Writer, m map[string]int, total int) {
	if total == 0 {
		return
	}

	// Convert to slice for sorting
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}

	// Sort by value (descending)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	for _, item := range sorted {
		percent := float64(item.Value) / float64(total) * 100
		fmt.Fprintf(w, "  %-12s %d (%.1f%%)\n", item.Key, item.Value, percent)
	}
}

func printTopTags(w io.Writer, m map[string]int, limit int) {
	type kv struct {
		Key   string
		Value int
	}
	var sorted []kv
	for k, v := range m {
		sorted = append(sorted, kv{k, v})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Value > sorted[j].Value
	})

	if len(sorted) > limit {
		sorted = sorted[:limit]
	}

	for _, item := range sorted {
		fmt.Fprintf(w, "  %-12s %d\n", item.Key, item.Value)
	}
}

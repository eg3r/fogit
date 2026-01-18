package features

import (
	"fmt"
	"time"

	"github.com/eg3r/fogit/pkg/fogit"
)

// RepoStats contains statistics about the repository
type RepoStats struct {
	TotalFeatures   int
	ByState         map[fogit.State]int
	ByPriority      map[fogit.Priority]int
	ByType          map[string]int
	ByCategory      map[string]int
	ByDomain        map[string]int
	ByTeam          map[string]int
	ByEpic          map[string]int
	TotalTags       int
	UniqueTags      map[string]int
	TotalFiles      int
	AvgAge          time.Duration
	AvgRelations    float64
	TotalRelations  int
	ByRelationType  map[string]int // Count by relationship type
	ByRelationCat   map[string]int // Count by relationship category
	ImpactRelations int            // Count of relationships in categories with include_in_impact: true
}

// CalculateStats computes statistics for a list of features
func CalculateStats(features []*fogit.Feature, cfg *fogit.Config) *RepoStats {
	stats := &RepoStats{
		ByState:        make(map[fogit.State]int),
		ByPriority:     make(map[fogit.Priority]int),
		ByType:         make(map[string]int),
		ByCategory:     make(map[string]int),
		ByDomain:       make(map[string]int),
		ByTeam:         make(map[string]int),
		ByEpic:         make(map[string]int),
		UniqueTags:     make(map[string]int),
		ByRelationType: make(map[string]int),
		ByRelationCat:  make(map[string]int),
	}

	stats.TotalFeatures = len(features)
	now := time.Now()
	var totalAge time.Duration

	for _, f := range features {
		// Count by state (derived from timestamps)
		stats.ByState[f.DeriveState()]++

		// Count by priority (from metadata)
		stats.ByPriority[f.GetPriority()]++

		// Count by type (from metadata)
		if fType := f.GetType(); fType != "" {
			stats.ByType[fType]++
		}

		// Count by category (from metadata)
		if fCategory := f.GetCategory(); fCategory != "" {
			stats.ByCategory[fCategory]++
		}

		// Count by domain (from metadata)
		if fDomain := f.GetDomain(); fDomain != "" {
			stats.ByDomain[fDomain]++
		}

		// Count by team (from metadata)
		if fTeam := f.GetTeam(); fTeam != "" {
			stats.ByTeam[fTeam]++
		}

		// Count by epic (from metadata)
		if fEpic := f.GetEpic(); fEpic != "" {
			stats.ByEpic[fEpic]++
		}

		// Count tags
		stats.TotalTags += len(f.Tags)
		for _, tag := range f.Tags {
			stats.UniqueTags[tag]++
		}

		// Count files
		stats.TotalFiles += len(f.Files)

		// Count relationships with category awareness
		stats.TotalRelations += len(f.Relationships)
		for _, rel := range f.Relationships {
			// Count by type
			relTypeStr := string(rel.Type)
			stats.ByRelationType[relTypeStr]++

			// Determine category
			category := "other"
			if cfg != nil {
				category = rel.GetCategory(cfg)
			}
			stats.ByRelationCat[category]++

			// Check impact
			if cfg != nil {
				if catConfig, ok := cfg.Relationships.Categories[category]; ok {
					if catConfig.IncludeInImpact {
						stats.ImpactRelations++
					}
				}
			}
		}

		// Calculate age
		age := now.Sub(f.GetCreatedAt())
		totalAge += age
	}

	if stats.TotalFeatures > 0 {
		stats.AvgAge = totalAge / time.Duration(stats.TotalFeatures)
		stats.AvgRelations = float64(stats.TotalRelations) / float64(stats.TotalFeatures)
	}

	return stats
}

// FormatTimeAgo formats a time as a human-readable "time ago" string
func FormatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
	if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / (24 * 7))
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}

	return t.Format("Jan 2, 2006")
}

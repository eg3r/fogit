package fogit

import (
	"sort"
)

// SortFeatures sorts a slice of features according to the filter criteria.
func SortFeatures(features []*Feature, filter *Filter) {
	if filter == nil || filter.SortBy == "" {
		return
	}

	sortFunc := getSortFunc(filter.SortBy, filter.SortOrder)
	sort.Slice(features, sortFunc(features))
}

// getSortFunc returns the appropriate sort function based on the sort field and order.
func getSortFunc(sortBy SortField, sortOrder SortOrder) func([]*Feature) func(i, j int) bool {
	ascending := sortOrder == "" || sortOrder == SortAscending

	return func(features []*Feature) func(i, j int) bool {
		switch sortBy {
		case SortByName:
			return func(i, j int) bool {
				if ascending {
					return features[i].Name < features[j].Name
				}
				return features[i].Name > features[j].Name
			}
		case SortByPriority:
			return func(i, j int) bool {
				// Priority order: critical > high > medium > low
				priorityOrder := map[Priority]int{
					PriorityCritical: 4,
					PriorityHigh:     3,
					PriorityMedium:   2,
					PriorityLow:      1,
				}
				iPriority := priorityOrder[features[i].GetPriority()]
				jPriority := priorityOrder[features[j].GetPriority()]
				if ascending {
					return iPriority < jPriority
				}
				return iPriority > jPriority
			}
		case SortByCreated:
			return func(i, j int) bool {
				if ascending {
					return features[i].GetCreatedAt().Before(features[j].GetCreatedAt())
				}
				return features[i].GetCreatedAt().After(features[j].GetCreatedAt())
			}
		case SortByModified:
			return func(i, j int) bool {
				if ascending {
					return features[i].GetModifiedAt().Before(features[j].GetModifiedAt())
				}
				return features[i].GetModifiedAt().After(features[j].GetModifiedAt())
			}
		default:
			// Default: sort by created date descending
			return func(i, j int) bool {
				return features[i].GetCreatedAt().After(features[j].GetCreatedAt())
			}
		}
	}
}

package fogit

import (
	"testing"
)

func TestSortField_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		field SortField
		want  bool
	}{
		{"name", SortByName, true},
		{"priority", SortByPriority, true},
		{"created", SortByCreated, true},
		{"modified", SortByModified, true},
		{"empty", "", true},
		{"invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.field.IsValid(); got != tt.want {
				t.Errorf("SortField.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortOrder_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		order SortOrder
		want  bool
	}{
		{"ascending", SortAscending, true},
		{"descending", SortDescending, true},
		{"empty", "", true},
		{"invalid", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.order.IsValid(); got != tt.want {
				t.Errorf("SortOrder.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  Filter
		wantErr bool
		errMsg  error
	}{
		{
			name:    "valid empty filter",
			filter:  Filter{},
			wantErr: false,
		},
		{
			name: "valid with state",
			filter: Filter{
				State: StateOpen,
			},
			wantErr: false,
		},
		{
			name: "valid with priority",
			filter: Filter{
				Priority: PriorityHigh,
			},
			wantErr: false,
		},
		{
			name: "valid with sort",
			filter: Filter{
				SortBy:    SortByName,
				SortOrder: SortAscending,
			},
			wantErr: false,
		},
		{
			name: "invalid state",
			filter: Filter{
				State: "invalid-state",
			},
			wantErr: true,
			errMsg:  ErrInvalidState,
		},
		{
			name: "invalid priority",
			filter: Filter{
				Priority: "invalid-priority",
			},
			wantErr: true,
			errMsg:  ErrInvalidPriority,
		},
		{
			name: "invalid sort field",
			filter: Filter{
				SortBy: "invalid-field",
			},
			wantErr: true,
			errMsg:  ErrInvalidSortField,
		},
		{
			name: "invalid sort order",
			filter: Filter{
				SortOrder: "invalid-order",
			},
			wantErr: true,
			errMsg:  ErrInvalidSortOrder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Filter.Validate() expected error but got nil")
					return
				}
				if err != tt.errMsg {
					t.Errorf("Filter.Validate() error = %v, want %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Filter.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestFilter_Matches(t *testing.T) {
	baseFeature := func() *Feature {
		f := NewFeature("Test Feature")
		f.ID = "test-id"
		f.SetType("software-feature")
		f.SetPriority(PriorityHigh)
		f.SetCategory("authentication")
		f.SetDomain("backend")
		f.SetTeam("security-team")
		f.SetEpic("user-management")
		f.SetModule("auth-service")
		f.Relationships = []Relationship{
			{
				Type:     "parent",
				TargetID: "parent-id",
			},
		}
		return f
	}()

	tests := []struct {
		name    string
		filter  Filter
		feature *Feature
		want    bool
	}{
		{
			name:    "empty filter matches everything",
			filter:  Filter{},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "state match",
			filter: Filter{
				State: StateOpen,
			},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "state no match",
			filter: Filter{
				State: StateClosed,
			},
			feature: baseFeature,
			want:    false,
		},
		{
			name: "priority match",
			filter: Filter{
				Priority: PriorityHigh,
			},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "priority no match",
			filter: Filter{
				Priority: PriorityLow,
			},
			feature: baseFeature,
			want:    false,
		},
		{
			name: "type match case insensitive",
			filter: Filter{
				Type: "SOFTWARE-FEATURE",
			},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "type no match",
			filter: Filter{
				Type: "bug-fix",
			},
			feature: baseFeature,
			want:    false,
		},
		{
			name: "category match",
			filter: Filter{
				Category: "Authentication",
			},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "domain match",
			filter: Filter{
				Domain: "BACKEND",
			},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "team match",
			filter: Filter{
				Team: "security-TEAM",
			},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "epic match",
			filter: Filter{
				Epic: "User-Management",
			},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "parent match",
			filter: Filter{
				Parent: "parent-id",
			},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "parent no match",
			filter: Filter{
				Parent: "other-parent-id",
			},
			feature: baseFeature,
			want:    false,
		},
		{
			name: "multiple filters all match",
			filter: Filter{
				State:    StateOpen,
				Priority: PriorityHigh,
				Category: "authentication",
			},
			feature: baseFeature,
			want:    true,
		},
		{
			name: "multiple filters one no match",
			filter: Filter{
				State:    StateOpen,
				Priority: PriorityLow, // Does not match
				Category: "authentication",
			},
			feature: baseFeature,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(tt.feature); got != tt.want {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFilter(t *testing.T) {
	filter := NewFilter()
	if filter == nil {
		t.Fatal("NewFilter() returned nil")
	}
	if filter.SortBy != SortByCreated {
		t.Errorf("NewFilter().SortBy = %v, want %v", filter.SortBy, SortByCreated)
	}
	if filter.SortOrder != SortDescending {
		t.Errorf("NewFilter().SortOrder = %v, want %v", filter.SortOrder, SortDescending)
	}
}

// Helper to create test features with proper initialization
func makeTestFeature(name string, opts ...func(*Feature)) *Feature {
	f := NewFeature(name)
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// Additional edge case tests

func TestFilter_Matches_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		filter  Filter
		feature *Feature
		want    bool
	}{
		{
			name:   "empty filter matches nil arrays",
			filter: Filter{},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.SetPriority(PriorityMedium)
				f.Tags = nil
			}),
			want: true,
		},
		{
			name: "filter with all possible fields",
			filter: Filter{
				State:    StateOpen,
				Priority: PriorityHigh,
				Type:     "software-feature",
				Category: "auth",
				Domain:   "backend",
				Team:     "security",
				Epic:     "onboarding",
				Parent:   "parent-id",
			},
			feature: makeTestFeature("Complete", func(f *Feature) {
				f.SetPriority(PriorityHigh)
				f.SetType("software-feature")
				f.SetCategory("auth")
				f.SetDomain("backend")
				f.SetTeam("security")
				f.SetEpic("onboarding")
				f.Relationships = []Relationship{
					{Type: "parent", TargetID: "parent-id"},
				}
			}),
			want: true,
		},
		{
			name:   "case insensitive type matching",
			filter: Filter{Type: "Software-Feature"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.SetPriority(PriorityMedium)
				f.SetType("software-feature")
			}),
			want: true,
		},
		{
			name:   "filter with unicode",
			filter: Filter{Category: "认证"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.SetPriority(PriorityMedium)
				f.SetCategory("认证")
			}),
			want: true,
		},
		{
			name:   "multiple parent relationships",
			filter: Filter{Parent: "parent-2"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.SetPriority(PriorityMedium)
				f.Relationships = []Relationship{
					{Type: "parent", TargetID: "parent-1"},
					{Type: "parent", TargetID: "parent-2"},
					{Type: "depends-on", TargetID: "dep-1"},
				}
			}),
			want: true,
		},
		{
			name:   "no parent relationships",
			filter: Filter{Parent: "parent-1"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.SetPriority(PriorityMedium)
				f.Relationships = []Relationship{
					{Type: "depends-on", TargetID: "dep-1"},
				}
			}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(tt.feature); got != tt.want {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Matches_Tags(t *testing.T) {
	tests := []struct {
		name    string
		filter  Filter
		feature *Feature
		want    bool
	}{
		{
			name:   "single tag match",
			filter: Filter{Tags: []string{"security"}},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Tags = []string{"security", "auth"}
			}),
			want: true,
		},
		{
			name:   "single tag no match",
			filter: Filter{Tags: []string{"admin"}},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Tags = []string{"security", "auth"}
			}),
			want: false,
		},
		{
			name:   "multiple tags AND logic - all match",
			filter: Filter{Tags: []string{"security", "auth"}},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Tags = []string{"security", "auth", "backend"}
			}),
			want: true,
		},
		{
			name:   "multiple tags AND logic - partial match",
			filter: Filter{Tags: []string{"security", "admin"}},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Tags = []string{"security", "auth"}
			}),
			want: false,
		},
		{
			name:   "tag case insensitive",
			filter: Filter{Tags: []string{"SECURITY"}},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Tags = []string{"security"}
			}),
			want: true,
		},
		{
			name:   "empty filter tags matches all",
			filter: Filter{Tags: []string{}},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Tags = []string{"security"}
			}),
			want: true,
		},
		{
			name:   "filter tags on feature with no tags",
			filter: Filter{Tags: []string{"security"}},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Tags = nil
			}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(tt.feature); got != tt.want {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Matches_Contributor(t *testing.T) {
	tests := []struct {
		name    string
		filter  Filter
		feature *Feature
		want    bool
	}{
		{
			name:   "contributor exact match",
			filter: Filter{Contributor: "alice@example.com"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Versions = map[string]*FeatureVersion{
					"1": {Authors: []string{"alice@example.com"}},
				}
			}),
			want: true,
		},
		{
			name:   "contributor no match",
			filter: Filter{Contributor: "bob@example.com"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Versions = map[string]*FeatureVersion{
					"1": {Authors: []string{"alice@example.com"}},
				}
			}),
			want: false,
		},
		{
			name:   "contributor case insensitive",
			filter: Filter{Contributor: "ALICE@EXAMPLE.COM"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Versions = map[string]*FeatureVersion{
					"1": {Authors: []string{"alice@example.com"}},
				}
			}),
			want: true,
		},
		{
			name:   "contributor partial match",
			filter: Filter{Contributor: "alice"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Versions = map[string]*FeatureVersion{
					"1": {Authors: []string{"alice@example.com"}},
				}
			}),
			want: true,
		},
		{
			name:   "contributor in multiple versions",
			filter: Filter{Contributor: "bob@example.com"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Versions = map[string]*FeatureVersion{
					"1": {Authors: []string{"alice@example.com"}},
					"2": {Authors: []string{"bob@example.com"}},
				}
			}),
			want: true,
		},
		{
			name:   "contributor with multiple authors per version",
			filter: Filter{Contributor: "charlie"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Versions = map[string]*FeatureVersion{
					"1": {Authors: []string{"alice@example.com", "charlie@example.com"}},
				}
			}),
			want: true,
		},
		{
			name:   "contributor on feature with no versions",
			filter: Filter{Contributor: "alice@example.com"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Versions = nil
			}),
			want: false,
		},
		{
			name:   "contributor on version with empty authors",
			filter: Filter{Contributor: "alice@example.com"},
			feature: makeTestFeature("Test", func(f *Feature) {
				f.Versions = map[string]*FeatureVersion{
					"1": {Authors: []string{}},
				}
			}),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(tt.feature); got != tt.want {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		filter  Filter
		wantErr bool
	}{
		{
			name:    "completely empty filter",
			filter:  Filter{},
			wantErr: false,
		},
		{
			name: "invalid state with valid other fields",
			filter: Filter{
				State:    State("invalid"),
				Priority: PriorityHigh,
				SortBy:   SortByName,
			},
			wantErr: true,
		},
		{
			name: "invalid priority with valid other fields",
			filter: Filter{
				State:    StateOpen,
				Priority: Priority("invalid"),
				SortBy:   SortByName,
			},
			wantErr: true,
		},
		{
			name: "invalid sort field only",
			filter: Filter{
				SortBy: SortField("invalid"),
			},
			wantErr: true,
		},
		{
			name: "invalid sort order only",
			filter: Filter{
				SortOrder: SortOrder("invalid"),
			},
			wantErr: true,
		},
		{
			name: "all fields valid",
			filter: Filter{
				State:     StateOpen,
				Priority:  PriorityHigh,
				Type:      "feature",
				Category:  "cat",
				Domain:    "domain",
				Team:      "team",
				Epic:      "epic",
				Parent:    "parent",
				SortBy:    SortByName,
				SortOrder: SortAscending,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Filter.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

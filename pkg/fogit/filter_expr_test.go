package fogit

import (
	"testing"
	"time"
)

func TestParseFilterExpr_Empty(t *testing.T) {
	expr, err := ParseFilterExpr("")
	if err != nil {
		t.Fatalf("ParseFilterExpr() error = %v", err)
	}

	// Empty expression should match everything
	feature := NewFeature("Test")
	if !expr.Matches(feature) {
		t.Error("Empty expression should match all features")
	}
}

func TestParseFilterExpr_SimpleField(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		setup      func() *Feature
		want       bool
	}{
		{
			name:       "state equals open",
			expression: "state:open",
			setup: func() *Feature {
				f := NewFeature("Test")
				// NewFeature creates in open state
				return f
			},
			want: true,
		},
		{
			name:       "state not matching",
			expression: "state:closed",
			setup: func() *Feature {
				f := NewFeature("Test")
				return f
			},
			want: false,
		},
		{
			name:       "priority shorthand",
			expression: "priority:high",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "metadata.priority explicit",
			expression: "metadata.priority:high",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "category shorthand",
			expression: "category:authentication",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetCategory("authentication")
				return f
			},
			want: true,
		},
		{
			name:       "case insensitive match",
			expression: "category:AUTHENTICATION",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetCategory("authentication")
				return f
			},
			want: true,
		},
		{
			name:       "name field quoted",
			expression: `name:"Test Feature"`,
			setup: func() *Feature {
				f := NewFeature("Test Feature")
				return f
			},
			want: true,
		},
		{
			name:       "name field simple",
			expression: "name:TestFeature",
			setup: func() *Feature {
				f := NewFeature("TestFeature")
				return f
			},
			want: true,
		},
		{
			name:       "tags contains",
			expression: "tags:security",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.Tags = []string{"auth", "security", "backend"}
				return f
			},
			want: true,
		},
		{
			name:       "tags not contains",
			expression: "tags:frontend",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.Tags = []string{"auth", "security", "backend"}
				return f
			},
			want: false,
		},
		{
			name:       "domain shorthand",
			expression: "domain:backend",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetDomain("backend")
				return f
			},
			want: true,
		},
		{
			name:       "team shorthand",
			expression: "team:security-team",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetTeam("security-team")
				return f
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseFilterExpr(tt.expression)
			if err != nil {
				t.Fatalf("ParseFilterExpr() error = %v", err)
			}

			feature := tt.setup()
			if got := expr.Matches(feature); got != tt.want {
				t.Errorf("Matches() = %v, want %v for expression %q", got, tt.want, tt.expression)
			}
		})
	}
}

func TestParseFilterExpr_Wildcards(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		setup      func() *Feature
		want       bool
	}{
		{
			name:       "name contains wildcard",
			expression: "name:*auth*",
			setup: func() *Feature {
				return NewFeature("User Authentication")
			},
			want: true,
		},
		{
			name:       "name starts with wildcard",
			expression: "name:User*",
			setup: func() *Feature {
				return NewFeature("User Authentication")
			},
			want: true,
		},
		{
			name:       "name ends with wildcard",
			expression: "name:*tion",
			setup: func() *Feature {
				return NewFeature("User Authentication")
			},
			want: true,
		},
		{
			name:       "wildcard no match",
			expression: "name:*login*",
			setup: func() *Feature {
				return NewFeature("User Authentication")
			},
			want: false,
		},
		{
			name:       "wildcard case insensitive",
			expression: "name:*AUTH*",
			setup: func() *Feature {
				return NewFeature("User Authentication")
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseFilterExpr(tt.expression)
			if err != nil {
				t.Fatalf("ParseFilterExpr() error = %v", err)
			}

			feature := tt.setup()
			if got := expr.Matches(feature); got != tt.want {
				t.Errorf("Matches() = %v, want %v for expression %q", got, tt.want, tt.expression)
			}
		})
	}
}

func TestParseFilterExpr_LogicalOperators(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		setup      func() *Feature
		want       bool
	}{
		{
			name:       "AND both true",
			expression: "priority:high AND state:open",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "AND one false",
			expression: "priority:high AND state:closed",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: false,
		},
		{
			name:       "OR both true",
			expression: "priority:high OR state:open",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "OR one true",
			expression: "priority:low OR state:open",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "OR both false",
			expression: "priority:low OR state:closed",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: false,
		},
		{
			name:       "NOT true becomes false",
			expression: "NOT state:open",
			setup: func() *Feature {
				return NewFeature("Test")
			},
			want: false,
		},
		{
			name:       "NOT false becomes true",
			expression: "NOT state:closed",
			setup: func() *Feature {
				return NewFeature("Test")
			},
			want: true,
		},
		{
			name:       "multiple AND",
			expression: "priority:high AND state:open AND category:auth",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				f.SetCategory("auth")
				return f
			},
			want: true,
		},
		{
			name:       "AND OR precedence",
			expression: "priority:high AND state:open OR category:security",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityLow)
				f.SetCategory("security")
				return f
			},
			want: true, // (priority:high AND state:open) OR category:security
		},
		{
			name:       "lowercase and",
			expression: "priority:high and state:open",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "lowercase or",
			expression: "priority:high or state:closed",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseFilterExpr(tt.expression)
			if err != nil {
				t.Fatalf("ParseFilterExpr() error = %v", err)
			}

			feature := tt.setup()
			if got := expr.Matches(feature); got != tt.want {
				t.Errorf("Matches() = %v, want %v for expression %q", got, tt.want, tt.expression)
			}
		})
	}
}

func TestParseFilterExpr_Grouping(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		setup      func() *Feature
		want       bool
	}{
		{
			name:       "simple grouping",
			expression: "(priority:high)",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "grouping changes precedence",
			expression: "priority:high AND (state:closed OR category:security)",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				f.SetCategory("security")
				return f
			},
			want: true,
		},
		{
			name:       "nested grouping",
			expression: "((priority:high OR priority:critical) AND state:open)",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityCritical)
				return f
			},
			want: true,
		},
		{
			name:       "complex grouping from spec",
			expression: "(priority:high OR priority:critical) AND state:open AND category:security",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityCritical)
				f.SetCategory("security")
				return f
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseFilterExpr(tt.expression)
			if err != nil {
				t.Fatalf("ParseFilterExpr() error = %v", err)
			}

			feature := tt.setup()
			if got := expr.Matches(feature); got != tt.want {
				t.Errorf("Matches() = %v, want %v for expression %q", got, tt.want, tt.expression)
			}
		})
	}
}

func TestParseFilterExpr_Comparisons(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		setup      func() *Feature
		want       bool
	}{
		{
			name:       "priority greater than",
			expression: "priority:>medium",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "priority greater equal",
			expression: "priority:>=high",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "priority less than",
			expression: "priority:<high",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityMedium)
				return f
			},
			want: true,
		},
		{
			name:       "priority less equal",
			expression: "priority:<=medium",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityLow)
				return f
			},
			want: true,
		},
		{
			name:       "priority comparison false",
			expression: "priority:>high",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityMedium)
				return f
			},
			want: false,
		},
		{
			name:       "date greater than",
			expression: "created:>2020-01-01",
			setup: func() *Feature {
				f := NewFeature("Test")
				// NewFeature uses time.Now() which is after 2020-01-01
				return f
			},
			want: true,
		},
		{
			name:       "date less than future",
			expression: "created:<2099-12-31",
			setup: func() *Feature {
				f := NewFeature("Test")
				return f
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseFilterExpr(tt.expression)
			if err != nil {
				t.Fatalf("ParseFilterExpr() error = %v", err)
			}

			feature := tt.setup()
			if got := expr.Matches(feature); got != tt.want {
				t.Errorf("Matches() = %v, want %v for expression %q", got, tt.want, tt.expression)
			}
		})
	}
}

func TestParseFilterExpr_DateComparisons(t *testing.T) {
	// Create feature with known date
	createFeatureWithDate := func(dateStr string) *Feature {
		f := NewFeature("Test")
		date, _ := time.Parse("2006-01-02", dateStr)
		if v := f.GetCurrentVersion(); v != nil {
			v.CreatedAt = date
			v.ModifiedAt = date
		}
		return f
	}

	tests := []struct {
		name       string
		expression string
		setup      func() *Feature
		want       bool
	}{
		{
			name:       "created after date",
			expression: "created:>2025-06-01",
			setup:      func() *Feature { return createFeatureWithDate("2025-06-15") },
			want:       true,
		},
		{
			name:       "created before date",
			expression: "created:<2025-06-01",
			setup:      func() *Feature { return createFeatureWithDate("2025-05-15") },
			want:       true,
		},
		{
			name:       "created on or after",
			expression: "created:>=2025-06-01",
			setup:      func() *Feature { return createFeatureWithDate("2025-06-01") },
			want:       true,
		},
		{
			name:       "created on or before",
			expression: "created:<=2025-06-01",
			setup:      func() *Feature { return createFeatureWithDate("2025-06-01") },
			want:       true,
		},
		{
			name:       "modified after date",
			expression: "modified:>2025-01-01",
			setup:      func() *Feature { return createFeatureWithDate("2025-06-15") },
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseFilterExpr(tt.expression)
			if err != nil {
				t.Fatalf("ParseFilterExpr() error = %v", err)
			}

			feature := tt.setup()
			if got := expr.Matches(feature); got != tt.want {
				t.Errorf("Matches() = %v, want %v for expression %q", got, tt.want, tt.expression)
			}
		})
	}
}

func TestParseFilterExpr_QuotedValues(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		setup      func() *Feature
		want       bool
	}{
		{
			name:       "quoted value with spaces",
			expression: `name:"User Authentication"`,
			setup: func() *Feature {
				return NewFeature("User Authentication")
			},
			want: true,
		},
		{
			name:       "quoted value not matching",
			expression: `name:"Other Feature"`,
			setup: func() *Feature {
				return NewFeature("User Authentication")
			},
			want: false,
		},
		{
			name:       "escaped quotes in value",
			expression: `description:"say \"hello\""`,
			setup: func() *Feature {
				f := NewFeature("Test")
				f.Description = `say "hello"`
				return f
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseFilterExpr(tt.expression)
			if err != nil {
				t.Fatalf("ParseFilterExpr() error = %v", err)
			}

			feature := tt.setup()
			if got := expr.Matches(feature); got != tt.want {
				t.Errorf("Matches() = %v, want %v for expression %q", got, tt.want, tt.expression)
			}
		})
	}
}

func TestParseFilterExpr_Errors(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "unmatched open paren",
			expression: "(priority:high",
			wantErr:    true,
		},
		{
			name:       "missing value",
			expression: "priority:",
			wantErr:    true,
		},
		{
			name:       "missing colon",
			expression: "priority high",
			wantErr:    true,
		},
		{
			name:       "unterminated quote",
			expression: `name:"test`,
			wantErr:    true,
		},
		{
			name:       "extra tokens",
			expression: "priority:high extra",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilterExpr(tt.expression)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilterExpr() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseFilterExpr_SpecExamples(t *testing.T) {
	// Test examples from spec/specification/08-interface.md
	tests := []struct {
		name       string
		expression string
		setup      func() *Feature
		want       bool
	}{
		{
			name:       "spec example 1: priority AND state",
			expression: "priority:high AND state:open",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "spec example 2: explicit metadata",
			expression: "metadata.priority:high AND state:open",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetPriority(PriorityHigh)
				return f
			},
			want: true,
		},
		{
			name:       "spec example 3: category OR",
			expression: "metadata.category:authentication OR metadata.category:security",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetCategory("authentication")
				return f
			},
			want: true,
		},
		{
			name:       "spec example 4: date AND team",
			expression: "created:>2025-01-01 AND metadata.team:backend",
			setup: func() *Feature {
				f := NewFeature("Test")
				f.SetTeam("backend")
				// created at time.Now() which is > 2025-01-01
				return f
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := ParseFilterExpr(tt.expression)
			if err != nil {
				t.Fatalf("ParseFilterExpr() error = %v", err)
			}

			feature := tt.setup()
			if got := expr.Matches(feature); got != tt.want {
				t.Errorf("Matches() = %v, want %v for expression %q", got, tt.want, tt.expression)
			}
		})
	}
}

func TestFieldExpr_String(t *testing.T) {
	tests := []struct {
		expr *FieldExpr
		want string
	}{
		{
			expr: &FieldExpr{Field: "state", Operator: OpEquals, Value: "open"},
			want: "state:open",
		},
		{
			expr: &FieldExpr{Field: "created", Operator: OpGreater, Value: "2025-01-01"},
			want: "created:>2025-01-01",
		},
		{
			expr: &FieldExpr{Field: "priority", Operator: OpGreaterEqual, Value: "medium"},
			want: "priority:>=medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.expr.String(); got != tt.want {
				t.Errorf("FieldExpr.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

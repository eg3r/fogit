package fogit

import (
	"testing"
)

func TestVersionConstraint_IsSimpleVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  interface{}
		expected bool
	}{
		{"integer", 2, true},
		{"int64", int64(3), true},
		{"float64", float64(4), true},
		{"string integer", "5", true},
		{"semver string", "1.0.0", false},
		{"invalid string", "abc", false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := &VersionConstraint{Operator: ">=", Version: tt.version}
			if got := vc.IsSimpleVersion(); got != tt.expected {
				t.Errorf("IsSimpleVersion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVersionConstraint_IsSemanticVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  interface{}
		expected bool
	}{
		{"valid semver", "1.0.0", true},
		{"valid semver with minor", "1.2.3", true},
		{"valid semver large numbers", "10.20.30", true},
		{"integer", 2, false},
		{"string integer", "5", false},
		{"invalid semver", "1.0", false},
		{"invalid semver with v", "v1.0.0", false},
		{"invalid string", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := &VersionConstraint{Operator: ">=", Version: tt.version}
			if got := vc.IsSemanticVersion(); got != tt.expected {
				t.Errorf("IsSemanticVersion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVersionConstraint_GetVersionString(t *testing.T) {
	tests := []struct {
		name     string
		version  interface{}
		expected string
	}{
		{"integer", 2, "2"},
		{"int64", int64(3), "3"},
		{"float64", float64(4), "4"},
		{"string integer", "5", "5"},
		{"semver string", "1.0.0", "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := &VersionConstraint{Operator: ">=", Version: tt.version}
			if got := vc.GetVersionString(); got != tt.expected {
				t.Errorf("GetVersionString() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVersionConstraint_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		operator  string
		version   interface{}
		wantError bool
	}{
		// Valid simple versions
		{"valid simple >=", ">=", 2, false},
		{"valid simple >", ">", 1, false},
		{"valid simple <", "<", 3, false},
		{"valid simple <=", "<=", 4, false},
		{"valid simple =", "=", 5, false},

		// Valid semantic versions
		{"valid semver >=", ">=", "1.0.0", false},
		{"valid semver >", ">", "1.2.3", false},
		{"valid semver =", "=", "2.0.0", false},

		// Invalid operators
		{"invalid operator !=", "!=", 2, true},
		{"invalid operator <>", "<>", 2, true},
		{"invalid operator empty", "", 2, true},

		// Invalid versions
		{"invalid version 0", ">=", 0, true},
		{"invalid version -1", ">=", -1, true},
		{"invalid version string", ">=", "abc", true},
		{"invalid semver format", ">=", "1.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := &VersionConstraint{Operator: tt.operator, Version: tt.version}
			err := vc.IsValid()
			if (err != nil) != tt.wantError {
				t.Errorf("IsValid() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestVersionConstraint_IsSatisfiedBy_Simple(t *testing.T) {
	tests := []struct {
		name          string
		operator      string
		version       int
		targetVersion string
		expected      bool
	}{
		// >= operator
		{">=2 satisfied by 2", ">=", 2, "2", true},
		{">=2 satisfied by 3", ">=", 2, "3", true},
		{">=2 not satisfied by 1", ">=", 2, "1", false},

		// > operator
		{">2 satisfied by 3", ">", 2, "3", true},
		{">2 not satisfied by 2", ">", 2, "2", false},
		{">2 not satisfied by 1", ">", 2, "1", false},

		// <= operator
		{"<=2 satisfied by 2", "<=", 2, "2", true},
		{"<=2 satisfied by 1", "<=", 2, "1", true},
		{"<=2 not satisfied by 3", "<=", 2, "3", false},

		// < operator
		{"<2 satisfied by 1", "<", 2, "1", true},
		{"<2 not satisfied by 2", "<", 2, "2", false},
		{"<2 not satisfied by 3", "<", 2, "3", false},

		// = operator
		{"=2 satisfied by 2", "=", 2, "2", true},
		{"=2 not satisfied by 1", "=", 2, "1", false},
		{"=2 not satisfied by 3", "=", 2, "3", false},

		// Simple constraint against semver target (extracts major version)
		{">=2 satisfied by 2.0.0", ">=", 2, "2.0.0", true},
		{">=2 satisfied by 2.1.0", ">=", 2, "2.1.0", true},
		{">=2 satisfied by 3.0.0", ">=", 2, "3.0.0", true},
		{">=2 not satisfied by 1.9.9", ">=", 2, "1.9.9", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := &VersionConstraint{Operator: tt.operator, Version: tt.version}
			if got := vc.IsSatisfiedBy(tt.targetVersion); got != tt.expected {
				t.Errorf("IsSatisfiedBy(%s) = %v, want %v", tt.targetVersion, got, tt.expected)
			}
		})
	}
}

func TestVersionConstraint_IsSatisfiedBy_Semantic(t *testing.T) {
	tests := []struct {
		name          string
		operator      string
		version       string
		targetVersion string
		expected      bool
	}{
		// >= operator
		{">=1.0.0 satisfied by 1.0.0", ">=", "1.0.0", "1.0.0", true},
		{">=1.0.0 satisfied by 1.0.1", ">=", "1.0.0", "1.0.1", true},
		{">=1.0.0 satisfied by 1.1.0", ">=", "1.0.0", "1.1.0", true},
		{">=1.0.0 satisfied by 2.0.0", ">=", "1.0.0", "2.0.0", true},
		{">=1.1.0 not satisfied by 1.0.9", ">=", "1.1.0", "1.0.9", false},

		// > operator
		{">1.0.0 satisfied by 1.0.1", ">", "1.0.0", "1.0.1", true},
		{">1.0.0 satisfied by 1.1.0", ">", "1.0.0", "1.1.0", true},
		{">1.0.0 not satisfied by 1.0.0", ">", "1.0.0", "1.0.0", false},
		{">1.0.0 not satisfied by 0.9.9", ">", "1.0.0", "0.9.9", false},

		// <= operator
		{"<=1.0.0 satisfied by 1.0.0", "<=", "1.0.0", "1.0.0", true},
		{"<=1.0.0 satisfied by 0.9.9", "<=", "1.0.0", "0.9.9", true},
		{"<=1.0.0 not satisfied by 1.0.1", "<=", "1.0.0", "1.0.1", false},

		// < operator
		{"<1.0.0 satisfied by 0.9.9", "<", "1.0.0", "0.9.9", true},
		{"<1.0.0 not satisfied by 1.0.0", "<", "1.0.0", "1.0.0", false},
		{"<1.0.0 not satisfied by 1.0.1", "<", "1.0.0", "1.0.1", false},

		// = operator
		{"=1.0.0 satisfied by 1.0.0", "=", "1.0.0", "1.0.0", true},
		{"=1.0.0 not satisfied by 1.0.1", "=", "1.0.0", "1.0.1", false},
		{"=1.0.0 not satisfied by 0.9.9", "=", "1.0.0", "0.9.9", false},

		// Semver constraint against simple target (treated as x.0.0)
		{">=1.0.0 satisfied by simple 1", ">=", "1.0.0", "1", true},
		{">=1.0.0 satisfied by simple 2", ">=", "1.0.0", "2", true},
		{">1.0.0 satisfied by simple 2", ">", "1.0.0", "2", true},
		{">1.0.0 not satisfied by simple 1", ">", "1.0.0", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := &VersionConstraint{Operator: tt.operator, Version: tt.version}
			if got := vc.IsSatisfiedBy(tt.targetVersion); got != tt.expected {
				t.Errorf("IsSatisfiedBy(%s) = %v, want %v", tt.targetVersion, got, tt.expected)
			}
		})
	}
}

func TestVersionConstraint_NilConstraint(t *testing.T) {
	var vc *VersionConstraint = nil

	// IsValid should not panic and return nil for nil constraint
	if err := vc.IsValid(); err != nil {
		t.Errorf("IsValid() on nil should return nil, got %v", err)
	}

	// IsSatisfiedBy should return true for nil constraint (any version allowed)
	if !vc.IsSatisfiedBy("1.0.0") {
		t.Error("IsSatisfiedBy() on nil should return true")
	}

	// Getter methods should not panic
	if vc.IsSimpleVersion() {
		t.Error("IsSimpleVersion() on nil should return false")
	}
	if vc.IsSemanticVersion() {
		t.Error("IsSemanticVersion() on nil should return false")
	}
	if vc.GetSimpleVersion() != 0 {
		t.Error("GetSimpleVersion() on nil should return 0")
	}
	if vc.GetSemanticVersion() != "" {
		t.Error("GetSemanticVersion() on nil should return empty string")
	}
	if vc.GetVersionString() != "" {
		t.Error("GetVersionString() on nil should return empty string")
	}
}

func TestExtractMajorVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"1", 1},
		{"2", 2},
		{"10", 10},
		{"1.0.0", 1},
		{"2.1.0", 2},
		{"10.5.3", 10},
		{"invalid", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := extractMajorVersion(tt.input); got != tt.expected {
				t.Errorf("extractMajorVersion(%s) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseSemverParts(t *testing.T) {
	tests := []struct {
		input    string
		expected [3]int
	}{
		{"1.0.0", [3]int{1, 0, 0}},
		{"1.2.3", [3]int{1, 2, 3}},
		{"10.20.30", [3]int{10, 20, 30}},
		{"1", [3]int{1, 0, 0}}, // Simple integer becomes x.0.0
		{"2", [3]int{2, 0, 0}},
		{"invalid", [3]int{0, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseSemverParts(tt.input); got != tt.expected {
				t.Errorf("parseSemverParts(%s) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCompareSemverParts(t *testing.T) {
	tests := []struct {
		name     string
		a        [3]int
		b        [3]int
		expected int
	}{
		{"equal", [3]int{1, 0, 0}, [3]int{1, 0, 0}, 0},
		{"a > b (major)", [3]int{2, 0, 0}, [3]int{1, 0, 0}, 1},
		{"a < b (major)", [3]int{1, 0, 0}, [3]int{2, 0, 0}, -1},
		{"a > b (minor)", [3]int{1, 2, 0}, [3]int{1, 1, 0}, 1},
		{"a < b (minor)", [3]int{1, 1, 0}, [3]int{1, 2, 0}, -1},
		{"a > b (patch)", [3]int{1, 0, 2}, [3]int{1, 0, 1}, 1},
		{"a < b (patch)", [3]int{1, 0, 1}, [3]int{1, 0, 2}, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareSemverParts(tt.a, tt.b); got != tt.expected {
				t.Errorf("compareSemverParts(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

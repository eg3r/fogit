package printer

import (
	"bytes"
	"strings"
	"testing"

	"github.com/eg3r/fogit/pkg/fogit"
)

func TestIsValidFormat(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{"table", true},
		{"json", true},
		{"csv", true},
		{"xml", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := IsValidFormat(tt.format); got != tt.want {
			t.Errorf("IsValidFormat(%q) = %v, want %v", tt.format, got, tt.want)
		}
	}
}

func TestHasActiveFilters(t *testing.T) {
	tests := []struct {
		name   string
		filter *fogit.Filter
		want   bool
	}{
		{"empty", &fogit.Filter{}, false},
		{"state", &fogit.Filter{State: "open"}, true},
		{"priority", &fogit.Filter{Priority: "high"}, true},
		{"type", &fogit.Filter{Type: "bug"}, true},
		{"category", &fogit.Filter{Category: "backend"}, true},
		{"domain", &fogit.Filter{Domain: "api"}, true},
		{"team", &fogit.Filter{Team: "core"}, true},
		{"epic", &fogit.Filter{Epic: "v1"}, true},
		{"parent", &fogit.Filter{Parent: "123"}, true},
		{"tags", &fogit.Filter{Tags: []string{"security"}}, true},
		{"contributor", &fogit.Filter{Contributor: "alice@example.com"}, true},
		{"empty_tags", &fogit.Filter{Tags: []string{}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasActiveFilters(tt.filter); got != tt.want {
				t.Errorf("HasActiveFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutputJSON(t *testing.T) {
	f := fogit.NewFeature("Test")
	f.ID = "1"
	features := []*fogit.Feature{f}
	var buf bytes.Buffer
	if err := OutputJSON(&buf, features); err != nil {
		t.Fatalf("OutputJSON() error = %v", err)
	}
	if !strings.Contains(buf.String(), `"ID": "1"`) {
		t.Errorf("OutputJSON() = %q, want ID: 1", buf.String())
	}
}

func TestOutputCSV(t *testing.T) {
	f := fogit.NewFeature("Test")
	f.ID = "1"
	features := []*fogit.Feature{f}
	var buf bytes.Buffer
	if err := OutputCSV(&buf, features); err != nil {
		t.Fatalf("OutputCSV() error = %v", err)
	}
	if !strings.Contains(buf.String(), "ID,Name") {
		t.Error("OutputCSV() missing header")
	}
	if !strings.Contains(buf.String(), "1,Test") {
		t.Error("OutputCSV() missing data")
	}
}

func TestOutputTable(t *testing.T) {
	f := fogit.NewFeature("Test Feature")
	f.ID = "1"
	features := []*fogit.Feature{f}
	var buf bytes.Buffer
	if err := OutputTable(&buf, features); err != nil {
		t.Fatalf("OutputTable() error = %v", err)
	}
	if !strings.Contains(buf.String(), "NAME") {
		t.Error("OutputTable() missing header")
	}
	if !strings.Contains(buf.String(), "Test Feature") {
		t.Error("OutputTable() missing data")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"exact length", 12, "exact length"},
		{"too long string", 10, "too lon..."},
	}

	for _, tt := range tests {
		if got := truncate(tt.s, tt.max); got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
		}
	}
}

package printer

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/eg3r/fogit/pkg/fogit"
)

// TestIsValidShowFormat tests format validation
func TestIsValidShowFormat(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{"text", true},
		{"json", true},
		{"yaml", true},
		{"xml", false},
		{"csv", false},
		{"", false},
		{"TEXT", false}, // case-sensitive
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			got := IsValidShowFormat(tt.format)
			if got != tt.want {
				t.Errorf("IsValidShowFormat(%q) = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}

// TestOutputFeatureText tests text format output
func TestOutputFeatureText(t *testing.T) {
	feature := fogit.NewFeature("Test Feature")
	feature.Description = "Test description"
	feature.SetType("test-type")
	feature.SetPriority(fogit.PriorityHigh)
	feature.SetCategory("test-category")
	feature.Tags = []string{"tag1", "tag2"}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := OutputFeatureText(os.Stdout, feature, false, false)

	w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("OutputFeatureText() error = %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check that essential fields are present
	requiredFields := []string{
		"ID:",
		"Name:",
		"Test Feature",
		"Description:",
		"Type:",
		"State:",
		"Priority:",
		"Category:",
		"Tags:",
		"Created:",
	}

	for _, field := range requiredFields {
		if !strings.Contains(output, field) {
			t.Errorf("output missing required field: %q", field)
		}
	}
}

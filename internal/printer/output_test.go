package printer

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestIsValidOutputFormat(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{"text", true},
		{"json", true},
		{"yaml", true},
		{"tree", true},
		{"csv", false},
		{"xml", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := IsValidOutputFormat(tt.format)
			if result != tt.valid {
				t.Errorf("IsValidOutputFormat(%q) = %v, want %v", tt.format, result, tt.valid)
			}
		})
	}
}

func TestOutputAsJSON(t *testing.T) {
	data := map[string]string{"key": "value"}
	var buf bytes.Buffer

	err := OutputAsJSON(&buf, data)
	if err != nil {
		t.Fatalf("OutputAsJSON() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"key"`) || !strings.Contains(output, `"value"`) {
		t.Errorf("OutputAsJSON() = %q, want JSON with key and value", output)
	}
}

func TestOutputAsYAML(t *testing.T) {
	data := map[string]string{"key": "value"}
	var buf bytes.Buffer

	err := OutputAsYAML(&buf, data)
	if err != nil {
		t.Fatalf("OutputAsYAML() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "key:") || !strings.Contains(output, "value") {
		t.Errorf("OutputAsYAML() = %q, want YAML with key and value", output)
	}
}

func TestOutputFormatted(t *testing.T) {
	data := map[string]string{"name": "test"}

	tests := []struct {
		name     string
		format   string
		wantJSON bool
		wantYAML bool
		wantText bool
		wantErr  bool
	}{
		{"json format", "json", true, false, false, false},
		{"yaml format", "yaml", false, true, false, false},
		{"text format", "text", false, false, true, false},
		{"invalid format", "invalid", false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			textCalled := false

			err := OutputFormatted(&buf, tt.format, data, func(w io.Writer) error {
				textCalled = true
				return nil
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("OutputFormatted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantJSON && !strings.Contains(buf.String(), `"name"`) {
				t.Errorf("OutputFormatted() expected JSON output")
			}
			if tt.wantYAML && !strings.Contains(buf.String(), "name:") {
				t.Errorf("OutputFormatted() expected YAML output")
			}
			if tt.wantText && !textCalled {
				t.Errorf("OutputFormatted() expected text function to be called")
			}
		})
	}
}

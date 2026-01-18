package common

import (
	"testing"
)

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		substr string
		want   bool
	}{
		{"exact match", "Hello World", "Hello", true},
		{"case insensitive", "Hello World", "hello", true},
		{"upper case search", "hello world", "WORLD", true},
		{"no match", "Hello World", "foo", false},
		{"empty substr", "Hello World", "", true},
		{"empty text", "", "Hello", false},
		{"both empty", "", "", true},
		{"partial match", "Hello World", "lo Wo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsIgnoreCase(tt.text, tt.substr); got != tt.want {
				t.Errorf("ContainsIgnoreCase(%q, %q) = %v, want %v", tt.text, tt.substr, got, tt.want)
			}
		})
	}
}

func TestSplitKeyValue(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		sep       string
		wantKey   string
		wantValue string
	}{
		{"basic", "key=value", "=", "key", "value"},
		{"with spaces", " key = value ", "=", "key", "value"},
		{"multiple separators", "key=val=ue", "=", "key", "val=ue"},
		{"no separator", "keyvalue", "=", "keyvalue", ""},
		{"empty value", "key=", "=", "key", ""},
		{"empty key", "=value", "=", "", "value"},
		{"custom separator", "key:value", ":", "key", "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value := SplitKeyValue(tt.input, tt.sep)
			if key != tt.wantKey || value != tt.wantValue {
				t.Errorf("SplitKeyValue(%q, %q) = (%q, %q), want (%q, %q)",
					tt.input, tt.sep, key, value, tt.wantKey, tt.wantValue)
			}
		})
	}
}

func TestSplitKeyValueEquals(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string
		wantValue string
	}{
		{"basic", "key=value", "key", "value"},
		{"with spaces", " foo = bar ", "foo", "bar"},
		{"no value", "key", "key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, value := SplitKeyValueEquals(tt.input)
			if key != tt.wantKey || value != tt.wantValue {
				t.Errorf("SplitKeyValueEquals(%q) = (%q, %q), want (%q, %q)",
					tt.input, key, value, tt.wantKey, tt.wantValue)
			}
		})
	}
}

func TestTruncateWithEllipsis(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{"no truncation needed", "Hello", 10, "Hello"},
		{"exact length", "Hello", 5, "Hello"},
		{"truncate with ellipsis", "Hello World", 8, "Hello..."},
		{"very short max", "Hello", 3, "Hel"},
		{"zero length", "Hello", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateWithEllipsis(tt.text, tt.maxLen); got != tt.want {
				t.Errorf("TruncateWithEllipsis(%q, %d) = %q, want %q",
					tt.text, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestIsBlank(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty", "", true},
		{"spaces only", "   ", true},
		{"tabs and spaces", " \t\n ", true},
		{"not blank", "hello", false},
		{"spaces around text", " hi ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBlank(tt.input); got != tt.want {
				t.Errorf("IsBlank(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCoalesce(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"first non-empty", []string{"", "hello", "world"}, "hello"},
		{"all empty", []string{"", "", ""}, ""},
		{"first is non-empty", []string{"first", "second"}, "first"},
		{"single value", []string{"only"}, "only"},
		{"no values", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Coalesce(tt.values...); got != tt.want {
				t.Errorf("Coalesce(%v) = %q, want %q", tt.values, got, tt.want)
			}
		})
	}
}

func TestGetSnippet(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		query  string
		maxLen int
		want   string
	}{
		{"short text no match", "Hello", "x", 100, "Hello"},
		{"text shorter than max", "Hello World", "World", 50, "Hello World"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetSnippet(tt.text, tt.query, tt.maxLen)
			if got != tt.want {
				t.Errorf("GetSnippet(%q, %q, %d) = %q, want %q",
					tt.text, tt.query, tt.maxLen, got, tt.want)
			}
		})
	}
}

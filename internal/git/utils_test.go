package git

import (
	"testing"
)

func TestParseAuthor(t *testing.T) {
	tests := []struct {
		name      string
		authorStr string
		wantName  string
		wantEmail string
	}{
		{
			name:      "name and email",
			authorStr: "John Doe <john@example.com>",
			wantName:  "John Doe",
			wantEmail: "john@example.com",
		},
		{
			name:      "email only",
			authorStr: "john@example.com",
			wantName:  "john@example.com",
			wantEmail: "john@example.com",
		},
		{
			name:      "empty",
			authorStr: "",
			wantName:  "",
			wantEmail: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAuthor(tt.authorStr)
			if tt.authorStr == "" {
				if got != nil {
					t.Errorf("ParseAuthor(%q) = %v, want nil", tt.authorStr, got)
				}
				return
			}

			if got == nil {
				t.Fatalf("ParseAuthor(%q) returned nil", tt.authorStr)
			}

			if got.Name != tt.wantName {
				t.Errorf("ParseAuthor(%q).Name = %q, want %q", tt.authorStr, got.Name, tt.wantName)
			}
			if got.Email != tt.wantEmail {
				t.Errorf("ParseAuthor(%q).Email = %q, want %q", tt.authorStr, got.Email, tt.wantEmail)
			}
		})
	}
}

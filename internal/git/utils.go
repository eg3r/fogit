package git

import (
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
)

// ParseAuthor parses an author string in the format "Name <email>" or just "email"
func ParseAuthor(authorStr string) *object.Signature {
	if authorStr == "" {
		return nil
	}

	var name, email string
	parts := strings.Split(authorStr, "<")
	if len(parts) == 2 {
		name = strings.TrimSpace(parts[0])
		email = strings.TrimSuffix(strings.TrimSpace(parts[1]), ">")
	} else {
		// Just email or name
		name = authorStr
		email = authorStr
	}

	return &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}
}

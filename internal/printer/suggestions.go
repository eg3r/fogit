package printer

import (
	"fmt"
	"io"

	"github.com/eg3r/fogit/internal/search"
)

// PrintSuggestions prints "Did you mean?" suggestions for a missing feature
func PrintSuggestions(w io.Writer, identifier string, suggestions []search.Match, commandExample string) {
	fmt.Fprintf(w, "Feature not found: %q\n\n", identifier)
	fmt.Fprintf(w, "⚠ Did you mean:\n")
	for i, match := range suggestions {
		fmt.Fprintf(w, "  %d. %s (%.0f%% match)\n", i+1, match.Feature.Name, match.Score)
		fmt.Fprintf(w, "     ID: %s\n", match.Feature.ID)
	}
	if commandExample != "" {
		fmt.Fprintf(w, "\nTry: %s\n", commandExample)
	}
}

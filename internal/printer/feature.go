package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/eg3r/fogit/pkg/fogit"
)

// IsValidShowFormat checks if the output format is supported
func IsValidShowFormat(format string) bool {
	return format == "text" || format == "json" || format == "yaml"
}

// OutputFeatureJSON writes feature as JSON to the writer
func OutputFeatureJSON(w io.Writer, feature *fogit.Feature) error {
	return OutputAsJSON(w, feature)
}

// OutputFeatureYAML writes feature as YAML to the writer
func OutputFeatureYAML(w io.Writer, feature *fogit.Feature) error {
	return OutputAsYAML(w, feature)
}

// OutputFeatureText writes feature as formatted text to the writer
func OutputFeatureText(w io.Writer, feature *fogit.Feature, showRels, showVers bool) error {
	fmt.Fprintf(w, "ID:          %s\n", feature.ID)
	fmt.Fprintf(w, "Name:        %s\n", feature.Name)

	if feature.Description != "" {
		fmt.Fprintf(w, "Description: %s\n", feature.Description)
	}

	if fType := feature.GetType(); fType != "" {
		fmt.Fprintf(w, "Type:        %s\n", fType)
	}

	fmt.Fprintf(w, "State:       %s\n", feature.DeriveState())
	if priority := feature.GetPriority(); priority != "" {
		fmt.Fprintf(w, "Priority:    %s\n", priority)
	}

	// Organization fields (from metadata)
	if category := feature.GetCategory(); category != "" {
		fmt.Fprintf(w, "Category:    %s\n", category)
	}
	if domain := feature.GetDomain(); domain != "" {
		fmt.Fprintf(w, "Domain:      %s\n", domain)
	}
	if team := feature.GetTeam(); team != "" {
		fmt.Fprintf(w, "Team:        %s\n", team)
	}
	if epic := feature.GetEpic(); epic != "" {
		fmt.Fprintf(w, "Epic:        %s\n", epic)
	}
	if module := feature.GetModule(); module != "" {
		fmt.Fprintf(w, "Module:      %s\n", module)
	}

	// Tags
	if len(feature.Tags) > 0 {
		fmt.Fprintf(w, "Tags:        %s\n", strings.Join(feature.Tags, ", "))
	}

	// Timestamps (from current version)
	createdAt := feature.GetCreatedAt()
	if !createdAt.IsZero() {
		fmt.Fprintf(w, "Created:     %s\n", createdAt.Format("2006-01-02 15:04:05 MST"))
	}
	modifiedAt := feature.GetModifiedAt()
	if !modifiedAt.IsZero() {
		fmt.Fprintf(w, "Modified:    %s\n", modifiedAt.Format("2006-01-02 15:04:05 MST"))
	}

	// Files
	if len(feature.Files) > 0 {
		fmt.Fprintf(w, "\nFiles:\n")
		for _, file := range feature.Files {
			fmt.Fprintf(w, "  - %s\n", file)
		}
	}

	// Relationships
	if showRels && len(feature.Relationships) > 0 {
		fmt.Fprintf(w, "\nRelationships:\n")
		for _, rel := range feature.Relationships {
			fmt.Fprintf(w, "  - %s: %s", rel.Type, rel.TargetID)
			if rel.Description != "" {
				fmt.Fprintf(w, " (%s)", rel.Description)
			}
			fmt.Fprintln(w)
		}
	}

	// Metadata
	if len(feature.Metadata) > 0 {
		fmt.Fprintf(w, "\nMetadata:\n")
		for key, value := range feature.Metadata {
			fmt.Fprintf(w, "  %s: %v\n", key, value)
		}
	}

	// Versions
	if showVers && len(feature.Versions) > 0 {
		fmt.Fprintf(w, "\nVersions:\n")
		for vID, v := range feature.Versions {
			fmt.Fprintf(w, "  Version %s: %v\n", vID, v)
		}
	}

	return nil
}

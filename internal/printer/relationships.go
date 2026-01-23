package printer

import (
	"fmt"
	"io"
	"strings"

	"github.com/eg3r/fogit/internal/features"
	"github.com/eg3r/fogit/pkg/fogit"
)

// OutputRelationshipsText prints relationships in text format
func OutputRelationshipsText(w io.Writer, feature *fogit.Feature, outgoing []fogit.Relationship, incoming []features.RelationshipWithSource) error {
	fmt.Fprintf(w, "Relationships for: %s\n\n", feature.Name)

	if len(outgoing) == 0 && len(incoming) == 0 {
		fmt.Fprintln(w, "No relationships found")
		return nil
	}

	if len(outgoing) > 0 {
		fmt.Fprintln(w, "Outgoing:")
		fmt.Fprintln(w, strings.Repeat("-", 80))
		for _, rel := range outgoing {
			// Use cached target name
			targetName := rel.TargetName
			if targetName == "" {
				targetName = rel.TargetID
			}

			// Safely format ID (may be empty)
			relID := rel.ID
			if len(relID) >= 8 {
				relID = relID[:8]
			} else if relID == "" {
				relID = "n/a"
			}

			fmt.Fprintf(w, "  [%s] %s -> %s\n", relID, rel.Type, targetName)
			if rel.Description != "" {
				fmt.Fprintf(w, "    Description: %s\n", rel.Description)
			}
			fmt.Fprintf(w, "    Created: %s\n", rel.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Fprintln(w)
	}

	if len(incoming) > 0 {
		fmt.Fprintln(w, "Incoming:")
		fmt.Fprintln(w, strings.Repeat("-", 80))
		for _, rel := range incoming {
			// Safely format ID (may be empty)
			relID := rel.Relation.ID
			if len(relID) >= 8 {
				relID = relID[:8]
			} else if relID == "" {
				relID = "n/a"
			}

			fmt.Fprintf(w, "  [%s] %s <- %s\n", relID, rel.Relation.Type, rel.SourceName)
			if rel.Relation.Description != "" {
				fmt.Fprintf(w, "    Description: %s\n", rel.Relation.Description)
			}
			fmt.Fprintf(w, "    Created: %s\n", rel.Relation.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}

	return nil
}

type relationshipsOutput struct {
	Feature  featureInfo            `json:"feature"`
	Outgoing []relationshipInfo     `json:"outgoing"`
	Incoming []relationshipWithInfo `json:"incoming"`
}

type featureInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type relationshipInfo struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	TargetID    string `json:"target_id"`
	TargetName  string `json:"target_name"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type relationshipWithInfo struct {
	SourceID   string           `json:"source_id"`
	SourceName string           `json:"source_name"`
	Relation   relationshipInfo `json:"relation"`
}

// OutputRelationshipsJSON prints relationships in JSON format
func OutputRelationshipsJSON(w io.Writer, feature *fogit.Feature, outgoing []fogit.Relationship, incoming []features.RelationshipWithSource) error {
	output := relationshipsOutput{
		Feature: featureInfo{
			ID:   feature.ID,
			Name: feature.Name,
		},
		Outgoing: make([]relationshipInfo, len(outgoing)),
		Incoming: make([]relationshipWithInfo, len(incoming)),
	}

	// Convert outgoing relationships
	for i, rel := range outgoing {
		output.Outgoing[i] = relationshipInfo{
			ID:          rel.ID,
			Type:        string(rel.Type),
			TargetID:    rel.TargetID,
			TargetName:  rel.TargetName,
			Description: rel.Description,
			CreatedAt:   rel.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	// Convert incoming relationships
	for i, rel := range incoming {
		output.Incoming[i] = relationshipWithInfo{
			SourceID:   rel.SourceID,
			SourceName: rel.SourceName,
			Relation: relationshipInfo{
				ID:          rel.Relation.ID,
				Type:        string(rel.Relation.Type),
				TargetID:    rel.Relation.TargetID,
				TargetName:  rel.Relation.TargetName,
				Description: rel.Relation.Description,
				CreatedAt:   rel.Relation.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			},
		}
	}

	return OutputAsJSON(w, output)
}

package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/eg3r/fogit/internal/search"
	"github.com/eg3r/fogit/pkg/fogit"
)

// IncomingRelationship represents a relationship from another feature pointing to the target
type IncomingRelationship struct {
	SourceID   string
	SourceName string
	Type       string
}

// Prompter defines the interface for user interaction
type Prompter interface {
	SelectFromSimilarFeatures(query string, matches []search.Match) (*fogit.Feature, bool, error)
	SelectVersionIncrement(currentVersion, versionFormat string) (fogit.VersionIncrement, error)
	Confirm(message string) (bool, error)
	ConfirmDeletion(feature *fogit.Feature, incomingRels []IncomingRelationship) (bool, error)
	ReadLine(prompt string) (string, error)
}

// StdPrompter implements Prompter using standard input/output
type StdPrompter struct {
	reader *bufio.Reader
}

// NewPrompter creates a new StdPrompter
func NewPrompter() *StdPrompter {
	return &StdPrompter{
		reader: bufio.NewReader(os.Stdin),
	}
}

// SelectFromSimilarFeatures prompts user to select from similar features or create new
// Returns: selected feature (if any), createNew flag, error
func (p *StdPrompter) SelectFromSimilarFeatures(query string, matches []search.Match) (*fogit.Feature, bool, error) {
	if len(matches) == 0 {
		return nil, true, nil // No matches, create new
	}

	// Show similar features found
	fmt.Printf("\nSimilar features found:\n")
	for i, match := range matches {
		matchState := match.Feature.DeriveState()
		stateLabel := formatStateLabel(matchState)
		fmt.Printf("  %d. %s %s - %.0f%% match\n", i+1, match.Feature.Name, stateLabel, match.Score)
	}

	// Prompt user
	fmt.Printf("\nWhat do you want to do?\n")
	for i := range matches {
		fmt.Printf("  [%d] Switch to \"%s\"\n", i+1, matches[i].Feature.Name)
	}
	fmt.Printf("  [n] Create new \"%s\" anyway\n", query)

	// Build prompt based on number of matches
	var promptStr string
	if len(matches) == 1 {
		promptStr = "\nEnter choice (1 or n): "
	} else {
		promptStr = fmt.Sprintf("\nEnter choice (1-%d or n): ", len(matches))
	}

	input, err := p.ReadLine(promptStr)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Handle user choice
	if input == "n" || input == "" {
		return nil, true, nil // Create new
	}

	// Try to parse as number
	var choice int
	_, err = fmt.Sscanf(input, "%d", &choice)
	if err == nil && choice >= 1 && choice <= len(matches) {
		return matches[choice-1].Feature, false, nil
	}

	// Invalid input, default to create new
	return nil, true, nil
}

// SelectVersionIncrement prompts user for version increment type
func (p *StdPrompter) SelectVersionIncrement(currentVersion, versionFormat string) (fogit.VersionIncrement, error) {
	if versionFormat == "semantic" {
		fmt.Printf("\nCreate new version (current: %s):\n", currentVersion)
		fmt.Printf("  [p] Patch (bug fixes, no API changes)\n")
		fmt.Printf("  [m] Minor (new features, backward compatible)\n")
		fmt.Printf("  [M] Major (breaking changes)\n")
		fmt.Printf("  [c] Cancel\n")
	} else {
		fmt.Printf("\nCreate new version (current: %s):\n", currentVersion)
		fmt.Printf("  [m] Minor (next version)\n")
		fmt.Printf("  [M] Major (next version, same as minor for simple versioning)\n")
		fmt.Printf("  [c] Cancel\n")
	}

	input, err := p.ReadLine("\nChoice [m]: ")
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Default to minor if empty
	if input == "" {
		input = "m"
	}

	switch input {
	case "p":
		if versionFormat != "semantic" {
			fmt.Println("Note: Patch version not applicable for simple versioning, using minor instead")
			return fogit.VersionIncrementMinor, nil
		}
		return fogit.VersionIncrementPatch, nil
	case "m":
		return fogit.VersionIncrementMinor, nil
	case "major":
		return fogit.VersionIncrementMajor, nil
	case "c":
		return "", fmt.Errorf("canceled")
	default:
		fmt.Printf("Invalid choice, using minor\n")
		return fogit.VersionIncrementMinor, nil
	}
}

// Confirm shows a yes/no confirmation prompt
func (p *StdPrompter) Confirm(message string) (bool, error) {
	input, err := p.ReadLine(fmt.Sprintf("%s (yes/no): ", message))
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	response := strings.TrimSpace(strings.ToLower(input))
	return response == "yes" || response == "y", nil
}

// ConfirmDeletion shows a deletion confirmation with feature details
func (p *StdPrompter) ConfirmDeletion(feature *fogit.Feature, incomingRels []IncomingRelationship) (bool, error) {
	fmt.Printf("Delete feature:\n")
	fmt.Printf("  ID:       %s\n", feature.ID)
	fmt.Printf("  Name:     %s\n", feature.Name)
	if feature.Description != "" {
		fmt.Printf("  Description: %s\n", feature.Description)
	}
	fmt.Printf("  State:    %s\n", feature.DeriveState())
	if priority := feature.GetPriority(); priority != "" {
		fmt.Printf("  Priority: %s\n", priority)
	}

	if len(incomingRels) > 0 {
		fmt.Printf("\n  Incoming relationships to be removed:\n")
		for _, ir := range incomingRels {
			fmt.Printf("    - %s [%s]\n", ir.SourceName, ir.Type)
		}
	}

	return p.Confirm("\nAre you sure you want to delete this feature?")
}

// ReadLine reads a single line from stdin with a prompt
func (p *StdPrompter) ReadLine(prompt string) (string, error) {
	fmt.Print(prompt)
	input, err := p.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

// formatStateLabel returns a formatted state label for display
func formatStateLabel(state fogit.State) string {
	switch state {
	case fogit.StateClosed:
		return "(closed)"
	case fogit.StateInProgress:
		return "(in progress)"
	default:
		return "(open)"
	}
}

// SelectFeature prompts user to select a feature from a list
func (p *StdPrompter) SelectFeature(features []*fogit.Feature, title string) (*fogit.Feature, error) {
	if len(features) == 0 {
		return nil, fmt.Errorf("no features to select from")
	}

	if len(features) == 1 {
		return features[0], nil
	}

	fmt.Printf("\n%s\n", title)
	for i, f := range features {
		state := f.DeriveState()
		fmt.Printf("  [%d] %s %s\n", i+1, f.Name, formatStateLabel(state))
	}

	input, err := p.ReadLine(fmt.Sprintf("\nSelect (1-%d): ", len(features)))
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	var choice int
	_, err = fmt.Sscanf(input, "%d", &choice)
	if err != nil || choice < 1 || choice > len(features) {
		return nil, fmt.Errorf("invalid selection")
	}

	return features[choice-1], nil
}

// InputText prompts for text input
func (p *StdPrompter) InputText(title, defaultValue string) (string, error) {
	var prompt string
	if defaultValue != "" {
		prompt = fmt.Sprintf("%s [%s]: ", title, defaultValue)
	} else {
		prompt = fmt.Sprintf("%s: ", title)
	}

	input, err := p.ReadLine(prompt)
	if err != nil {
		return "", err
	}

	if input == "" && defaultValue != "" {
		return defaultValue, nil
	}
	return input, nil
}

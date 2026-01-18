package printer

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/eg3r/fogit/pkg/fogit"
)

// IsValidFormat checks if the output format is supported
func IsValidFormat(format string) bool {
	return format == "table" || format == "json" || format == "csv"
}

// OutputJSON writes features as JSON to the writer
func OutputJSON(w io.Writer, features []*fogit.Feature) error {
	return OutputAsJSON(w, features)
}

// OutputCSV writes features as CSV to the writer
func OutputCSV(w io.Writer, features []*fogit.Feature) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "Name", "Type", "State", "Priority", "Category", "Domain", "Team", "Epic", "Created", "Updated"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, f := range features {
		row := []string{
			f.ID,
			f.Name,
			f.GetType(),
			string(f.DeriveState()),
			string(f.GetPriority()),
			f.GetCategory(),
			f.GetDomain(),
			f.GetTeam(),
			f.GetEpic(),
			f.GetCreatedAt().Format("2006-01-02 15:04:05"),
			f.GetModifiedAt().Format("2006-01-02 15:04:05"),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// OutputTable writes features as a formatted table to the writer
func OutputTable(w io.Writer, features []*fogit.Feature) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	// Write header
	fmt.Fprintln(tw, "NAME\tSTATE\tPRIORITY\tTYPE\tCATEGORY\tTEAM")
	fmt.Fprintln(tw, strings.Repeat("-", 80))

	// Write rows
	for _, f := range features {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			truncate(f.Name, 30),
			f.DeriveState(),
			f.GetPriority(),
			truncate(f.GetType(), 15),
			truncate(f.GetCategory(), 15),
			truncate(f.GetTeam(), 15))
	}

	return nil
}

// truncate shortens a string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// HasActiveFilters checks if any filter criteria are set
func HasActiveFilters(filter *fogit.Filter) bool {
	return filter.State != "" ||
		filter.Priority != "" ||
		filter.Type != "" ||
		filter.Category != "" ||
		filter.Domain != "" ||
		filter.Team != "" ||
		filter.Epic != "" ||
		filter.Parent != "" ||
		len(filter.Tags) > 0 ||
		filter.Contributor != ""
}

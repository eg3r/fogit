package printer

import (
	"fmt"
	"io"

	"github.com/eg3r/fogit/pkg/fogit"
)

// OutputFilesSummary prints a summary of file associations
func OutputFilesSummary(w io.Writer, features []*fogit.Feature) error {
	// Count total files and build file -> features map
	fileToFeatures := make(map[string][]string)
	totalFiles := 0

	for _, f := range features {
		for _, file := range f.Files {
			fileToFeatures[file] = append(fileToFeatures[file], f.Name)
			totalFiles++
		}
	}

	if totalFiles == 0 {
		fmt.Fprintln(w, "No file associations found")
		return nil
	}

	fmt.Fprintf(w, "File Associations Summary\n")
	fmt.Fprintf(w, "=========================\n\n")
	fmt.Fprintf(w, "Total file references: %d\n", totalFiles)
	fmt.Fprintf(w, "Unique files: %d\n", len(fileToFeatures))
	fmt.Fprintf(w, "Features with files: %d/%d\n\n", countFeaturesWithFiles(features), len(features))

	// Show files referenced by multiple features
	multipleRefs := []string{}
	for file, feats := range fileToFeatures {
		if len(feats) > 1 {
			multipleRefs = append(multipleRefs, file)
		}
	}

	if len(multipleRefs) > 0 {
		fmt.Fprintf(w, "Files referenced by multiple features:\n")
		for _, file := range multipleRefs {
			feats := fileToFeatures[file]
			fmt.Fprintf(w, "  %s (%d features)\n", file, len(feats))
			for _, feat := range feats {
				fmt.Fprintf(w, "    - %s\n", feat)
			}
		}
	}

	return nil
}

// OutputFilesForFeature prints files associated with a feature
func OutputFilesForFeature(w io.Writer, feature *fogit.Feature) error {
	if len(feature.Files) == 0 {
		fmt.Fprintf(w, "No files associated with feature '%s'\n", feature.Name)
		return nil
	}

	fmt.Fprintf(w, "Files in feature '%s':\n\n", feature.Name)
	for _, file := range feature.Files {
		fmt.Fprintf(w, "  %s\n", file)
	}
	fmt.Fprintf(w, "\nTotal: %d file(s)\n", len(feature.Files))

	return nil
}

// OutputFeaturesForFile prints features associated with a file
func OutputFeaturesForFile(w io.Writer, filePath string, features []*fogit.Feature) error {
	if len(features) == 0 {
		fmt.Fprintf(w, "No features found for file: %s\n", filePath)
		return nil
	}

	fmt.Fprintf(w, "Features associated with '%s':\n\n", filePath)
	for _, f := range features {
		fmt.Fprintf(w, "  %s\n", f.Name)
		fmt.Fprintf(w, "    ID: %s\n", f.ID)
		fmt.Fprintf(w, "    State: %s\n", f.DeriveState())
		if priority := f.GetPriority(); priority != "" {
			fmt.Fprintf(w, "    Priority: %s\n", priority)
		}
		if fType := f.GetType(); fType != "" {
			fmt.Fprintf(w, "    Type: %s\n", fType)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "Total: %d feature(s)\n", len(features))

	return nil
}

func countFeaturesWithFiles(features []*fogit.Feature) int {
	count := 0
	for _, f := range features {
		if len(f.Files) > 0 {
			count++
		}
	}
	return count
}

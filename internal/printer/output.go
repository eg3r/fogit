package printer

import (
	"encoding/json"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// Format represents an output format
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
	FormatTree Format = "tree"
)

// IsValidOutputFormat checks if the format is a valid output format
func IsValidOutputFormat(format string) bool {
	switch Format(format) {
	case FormatText, FormatJSON, FormatYAML, FormatTree:
		return true
	default:
		return false
	}
}

// OutputJSON writes any data as JSON to the writer
func OutputAsJSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// OutputYAML writes any data as YAML to the writer
func OutputAsYAML(w io.Writer, data interface{}) error {
	encoder := yaml.NewEncoder(w)
	defer encoder.Close()
	return encoder.Encode(data)
}

// OutputFormatted outputs data in the specified format
// For text format, it calls the provided textFn function
func OutputFormatted(w io.Writer, format string, data interface{}, textFn func(io.Writer) error) error {
	switch Format(format) {
	case FormatJSON:
		return OutputAsJSON(w, data)
	case FormatYAML:
		return OutputAsYAML(w, data)
	case FormatText, FormatTree:
		if textFn != nil {
			return textFn(w)
		}
		return fmt.Errorf("text output not implemented for this data type")
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

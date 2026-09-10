package output

import (
	"encoding/json"
	"fmt"
)

// Formatter interface for different output formats.
type Formatter interface {
	Format(data interface{}) (string, error)
}

// NewFormatter returns a formatter based on the format string.
func NewFormatter(format string) Formatter {
	switch format {
	case "json":
		return &JSONFormatter{}
	case "table":
		return &TableFormatter{}
	default:
		return &TableFormatter{}
	}
}

// TableFormatter renders output as a formatted table.
type TableFormatter struct{}

// Format implements Formatter for TableFormatter.
func (tf *TableFormatter) Format(data interface{}) (string, error) {
	return FormatTable(data)
}

// JSONFormatter renders output as JSON.
type JSONFormatter struct{}

// Format implements Formatter for JSONFormatter.
func (jf *JSONFormatter) Format(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(bytes), nil
}

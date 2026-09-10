package output

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

const (
	defaultTerminalWidth = 120
	minColumnWidth       = 8
	defaultColumnWidth   = 24
)

// FormatTable renders data as a formatted ASCII table.
func FormatTable(data interface{}) (string, error) {
	if data == nil {
		return "", nil
	}

	v := reflect.ValueOf(data)

	// Handle slices
	if v.Kind() == reflect.Slice {
		if v.Len() == 0 {
			return "(no results)", nil
		}
		return formatTableFromSlice(v)
	}

	// Handle maps (mainly for request counts)
	if v.Kind() == reflect.Map {
		return formatTableFromMap(v)
	}

	// Single object: treat as a single-item table
	return formatSingleRow(v)
}

// formatTableFromSlice converts a slice of structs to a table.
func formatTableFromSlice(v reflect.Value) (string, error) {
	t := table.NewWriter()

	// Get headers from first element
	firstElem := v.Index(0)
	headers := getTableHeaders(firstElem)
	configureTableForTerminal(t, headers)
	t.AppendHeader(stringsToTableRow(headers))

	// Add rows
	for i := 0; i < v.Len(); i++ {
		row := getTableRow(v.Index(i))
		t.AppendRow(toTableRow(row))
	}

	return t.Render(), nil
}

// formatTableFromMap converts a map to a table (for RequestCounts).
func formatTableFromMap(v reflect.Value) (string, error) {
	t := table.NewWriter()
	headers := []string{"IPv4", "MAC", "Requests"}
	configureTableForTerminal(t, headers)
	t.AppendHeader(stringsToTableRow(headers))

	for _, key := range v.MapKeys() {
		val := v.MapIndex(key).Elem()
		ipv4 := key.String()

		// Try to extract request count from the value
		count := val.Len() // For lists this gets the length
		mac := ""

		// Try to extract MAC from first element if it's a list
		if val.Kind() == reflect.Slice && val.Len() > 0 {
			first := val.Index(0)
			if macField := first.FieldByName("Mac"); macField.IsValid() {
				mac = macField.String()
			}
		}

		t.AppendRow(table.Row{ipv4, mac, count})
	}

	return t.Render(), nil
}

// formatSingleRow converts a single value to a table.
func formatSingleRow(v reflect.Value) (string, error) {
	t := table.NewWriter()

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		headers := getTableHeaders(v)
		row := getTableRow(v)

		configureTableForTerminal(t, headers)
		t.AppendHeader(stringsToTableRow(headers))
		t.AppendRow(toTableRow(row))
	}

	return t.Render(), nil
}

// getTableHeaders extracts field names as headers.
func getTableHeaders(v reflect.Value) []string {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var headers []string
	if v.Kind() != reflect.Struct {
		return headers
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		// Use JSON tag if available
		tag := field.Tag.Get("json")
		if tag != "" && tag != "-" {
			// Remove flags like omitempty
			parts := strings.Split(tag, ",")
			headers = append(headers, parts[0])
		} else {
			// Convert field name to title case
			headers = append(headers, toTitleCase(field.Name))
		}
	}

	return headers
}

// getTableRow extracts field values as a row.
func getTableRow(v reflect.Value) []interface{} {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var row []interface{}
	if v.Kind() != reflect.Struct {
		return row
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldValue := v.Field(i)
		row = append(row, formatFieldValue(fieldValue))
	}

	return row
}

// formatFieldValue formats a field value for table display.
func formatFieldValue(v reflect.Value) string {
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Slice:
		// Join slice elements with comma
		var elements []string
		for i := 0; i < v.Len(); i++ {
			elements = append(elements, formatFieldValue(v.Index(i)))
		}
		return strings.Join(elements, ",")
	case reflect.Ptr:
		if v.IsNil() {
			return ""
		}
		return formatFieldValue(v.Elem())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// toTableRow converts []interface{} to table.Row.
func toTableRow(row []interface{}) table.Row {
	return table.Row(row)
}

func stringsToTableRow(row []string) table.Row {
	converted := make([]interface{}, len(row))
	for i, value := range row {
		converted[i] = value
	}
	return toTableRow(converted)
}

func configureTableForTerminal(t table.Writer, headers []string) {
	width, ok := terminalWidthForOutput(os.Stdout)
	if !ok || len(headers) == 0 {
		return
	}

	t.SetAllowedRowLength(width)
	t.SetColumnConfigs(columnConfigsForHeaders(headers, width))
}

func terminalWidthForOutput(stdout *os.File) (int, bool) {
	if stdout == nil || !isTerminal(stdout) {
		return 0, false
	}

	if width, _, err := term.GetSize(int(stdout.Fd())); err == nil && width > 0 {
		return width, true
	}

	if columns := strings.TrimSpace(os.Getenv("COLUMNS")); columns != "" {
		if width, err := strconv.Atoi(columns); err == nil && width > 0 {
			return width, true
		}
	}

	return defaultTerminalWidth, true
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func columnConfigsForHeaders(headers []string, terminalWidth int) []table.ColumnConfig {
	widths := scaledColumnWidths(headers, terminalWidth)
	configs := make([]table.ColumnConfig, 0, len(headers))
	for i := range headers {
		configs = append(configs, table.ColumnConfig{
			Number:           i + 1,
			WidthMax:         widths[i],
			WidthMaxEnforcer: text.WrapSoft,
		})
	}
	return configs
}

func scaledColumnWidths(headers []string, terminalWidth int) []int {
	if len(headers) == 0 {
		return nil
	}

	prefWidths := make([]int, len(headers))
	sumPref := 0
	for i, header := range headers {
		pref := preferredColumnWidth(header)
		if pref < minColumnWidth {
			pref = minColumnWidth
		}
		prefWidths[i] = pref
		sumPref += pref
	}

	available := terminalWidth - (len(headers)*3 + 1)
	if available <= 0 || sumPref <= available {
		return prefWidths
	}

	scaled := make([]int, len(prefWidths))
	sumScaled := 0
	for i, pref := range prefWidths {
		value := pref * available / sumPref
		if value < minColumnWidth {
			value = minColumnWidth
		}
		scaled[i] = value
		sumScaled += value
	}

	for sumScaled > available {
		idx := widestColumnIndex(scaled)
		if scaled[idx] <= minColumnWidth {
			break
		}
		scaled[idx]--
		sumScaled--
	}

	return scaled
}

func widestColumnIndex(widths []int) int {
	idx := 0
	for i := 1; i < len(widths); i++ {
		if widths[i] > widths[idx] {
			idx = i
		}
	}
	return idx
}

func preferredColumnWidth(header string) int {
	switch strings.ToLower(strings.TrimSpace(header)) {
	case "mac":
		return 17
	case "ipv4":
		return 15
	case "hostname":
		return 24
	case "client_id":
		return 24
	case "vendor_class":
		return 20
	case "requested_options":
		return 28
	case "added", "renewed", "updated", "received", "first_seen", "last_seen", "requested":
		return 19
	case "tags":
		return 20
	case "lease_time":
		return 12
	default:
		return defaultColumnWidth
	}
}

// toTitleCase converts snake_case or camelCase to Title Case.
func toTitleCase(s string) string {
	// Simple conversion: split on underscore and capitalize
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + part[1:]
		}
	}
	return strings.Join(parts, " ")
}

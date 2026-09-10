package output

import (
	"strings"
	"testing"
)

func TestFormatTableNilAndEmptySlice(t *testing.T) {
	out, err := FormatTable(nil)
	if err != nil {
		t.Fatalf("expected no error for nil data, got: %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output for nil data, got: %q", out)
	}

	out, err = FormatTable([]struct{}{})
	if err != nil {
		t.Fatalf("expected no error for empty slice, got: %v", err)
	}
	if out != "(no results)" {
		t.Fatalf("expected '(no results)' for empty slice, got: %q", out)
	}
}

func TestFormatTableIncludesHeaders(t *testing.T) {
	type sample struct {
		MAC      string `json:"mac"`
		Hostname string `json:"hostname"`
	}

	out, err := FormatTable([]sample{{MAC: "aa:bb:cc:dd:ee:ff", Hostname: "device-1"}})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	lowerOut := strings.ToLower(out)
	if !strings.Contains(lowerOut, "mac") {
		t.Fatalf("expected rendered table to include mac header, got: %s", out)
	}
	if !strings.Contains(lowerOut, "hostname") {
		t.Fatalf("expected rendered table to include hostname header, got: %s", out)
	}
}

func TestPreferredColumnWidth(t *testing.T) {
	tests := []struct {
		header   string
		expected int
	}{
		{header: "mac", expected: 17},
		{header: "ipv4", expected: 15},
		{header: "requested_options", expected: 28},
		{header: "unknown_field", expected: defaultColumnWidth},
	}

	for _, test := range tests {
		if got := preferredColumnWidth(test.header); got != test.expected {
			t.Fatalf("preferredColumnWidth(%q) = %d, expected %d", test.header, got, test.expected)
		}
	}
}

func TestScaledColumnWidthsShrinkOnNarrowTerminal(t *testing.T) {
	headers := []string{"mac", "hostname", "requested_options", "vendor_class"}
	widths := scaledColumnWidths(headers, 52)

	if len(widths) != len(headers) {
		t.Fatalf("expected %d widths, got %d", len(headers), len(widths))
	}

	sum := 0
	for i, w := range widths {
		if w < minColumnWidth {
			t.Fatalf("column %d width too small: %d", i, w)
		}
		sum += w
	}

	available := 52 - (len(headers)*3 + 1)
	if sum > available {
		t.Fatalf("expected summed widths %d to fit available %d", sum, available)
	}
}

func TestScaledColumnWidthsPreferConfiguredSizesWhenWide(t *testing.T) {
	headers := []string{"mac", "ipv4", "hostname"}
	widths := scaledColumnWidths(headers, 200)

	expected := []int{17, 15, 24}
	for i := range expected {
		if widths[i] != expected[i] {
			t.Fatalf("column %d width = %d, expected %d", i, widths[i], expected[i])
		}
	}
}

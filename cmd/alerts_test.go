package cmd

import (
	"testing"

	"github.com/amigus/dnsmasq-web/internal/client"
)

func TestFreshAlerts(t *testing.T) {
	seen := map[uint]bool{}

	first := freshAlerts([]client.Alert{{ID: 1, MAC: "aa:bb:cc:dd:ee:01"}, {ID: 2, MAC: "aa:bb:cc:dd:ee:02"}}, seen)
	if len(first) != 2 {
		t.Fatalf("expected 2 fresh alerts, got %d", len(first))
	}

	repeat := freshAlerts([]client.Alert{{ID: 1, MAC: "aa:bb:cc:dd:ee:01"}}, seen)
	if len(repeat) != 0 {
		t.Fatalf("expected repeated alert to be suppressed, got %d", len(repeat))
	}

	second := freshAlerts([]client.Alert{{ID: 2}, {ID: 3}}, seen)
	if len(second) != 1 || second[0].ID != 3 {
		t.Fatalf("expected only alert 3, got %+v", second)
	}
}

func TestAlertsCommandStructure(t *testing.T) {
	subcommands := map[string]bool{}
	for _, sub := range alertsCmd.Commands() {
		subcommands[sub.Name()] = true
	}
	for _, want := range []string{"list", "get", "acknowledge", "watch"} {
		if !subcommands[want] {
			t.Fatalf("expected alerts subcommand %q, have %v", want, subcommands)
		}
	}
}

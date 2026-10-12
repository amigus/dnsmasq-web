package cmd

import (
	"strings"
	"testing"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/spf13/cobra"
)

func TestAnnotationPatchFromFlags(t *testing.T) {
	tests := []struct {
		name    string
		flags   []string
		wantErr string
		check   func(t *testing.T, patch *client.AnnotationPatch)
	}{
		{
			name:  "name only",
			flags: []string{"name=Living-room TV"},
			check: func(t *testing.T, patch *client.AnnotationPatch) {
				if patch.Name == nil || *patch.Name != "Living-room TV" {
					t.Fatalf("expected name to be set, got %+v", patch)
				}
				if patch.Owner != nil || patch.Notes != nil || patch.Tags != nil {
					t.Fatalf("expected other fields to be omitted, got %+v", patch)
				}
			},
		},
		{
			name:  "clear name",
			flags: []string{"clear-name=true"},
			check: func(t *testing.T, patch *client.AnnotationPatch) {
				if patch.Name == nil || *patch.Name != "" {
					t.Fatalf("expected empty name patch, got %+v", patch)
				}
			},
		},
		{
			name:    "contradictory name",
			flags:   []string{"name=x", "clear-name=true"},
			wantErr: "--name and --clear-name",
		},
		{
			name:  "repeatable tags",
			flags: []string{"tag=iot", "tag=media"},
			check: func(t *testing.T, patch *client.AnnotationPatch) {
				if patch.Tags == nil || len(*patch.Tags) != 2 || (*patch.Tags)[0] != "iot" || (*patch.Tags)[1] != "media" {
					t.Fatalf("expected tags [iot media], got %+v", patch.Tags)
				}
			},
		},
		{
			name:  "clear tags",
			flags: []string{"clear-tags=true"},
			check: func(t *testing.T, patch *client.AnnotationPatch) {
				if patch.Tags == nil || len(*patch.Tags) != 0 {
					t.Fatalf("expected empty tag slice, got %+v", patch.Tags)
				}
			},
		},
		{
			name:    "contradictory tags",
			flags:   []string{"tag=iot", "clear-tags=true"},
			wantErr: "--tag and --clear-tags",
		},
		{
			name:    "no fields",
			wantErr: "at least one annotation flag",
		},
		{
			name:  "all fields",
			flags: []string{"name=n", "owner=o", "notes=nt", "tag=t1"},
			check: func(t *testing.T, patch *client.AnnotationPatch) {
				if patch.Name == nil || patch.Owner == nil || patch.Notes == nil || patch.Tags == nil {
					t.Fatalf("expected all fields set, got %+v", patch)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "set"}
			addAnnotationSetFlags(cmd)
			for _, flag := range tt.flags {
				key, value, _ := strings.Cut(flag, "=")
				if err := cmd.Flags().Set(key, value); err != nil {
					t.Fatalf("set flag %s: %v", flag, err)
				}
			}

			patch, err := annotationPatchFromFlags(cmd)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, patch)
			}
		})
	}
}

func TestDeviceTableRows(t *testing.T) {
	devices := []client.Device{{
		MAC:          "bc:32:b2:3b:13:d4",
		Observed:     client.DeviceObservation{Hostname: "phone", RequestCount: 3},
		ActiveLeases: []client.DeviceLease{{IPv4: "192.168.1.9"}},
		Annotation:   &client.DeviceAnnotation{Name: "Phone", Owner: "Adam", Tags: []string{"trusted", "mobile"}},
	}}

	rows := deviceTableRows(devices)
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row.Hostname != "phone" || row.Requests != 3 || row.ActiveIPv4s != "192.168.1.9" || row.Name != "Phone" || row.Owner != "Adam" || row.Tags != "trusted,mobile" {
		t.Fatalf("unexpected row %+v", row)
	}

	rows = deviceTableRows([]client.Device{{MAC: "aa:bb:cc:dd:ee:ff"}})
	if rows[0].ActiveIPv4s != "" || rows[0].Name != "" || rows[0].Tags != "" {
		t.Fatalf("expected empty annotation columns, got %+v", rows[0])
	}
}

func TestDevicesCommandStructure(t *testing.T) {
	subcommands := map[string]bool{}
	for _, sub := range devicesCmd.Commands() {
		subcommands[sub.Name()] = true
	}
	for _, want := range []string{"list", "get", "addresses", "annotation"} {
		if !subcommands[want] {
			t.Fatalf("expected devices subcommand %q, have %v", want, subcommands)
		}
	}

	annotation := map[string]bool{}
	for _, sub := range devicesAnnotationCmd.Commands() {
		annotation[sub.Name()] = true
	}
	for _, want := range []string{"set", "delete"} {
		if !annotation[want] {
			t.Fatalf("expected annotation subcommand %q, have %v", want, annotation)
		}
	}
}

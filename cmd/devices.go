package cmd

import (
	"fmt"
	"strings"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:        "devices <ipv4>",
	Short:      "Show MAC history for an IPv4 address",
	Deprecated: "use 'devices list' for the canonical device inventory or 'addresses get <ipv4>' for IPv4 history",
	Args:       cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		ipv4 := args[0]

		history, err := cl.GetDevices(ipv4)
		if err != nil {
			return fmt.Errorf("failed to fetch device history: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(history)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var devicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List observed devices with optional annotations",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		devices, err := cl.GetV1Devices()
		if err != nil {
			return fmt.Errorf("failed to fetch devices: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		var data any = devices
		if cfg.OutputFormat != "json" {
			data = deviceTableRows(devices)
		}
		out, err := formatter.Format(data)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var devicesGetCmd = &cobra.Command{
	Use:   "get <mac>",
	Short: "Show one observed device by MAC address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		device, err := cl.GetV1Device(args[0])
		if err != nil {
			return fmt.Errorf("failed to fetch device: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		var data any = device
		if cfg.OutputFormat != "json" {
			data = deviceTableRows([]client.Device{*device})
		}
		out, err := formatter.Format(data)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var devicesAddressesCmd = &cobra.Command{
	Use:   "addresses <mac>",
	Short: "Show historical IPv4 assignments for a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		addresses, err := cl.GetV1DeviceAddresses(args[0])
		if err != nil {
			return fmt.Errorf("failed to fetch device addresses: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(addresses)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var devicesAnnotationCmd = &cobra.Command{
	Use:   "annotation",
	Short: "Manage application-owned device annotations",
}

var devicesAnnotationSetCmd = &cobra.Command{
	Use:   "set <mac>",
	Short: "Create or update the annotation for a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		patch, err := annotationPatchFromFlags(cmd)
		if err != nil {
			return err
		}

		annotation, err := cl.UpdateV1DeviceAnnotation(args[0], *patch)
		if err != nil {
			return fmt.Errorf("failed to update annotation: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(annotation)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var devicesAnnotationDeleteCmd = &cobra.Command{
	Use:   "delete <mac>",
	Short: "Delete the annotation for a device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)

		if err := cl.DeleteV1DeviceAnnotation(args[0]); err != nil {
			return fmt.Errorf("failed to delete annotation: %w", err)
		}

		fmt.Println("annotation deleted")
		return nil
	},
}

// annotationPatchFromFlags builds an annotation patch from set and clear
// flags, rejecting contradictory use of the same field and empty patches.
func annotationPatchFromFlags(cmd *cobra.Command) (*client.AnnotationPatch, error) {
	patch := &client.AnnotationPatch{}

	nameSet := cmd.Flags().Changed("name")
	nameClear := cmd.Flags().Changed("clear-name")
	if nameSet && nameClear {
		return nil, fmt.Errorf("--name and --clear-name cannot be used together")
	}
	if nameClear {
		empty := ""
		patch.Name = &empty
	} else if nameSet {
		value, _ := cmd.Flags().GetString("name")
		patch.Name = &value
	}

	ownerSet := cmd.Flags().Changed("owner")
	ownerClear := cmd.Flags().Changed("clear-owner")
	if ownerSet && ownerClear {
		return nil, fmt.Errorf("--owner and --clear-owner cannot be used together")
	}
	if ownerClear {
		empty := ""
		patch.Owner = &empty
	} else if ownerSet {
		value, _ := cmd.Flags().GetString("owner")
		patch.Owner = &value
	}

	notesSet := cmd.Flags().Changed("notes")
	notesClear := cmd.Flags().Changed("clear-notes")
	if notesSet && notesClear {
		return nil, fmt.Errorf("--notes and --clear-notes cannot be used together")
	}
	if notesClear {
		empty := ""
		patch.Notes = &empty
	} else if notesSet {
		value, _ := cmd.Flags().GetString("notes")
		patch.Notes = &value
	}

	tagsSet := cmd.Flags().Changed("tag")
	tagsClear := cmd.Flags().Changed("clear-tags")
	if tagsSet && tagsClear {
		return nil, fmt.Errorf("--tag and --clear-tags cannot be used together")
	}
	if tagsClear {
		empty := []string{}
		patch.Tags = &empty
	} else if tagsSet {
		tags, _ := cmd.Flags().GetStringSlice("tag")
		patch.Tags = &tags
	}

	if patch.Name == nil && patch.Owner == nil && patch.Notes == nil && patch.Tags == nil {
		return nil, fmt.Errorf("at least one annotation flag is required")
	}
	return patch, nil
}

// deviceTableRow flattens a device for table output.
type deviceTableRow struct {
	MAC         string `json:"mac"`
	Hostname    string `json:"hostname"`
	ClientID    string `json:"client_id"`
	VendorClass string `json:"vendor_class"`
	FirstSeen   string `json:"first_seen"`
	LastSeen    string `json:"last_seen"`
	Requests    int    `json:"request_count"`
	ActiveIPv4s string `json:"active_ipv4s"`
	Name        string `json:"annotation_name"`
	Owner       string `json:"annotation_owner"`
	Tags        string `json:"annotation_tags"`
}

// deviceTableRows flattens devices into rows for table output.
func deviceTableRows(devices []client.Device) []deviceTableRow {
	rows := make([]deviceTableRow, 0, len(devices))
	for _, device := range devices {
		row := deviceTableRow{
			MAC:         device.MAC,
			Hostname:    device.Observed.Hostname,
			ClientID:    device.Observed.ClientID,
			VendorClass: device.Observed.VendorClass,
			FirstSeen:   device.Observed.FirstSeen,
			LastSeen:    device.Observed.LastSeen,
			Requests:    device.Observed.RequestCount,
		}
		ipv4s := make([]string, 0, len(device.ActiveLeases))
		for _, lease := range device.ActiveLeases {
			ipv4s = append(ipv4s, lease.IPv4)
		}
		row.ActiveIPv4s = strings.Join(ipv4s, ",")
		if device.Annotation != nil {
			row.Name = device.Annotation.Name
			row.Owner = device.Annotation.Owner
			row.Tags = strings.Join(device.Annotation.Tags, ",")
		}
		rows = append(rows, row)
	}
	return rows
}

// addAnnotationSetFlags registers the annotation set flags on cmd.
func addAnnotationSetFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "Annotation name")
	cmd.Flags().String("owner", "", "Annotation owner")
	cmd.Flags().String("notes", "", "Annotation notes")
	cmd.Flags().StringSlice("tag", nil, "Replace annotation tags (repeatable)")
	cmd.Flags().Bool("clear-name", false, "Clear the annotation name")
	cmd.Flags().Bool("clear-owner", false, "Clear the annotation owner")
	cmd.Flags().Bool("clear-notes", false, "Clear the annotation notes")
	cmd.Flags().Bool("clear-tags", false, "Clear all annotation tags")
}

func init() {
	addAnnotationSetFlags(devicesAnnotationSetCmd)
	devicesCmd.AddCommand(devicesListCmd)
	devicesCmd.AddCommand(devicesGetCmd)
	devicesCmd.AddCommand(devicesAddressesCmd)
	devicesAnnotationCmd.AddCommand(devicesAnnotationSetCmd)
	devicesAnnotationCmd.AddCommand(devicesAnnotationDeleteCmd)
	devicesCmd.AddCommand(devicesAnnotationCmd)
}

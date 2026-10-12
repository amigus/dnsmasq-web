package cmd

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var addressesCmd = &cobra.Command{
	Use:        "addresses <mac>",
	Short:      "Show address history for a MAC address",
	Deprecated: "use /api/v1/devices/:mac/addresses for MAC history; CLI migration pending",
	Args:      cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		mac := args[0]

		history, err := cl.GetAddresses(mac)
		if err != nil {
			return fmt.Errorf("failed to fetch address history: %w", err)
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

var addressesGetCmd = &cobra.Command{
	Use:   "get <ipv4>",
	Short: "Show assignment history for an IPv4 address",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		history, err := cl.GetV1Address(args[0])
		if err != nil {
			return fmt.Errorf("failed to fetch address history: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		var data any = history
		if cfg.OutputFormat != "json" {
			data = history.Assignments
		}
		out, err := formatter.Format(data)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

func init() {
	addressesCmd.AddCommand(addressesGetCmd)
}

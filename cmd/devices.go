package cmd

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:   "devices <ipv4>",
	Short: "Show MAC history for an IPv4 address",
	Args:  cobra.ExactArgs(1),
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

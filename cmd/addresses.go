package cmd

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var addressesCmd = &cobra.Command{
	Use:   "addresses <mac>",
	Short: "Show address history for a MAC address",
	Args:  cobra.ExactArgs(1),
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

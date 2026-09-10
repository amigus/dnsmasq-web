package cmd

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var leasesCmd = &cobra.Command{
	Use:   "leases",
	Short: "List all active DHCP leases",
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		leases, err := cl.GetLeases()
		if err != nil {
			return fmt.Errorf("failed to fetch leases: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(leases)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

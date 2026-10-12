package cmd

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var requestsCmd = &cobra.Command{
	Use:        "requests",
	Short:      "Show request history for a CIDR or range",
	Deprecated: "use 'dhcp requests' for the canonical v1 API",
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		cidr, _ := cmd.Flags().GetString("cidr")
		ipRange, _ := cmd.Flags().GetString("range")

		if cidr == "" && ipRange == "" {
			return fmt.Errorf("either --cidr or --range is required")
		}

		requests, err := cl.GetRequests(cidr, ipRange)
		if err != nil {
			return fmt.Errorf("failed to fetch requests: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(requests)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

func init() {
	requestsCmd.Flags().String("cidr", "", "CIDR notation (e.g., 192.168.1.0/24)")
	requestsCmd.Flags().String("range", "", "IP range (e.g., 192.168.1.1-10)")
}

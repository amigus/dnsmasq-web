package cmd

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var dhcpCmd = &cobra.Command{
	Use:   "dhcp",
	Short: "Query DHCP diagnostics",
}

var dhcpRequestsCmd = &cobra.Command{
	Use:   "requests",
	Short: "List DHCP request events for one selector",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		mac, _ := cmd.Flags().GetString("mac")
		ipv4, _ := cmd.Flags().GetString("ipv4")
		cidr, _ := cmd.Flags().GetString("cidr")
		ipRange, _ := cmd.Flags().GetString("range")
		since, _ := cmd.Flags().GetString("since")
		selectors := 0
		for _, selector := range []string{mac, ipv4, cidr, ipRange} {
			if selector != "" {
				selectors++
			}
		}
		if selectors != 1 {
			return fmt.Errorf("exactly one of --mac, --ipv4, --cidr, or --range is required")
		}

		requests, err := cl.GetV1Requests(mac, ipv4, cidr, ipRange, since)
		if err != nil {
			return fmt.Errorf("failed to fetch DHCP requests: %w", err)
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
	dhcpRequestsCmd.Flags().String("mac", "", "MAC address")
	dhcpRequestsCmd.Flags().String("ipv4", "", "IPv4 address")
	dhcpRequestsCmd.Flags().String("cidr", "", "CIDR notation (e.g., 192.168.1.0/24)")
	dhcpRequestsCmd.Flags().String("range", "", "IP range (e.g., 192.168.1.1-10)")
	dhcpRequestsCmd.Flags().String("since", "", "Filter events since date (YYYY-MM-DD)")
	dhcpCmd.AddCommand(dhcpRequestsCmd)
}

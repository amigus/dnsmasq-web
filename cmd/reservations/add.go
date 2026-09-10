package reservations

import (
	"fmt"
	"strings"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new DHCP reservation",
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		mac, _ := cmd.Flags().GetString("mac")
		ipv4, _ := cmd.Flags().GetString("ipv4")
		hostname, _ := cmd.Flags().GetString("hostname")
		leaseTime, _ := cmd.Flags().GetString("lease-time")
		tagsStr, _ := cmd.Flags().GetString("tags")

		if mac == "" || ipv4 == "" {
			return fmt.Errorf("--mac and --ipv4 are required")
		}

		// Parse tags
		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
		}

		res, err := cl.CreateReservation(mac, ipv4, hostname, tags, leaseTime)
		if err != nil {
			return fmt.Errorf("failed to create reservation: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(res)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

func init() {
	addCmd.Flags().String("mac", "", "MAC address (required)")
	addCmd.Flags().String("ipv4", "", "IPv4 address (required)")
	addCmd.Flags().String("hostname", "", "Hostname (optional)")
	addCmd.Flags().String("tags", "", "Comma-separated tags (optional)")
	addCmd.Flags().String("lease-time", "", "Lease time (optional)")

	addCmd.MarkFlagRequired("mac")
	addCmd.MarkFlagRequired("ipv4")
}

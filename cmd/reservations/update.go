package reservations

import (
	"fmt"
	"strings"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <mac>",
	Short: "Update an existing DHCP reservation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		mac := args[0]

		// Build update map from provided flags
		update := make(map[string]interface{})

		if ipv4, _ := cmd.Flags().GetString("ipv4"); ipv4 != "" {
			update["ipv4"] = ipv4
		}
		if hostname, _ := cmd.Flags().GetString("hostname"); hostname != "" {
			update["hostname"] = hostname
		}
		if tagsStr, _ := cmd.Flags().GetString("tags"); tagsStr != "" {
			tags := strings.Split(tagsStr, ",")
			update["tags"] = tags
		}
		if leaseTime, _ := cmd.Flags().GetString("lease-time"); leaseTime != "" {
			update["lease_time"] = leaseTime
		}

		if len(update) == 0 {
			return fmt.Errorf("at least one field to update must be provided")
		}

		res, err := cl.UpdateReservation(mac, update)
		if err != nil {
			return fmt.Errorf("failed to update reservation: %w", err)
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
	updateCmd.Flags().String("ipv4", "", "New IPv4 address")
	updateCmd.Flags().String("hostname", "", "New hostname")
	updateCmd.Flags().String("tags", "", "Comma-separated tags")
	updateCmd.Flags().String("lease-time", "", "New lease time")
}

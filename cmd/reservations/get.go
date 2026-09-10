package reservations

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <mac>",
	Short: "Get a specific DHCP reservation by MAC",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		mac := args[0]

		res, err := cl.GetReservation(mac)
		if err != nil {
			return fmt.Errorf("failed to fetch reservation: %w", err)
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

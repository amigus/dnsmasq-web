package reservations

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all DHCP reservations",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get the client from parent context
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		reservations, err := cl.GetReservations()
		if err != nil {
			return fmt.Errorf("failed to fetch reservations: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(reservations)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

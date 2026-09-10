package reservations

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <mac>",
	Short: "Delete a DHCP reservation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)

		mac := args[0]

		if err := cl.DeleteReservation(mac); err != nil {
			return fmt.Errorf("failed to delete reservation: %w", err)
		}

		fmt.Printf("Reservation %s deleted successfully\n", mac)
		return nil
	},
}

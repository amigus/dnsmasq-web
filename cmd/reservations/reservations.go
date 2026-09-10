package reservations

import "github.com/spf13/cobra"

var ReservationsCmd = &cobra.Command{
	Use:   "reservations",
	Short: "Manage DHCP reservations",
}

func init() {
	// Add subcommands
	ReservationsCmd.AddCommand(listCmd)
	ReservationsCmd.AddCommand(getCmd)
	ReservationsCmd.AddCommand(addCmd)
	ReservationsCmd.AddCommand(updateCmd)
	ReservationsCmd.AddCommand(deleteCmd)
}

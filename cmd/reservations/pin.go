package reservations

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var pinCmd = &cobra.Command{
	Use:   "pin <mac>",
	Short: "Create a reservation from a device's current lease",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		hostname, _ := cmd.Flags().GetString("hostname")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		leaseTime, _ := cmd.Flags().GetString("lease-time")
		force, _ := cmd.Flags().GetBool("force")

		res, err := cl.PinReservation(args[0], hostname, tags, leaseTime, force)
		if err != nil {
			return fmt.Errorf("failed to pin reservation: %w", err)
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
	pinCmd.Flags().StringSlice("tag", nil, "Reservation tag (repeatable)")
	pinCmd.Flags().String("hostname", "", "Hostname override")
	pinCmd.Flags().String("lease-time", "", "Lease time override")
	pinCmd.Flags().Bool("force", false, "Override a conflicting lease")
}

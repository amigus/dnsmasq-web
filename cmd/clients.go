package cmd

import (
	"fmt"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var clientsCmd = &cobra.Command{
	Use:   "clients",
	Short: "List DHCP clients with request history",
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		since, _ := cmd.Flags().GetString("since")

		clients, err := cl.GetClients(since)
		if err != nil {
			return fmt.Errorf("failed to fetch clients: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(clients)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

func init() {
	clientsCmd.Flags().String("since", "", "Filter clients by date (YYYY-MM-DD)")
}

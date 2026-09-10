package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server [server flags]",
	Short: "Run dnsmasq-web API server/daemon",
	Long: `Run the dnsmasq-web API server.

In the unified binary, this command delegates to the original server flags,
for example: dnsmasq-web server -d -l :867 -f leases.db`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("server mode is handled by the unified binary dispatcher; run: dnsmasq-web server -d ...")
	},
}

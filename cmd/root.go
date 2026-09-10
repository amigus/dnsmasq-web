package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/amigus/dnsmasq-web/cmd/reservations"
	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/spf13/cobra"
)

var (
	cfg *client.Config
	cl  *client.Client
)

var rootCmd = &cobra.Command{
	Use:   "dnsmasq-web",
	Short: "Unified CLI for dnsmasq-web",
	Long: `dnsmasq-web is a unified server and CLI for the dnsmasq-web API.

Use 'dnsmasq-web server ...' to run the API server.
Use the remaining subcommands to query DHCP leases, clients, devices, and
manage DHCP reservations.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "server" {
			return nil
		}

		// Initialize config and HTTP client
		cfg = client.NewConfig()

		// Override env/default config only when flags are explicitly set.
		applyFlagOverrides(cmd, cfg)

		// Create HTTP client
		var err error
		cl, err = client.NewClient(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize client: %w", err)
		}

		// Store in context for subcommands
		ctx := context.WithValue(cmd.Context(), "client", cl)
		ctx = context.WithValue(ctx, "config", cfg)
		cmd.SetContext(ctx)

		return nil
	},
}

func applyFlagOverrides(cmd *cobra.Command, cfg *client.Config) {
	if cmd.Flags().Changed("server") {
		cfg.Server, _ = cmd.Flags().GetString("server")
	}
	if cmd.Flags().Changed("ssh-user") {
		cfg.SSHUser, _ = cmd.Flags().GetString("ssh-user")
	}
	if cmd.Flags().Changed("ssh-server") {
		cfg.SSHServer, _ = cmd.Flags().GetString("ssh-server")
	}
	if cmd.Flags().Changed("token-socket") {
		cfg.TokenSocket, _ = cmd.Flags().GetString("token-socket")
	}
	if cmd.Flags().Changed("token") {
		cfg.Token, _ = cmd.Flags().GetString("token")
	}
	if cmd.Flags().Changed("output") {
		cfg.OutputFormat, _ = cmd.Flags().GetString("output")
	}
	if cmd.Flags().Changed("no-auth") {
		cfg.NoAuth, _ = cmd.Flags().GetBool("no-auth")
	}
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().String("server", "[::1]:8080", "Server address (hostname:port)")
	rootCmd.PersistentFlags().String("ssh-user", "root", "SSH user for remote server")
	rootCmd.PersistentFlags().String("ssh-server", "", "SSH server address (computed from server if not set)")
	rootCmd.PersistentFlags().String("token-socket", "/run/dnsmasq-web.sock", "Unix socket for local token retrieval")
	rootCmd.PersistentFlags().String("token", "", "Explicit token (skips socket/SSH fetch)")
	rootCmd.PersistentFlags().String("output", "table", "Output format: table or json")
	rootCmd.PersistentFlags().Bool("no-auth", false, "Skip token validation (server must have -T 0)")

	// Add subcommands
	rootCmd.AddCommand(leasesCmd)
	rootCmd.AddCommand(clientsCmd)
	rootCmd.AddCommand(addressesCmd)
	rootCmd.AddCommand(devicesCmd)
	rootCmd.AddCommand(requestsCmd)
	rootCmd.AddCommand(reservations.ReservationsCmd)
	rootCmd.AddCommand(serverCmd)
}

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/amigus/dnsmasq-web/internal/output"
	"github.com/spf13/cobra"
)

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage new-device alerts",
}

var alertsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List device alerts, newest first",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		unacknowledgedOnly, _ := cmd.Flags().GetBool("unacknowledged")
		alerts, err := cl.GetV1Alerts(unacknowledgedOnly)
		if err != nil {
			return fmt.Errorf("failed to fetch alerts: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(alerts)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var alertsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one device alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		alert, err := cl.GetV1Alert(args[0])
		if err != nil {
			return fmt.Errorf("failed to fetch alert: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(alert)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var alertsAcknowledgeCmd = &cobra.Command{
	Use:   "acknowledge <id>",
	Short: "Acknowledge a device alert",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)
		cfg := cmd.Context().Value("config").(*client.Config)

		alert, err := cl.AcknowledgeV1Alert(args[0])
		if err != nil {
			return fmt.Errorf("failed to acknowledge alert: %w", err)
		}

		formatter := output.NewFormatter(cfg.OutputFormat)
		out, err := formatter.Format(alert)
		if err != nil {
			return err
		}

		fmt.Println(out)
		return nil
	},
}

var alertsWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Poll for unacknowledged alerts and print new ones",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cl := cmd.Context().Value("client").(*client.Client)

		interval, _ := cmd.Flags().GetDuration("interval")
		if interval <= 0 {
			return fmt.Errorf("--interval must be a positive duration")
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()

		seen := make(map[uint]bool)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			alerts, err := cl.GetV1Alerts(true)
			if err != nil {
				fmt.Fprintln(os.Stderr, "poll failed:", err)
			} else {
				for _, alert := range freshAlerts(alerts, seen) {
					fmt.Printf("new alert %d: %s (%s) first seen %s\n", alert.ID, alert.MAC, alert.Hostname, alert.DetectedAt)
				}
			}

			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	},
}

// freshAlerts returns alerts not yet seen and records their IDs in seen.
func freshAlerts(alerts []client.Alert, seen map[uint]bool) []client.Alert {
	fresh := make([]client.Alert, 0, len(alerts))
	for _, alert := range alerts {
		if seen[alert.ID] {
			continue
		}
		seen[alert.ID] = true
		fresh = append(fresh, alert)
	}
	return fresh
}

func init() {
	alertsListCmd.Flags().Bool("unacknowledged", false, "Only show alerts without an acknowledgement")
	alertsWatchCmd.Flags().Duration("interval", 30*time.Second, "Poll interval, e.g. 10s or 1m")
	alertsCmd.AddCommand(alertsListCmd)
	alertsCmd.AddCommand(alertsGetCmd)
	alertsCmd.AddCommand(alertsAcknowledgeCmd)
	alertsCmd.AddCommand(alertsWatchCmd)
}

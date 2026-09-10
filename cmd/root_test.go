package cmd

import (
	"testing"

	"github.com/amigus/dnsmasq-web/internal/client"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	c := &cobra.Command{Use: "test"}
	c.Flags().String("server", "[::1]:8080", "")
	c.Flags().String("ssh-user", "root", "")
	c.Flags().String("ssh-server", "", "")
	c.Flags().String("token-socket", "/run/dnsmasq-web.sock", "")
	c.Flags().String("token", "", "")
	c.Flags().String("output", "table", "")
	c.Flags().Bool("no-auth", false, "")
	return c
}

func TestApplyFlagOverrides_DoesNotClobberEnvConfig(t *testing.T) {
	cfg := &client.Config{
		Server:       "192.168.1.2:867",
		SSHUser:      "admin",
		SSHServer:    "192.168.1.2",
		TokenSocket:  "/tmp/dnsmasq-web.sock",
		Token:        "abc",
		NoAuth:       true,
		OutputFormat: "json",
	}

	cmd := newTestCmd()
	applyFlagOverrides(cmd, cfg)

	if cfg.Server != "192.168.1.2:867" {
		t.Fatalf("expected server to remain from env config, got %q", cfg.Server)
	}
	if cfg.SSHUser != "admin" {
		t.Fatalf("expected ssh-user to remain from env config, got %q", cfg.SSHUser)
	}
	if cfg.OutputFormat != "json" {
		t.Fatalf("expected output to remain from env config, got %q", cfg.OutputFormat)
	}
	if cfg.NoAuth != true {
		t.Fatalf("expected no-auth to remain true from env config")
	}
}

func TestApplyFlagOverrides_UsesExplicitFlags(t *testing.T) {
	cfg := &client.Config{
		Server:       "192.168.1.2:867",
		SSHUser:      "admin",
		OutputFormat: "json",
		NoAuth:       false,
	}

	cmd := newTestCmd()
	if err := cmd.Flags().Set("server", "10.0.0.2:8080"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("ssh-user", "root"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("output", "table"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("no-auth", "true"); err != nil {
		t.Fatal(err)
	}

	applyFlagOverrides(cmd, cfg)

	if cfg.Server != "10.0.0.2:8080" {
		t.Fatalf("expected explicit --server to win, got %q", cfg.Server)
	}
	if cfg.SSHUser != "root" {
		t.Fatalf("expected explicit --ssh-user to win, got %q", cfg.SSHUser)
	}
	if cfg.OutputFormat != "table" {
		t.Fatalf("expected explicit --output to win, got %q", cfg.OutputFormat)
	}
	if cfg.NoAuth != true {
		t.Fatalf("expected explicit --no-auth=true to win")
	}
}

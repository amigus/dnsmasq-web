package client

import (
	"os"
	"strings"
)

// Config holds the configuration for communicating with the dnsmasq-web server.
type Config struct {
	Server       string // Server address (e.g., "[::1]:8080" or "192.168.1.1:8080")
	SSHServer    string // SSH server (extracted from Server if remote, or override)
	SSHUser      string // SSH user (default: root)
	TokenSocket  string // Unix domain socket path for local token retrieval
	Token        string // Explicit token (skip socket fetch if set)
	NoAuth       bool   // Skip token checking (server must have -T 0)
	OutputFormat string // Output format: "table" or "json"
}

// NewConfig creates a new Config with environment variable and default precedence.
// Flags can override env vars via the caller.
func NewConfig() *Config {
	return &Config{
		Server:       getEnv("DNSMASQ_WEB_SERVER", "[::1]:8080"),
		SSHServer:    getEnv("DNSMASQ_WEB_SSH_SERVER", ""),
		SSHUser:      getEnv("DNSMASQ_WEB_SSH_USER", "root"),
		TokenSocket:  getEnv("DNSMASQ_WEB_TOKEN_SOCKET", "/run/dnsmasq-web.sock"),
		Token:        getEnv("DNSMASQ_WEB_TOKEN", ""),
		NoAuth:       getEnvBool("DNSMASQ_WEB_NO_AUTH", false),
		OutputFormat: getEnv("DNSMASQ_WEB_OUTPUT", "table"),
	}
}

// ServerIsLocal returns true if the server address is localhost or 127.0.0.1.
func (c *Config) ServerIsLocal() bool {
	ip := extractIP(c.Server)
	return ip == "::1" || ip == "127.0.0.1" || ip == "localhost"
}

// ResolveSSHServer sets the SSH server from the Server address if not already set.
func (c *Config) ResolveSSHServer() {
	if c.SSHServer == "" && !c.ServerIsLocal() {
		c.SSHServer = extractIP(c.Server)
	}
}

// extractIP extracts the IP address from a server string like "[::1]:8080" or "192.168.1.1:8080".
func extractIP(server string) string {
	// Handle IPv6 addresses in brackets
	if strings.HasPrefix(server, "[") {
		if idx := strings.Index(server, "]"); idx > 0 {
			return server[1:idx]
		}
	}
	// Handle IPv4 or hostname:port
	if idx := strings.LastIndex(server, ":"); idx > 0 {
		return server[:idx]
	}
	return server
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		return strings.ToLower(val) == "true" || val == "1"
	}
	return defaultVal
}

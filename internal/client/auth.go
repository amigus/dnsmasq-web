package client

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// TokenProvider handles token acquisition from a local Unix socket or remote SSH server.
type TokenProvider struct {
	Config *Config
}

// NewTokenProvider creates a new TokenProvider.
func NewTokenProvider(cfg *Config) *TokenProvider {
	return &TokenProvider{Config: cfg}
}

// GetToken returns a valid token for the configured server.
// It uses an explicit token if set, otherwise retrieves from local socket or SSH.
// If token retrieval fails, it returns an error (but may not block help commands).
func (tp *TokenProvider) GetToken() (string, error) {
	if tp.Config.Token != "" {
		return tp.Config.Token, nil
	}

	if tp.Config.ServerIsLocal() {
		return tp.getLocalToken()
	}

	return tp.getRemoteToken()
}

// getLocalToken retrieves a token from the local Unix domain socket using curl.
func (tp *TokenProvider) getLocalToken() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"curl",
		"--unix-socket", tp.Config.TokenSocket,
		"-s", ".",
	)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get token from socket: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getRemoteToken retrieves a token from a remote server via SSH.
func (tp *TokenProvider) getRemoteToken() (string, error) {
	server := tp.Config.SSHServer
	if server == "" {
		return "", fmt.Errorf("remote server SSH address not set")
	}

	user := tp.Config.SSHUser
	if user != "" {
		server = user + "@" + server
	}

	// SSH into the remote server and run the curl command to get the token
	cmdStr := fmt.Sprintf(
		"curl --unix-socket %s -s .",
		tp.Config.TokenSocket,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh", "-nqt", server, cmdStr)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get token via SSH: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

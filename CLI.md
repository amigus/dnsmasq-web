# dnsmasq-web CLI

The `dnsmasq-web` binary contains both the API server and the API client.
Use `dnsmasq-web server ...` to run the daemon and use every other subcommand as a client command.

## Installation

The client is included in the same `dnsmasq-web` binary that is shipped in releases and built locally.

## Configuration

Configuration is handled via environment variables and command-line flags. Precedence is: **flags > env vars > defaults**.

### Environment Variables

- `DNSMASQ_WEB_SERVER` — Server address (default: `[::1]:8080`)
- `DNSMASQ_WEB_SSH_USER` — SSH user for remote servers (default: `root`)
- `DNSMASQ_WEB_SSH_SERVER` — SSH server address (computed from `DNSMASQ_WEB_SERVER` if remote)
- `DNSMASQ_WEB_TOKEN_SOCKET` — Unix socket path for token retrieval (default: `/run/dnsmasq-web.sock`)
- `DNSMASQ_WEB_TOKEN` — Explicit token (skips socket/SSH fetch)
- `DNSMASQ_WEB_OUTPUT` — Output format: `table` or `json` (default: `table`)
- `DNSMASQ_WEB_NO_AUTH` — Skip token validation (requires server with `-T 0`)

### Global Flags

```bash
--server STRING              Server address (hostname:port)
--ssh-user STRING            SSH user for remote server (default: root)
--ssh-server STRING          SSH server address (auto-detected if remote)
--token-socket STRING        Unix socket for local token retrieval
--token STRING               Explicit token
--output FORMAT              Output format: table or json (default: table)
--no-auth                    Skip token validation (server -T 0 required)
```

## Commands

### Leases

```bash
dnsmasq-web leases
dnsmasq-web leases --output json
```

### Clients

```bash
dnsmasq-web clients
dnsmasq-web clients --since 2024-09-01
dnsmasq-web clients --output json
```

### Addresses

```bash
dnsmasq-web addresses aa:bb:cc:dd:ee:ff
dnsmasq-web addresses aa:bb:cc:dd:ee:ff --output json
```

### Devices

```bash
dnsmasq-web devices 192.168.1.90
dnsmasq-web devices 192.168.1.90 --output json
```

### Requests

```bash
# By CIDR
dnsmasq-web requests --cidr 192.168.1.0/24
dnsmasq-web requests --cidr 192.168.1.0/28 --output json

# By range
dnsmasq-web requests --range 192.168.1.1-10
dnsmasq-web requests --range 192.168.1.1-15 --output json
```

### Reservations

```bash
dnsmasq-web reservations list
dnsmasq-web reservations get aa:bb:cc:dd:ee:ff
dnsmasq-web reservations add --mac aa:bb:cc:dd:ee:ff --ipv4 192.168.1.50
dnsmasq-web reservations update aa:bb:cc:dd:ee:ff --ipv4 192.168.1.51
dnsmasq-web reservations delete aa:bb:cc:dd:ee:ff
```

## Examples

### Local server

```bash
dnsmasq-web leases
dnsmasq-web reservations add --mac aa:bb:cc:dd:ee:ff --ipv4 192.168.1.50
```

### Remote server via SSH

```bash
export DNSMASQ_WEB_SERVER=192.168.1.1:8080
export DNSMASQ_WEB_SSH_USER=admin

dnsmasq-web leases
dnsmasq-web reservations list
```

### With explicit token

```bash
export DNSMASQ_WEB_TOKEN=<token-from-server>
dnsmasq-web leases
```

### JSON output for parsing

```bash
dnsmasq-web clients --output json | jq '.[] | {mac, hostname, requests}'
dnsmasq-web addresses aa:bb:cc:dd:ee:ff --output json | jq '.[] | select(.ipv4 == "192.168.1.90")'
```

## Shell Completion

```bash
dnsmasq-web completion bash | sudo tee /etc/bash_completion.d/dnsmasq-web
dnsmasq-web completion zsh | sudo tee /usr/share/zsh/site-functions/_dnsmasq-web
```

## Build Modes

- `bash scripts/build.sh debug` writes the host debug binary to `dist/debug/dnsmasq-web`.
- `bash scripts/build.sh release` writes the supported release binaries to `dist/release/`.
- `mage` and `mage release` wrap those same entrypoints.

## Notes

- **Token Acquisition**: Tokens are fetched from a Unix domain socket on the server. For remote servers accessed via SSH, the socket path must be the same on both local and remote systems.
- **Token Lifetime**: Tokens are held in memory on the server and are discarded when it restarts. Re-run the command without the --token flag to fetch a fresh one.
- **SSH Configuration**: For seamless remote access, configure SSH to connect without prompts.
- **Timeouts**: Local token retrieval times out after 2 seconds; SSH token retrieval after 5 seconds.
- **Error Messages**: The CLI provides clear error messages for invalid arguments, authentication failures, and server errors.

## Troubleshooting

### "failed to get token from socket"

- Ensure the server is running (`dnsmasq-web server -d`)
- Verify the socket path: `--token-socket /run/dnsmasq-web.sock`
- Check socket permissions

### "Unauthorized token"

- Token may have expired or exceeded usage limits.
- Use `--token` to provide an explicit token, or remove the flag to let the CLI fetch a fresh one.

### "Server error (HTTP 4xx)"

- Check the server logs.
- Verify the request format, such as MAC addresses and IPv4 ranges.
- Ensure the server has the required subnet configured with `-s`.

## Comparison with Legacy Shell Helpers

The unified CLI provides the same API surface as the legacy shell helpers but with:

- **Faster execution** — No shell overhead or subprocess spawning
- **Better error handling** — Structured error messages instead of redirect parsing
- **Shell completion** — Auto-completion for commands and flags
- **JSON output** — Native JSON support for scripting
- **Type safety** — Go's type system catches errors earlier


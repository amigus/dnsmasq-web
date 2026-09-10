# Project Guidelines

## Build And Test
- Prefer `go test ./...` for validation.
- Use `mage` for the default local debug build, which writes the unified binary to `dist/debug/dnsmasq-web`.
- Use `bash scripts/build.sh release` or `mage release` for the multi-arch release build. It uses Docker by default so local output matches CI artifact targets and naming.
- For local release iteration, prefer the reusable prepared image via `bash scripts/build.sh prepare-release-image` or `mage prepareReleaseImage`; local release runs also reuse Docker volumes for the Go module and build caches.
- This project uses `github.com/mattn/go-sqlite3`, so builds require CGO. The release script bootstraps its toolchains in a container; native debug builds need a working C compiler on the host.

## Architecture
- `main.go` owns CLI flags, startup validation, Gin setup, and daemon versus foreground execution.
- `daemon.go` handles daemonization, inherited listeners, pid files, and privilege dropping.
- `lease_database.go` is the HTTP query layer over SQLite via GORM for `/leases`, `/clients`, `/addresses/:mac`, `/devices/:ipv4`, and `/requests`.
- `host_dir.go` is the reservation write path for dnsmasq `dhcp-hostsdir` files under `/reservations`.
- `token.go` and `token_gin.go` implement in-memory token issuance and Gin middleware for daemon mode.
- Start with [README.md](../README.md), [main.go](../main.go), [lease_database.go](../lease_database.go), and [host_dir.go](../host_dir.go) before changing behavior.

## Go Conventions
- Match the existing standard-library-first style and keep changes small and explicit.
- Preserve current JSON shapes and route structure unless the task explicitly changes the API contract.
- Keep validation close to request handling. This codebase returns JSON error bodies directly from handlers instead of introducing extra abstraction layers.
- When touching queries, preserve the current MAC and IPv4 normalization behavior and keep subnet filtering explicit.

## DHCP And Reservation Rules
- Treat `host_dir.go` as dnsmasq integration code, not generic file CRUD.
- Reservation files must stay compatible with dnsmasq `dhcp-hostsdir`: `MAC,set:tag1,set:tag2,IPv4[,hostname[,lease_time]]`.
- MAC addresses are validated and normalized before use. Do not introduce alternate formats in filenames, JSON, or SQL filters.
- `/requests` only accepts ranges within the configured subnet. Preserve that scope check when changing request-history logic.
- Token state is in memory only and is lost on restart; do not assume persistence unless you add it intentionally.

## API Design
- Keep handlers aligned with the existing REST surface documented in [README.md](../README.md).
- Prefer additive API changes over breaking renames. If you must change request or response fields, update the README examples in the same change.
- Follow the current response pattern: structured JSON on both success and failure, with meaningful HTTP status codes.
- When changing auth behavior, account for both HTTP requests and the Unix-socket token publisher used by the unified CLI.

## Practical Checks
- For handler changes, run `go test ./...`.
- For build or packaging changes, run the relevant `mage` target or `bash scripts/build.sh` mode you modified.

# Unified Binary Notes

The project now treats `dnsmasq-web` as the only supported executable surface.

## Runtime Contract

- `dnsmasq-web server ...` runs the API server and daemon mode.
- `dnsmasq-web <subcommand>` runs the Cobra-based client.
- The separate `dnsmasq-web-cli` wrapper is no longer part of the supported build or release process.
- The shell helpers under `cli/` are retained only as legacy reference material and are no longer built or shipped.

## Build Contract

- `go build -o dnsmasq-web .` builds the local debug binary in the repository root.
- `mage` builds the shell-helper environment and the host binaries (`dnsmasq-web-amd64`, `dnsmasq-web-arm64`).
- Release artifacts are built in CI by `.github/workflows/release.yml` via `go-cross/cgo-actions@v1`.

## Verification

- `go test ./...`
- `go build -o dnsmasq-web .`
- `./dnsmasq-web --help`
- `./dnsmasq-web server -V`

## Follow-On Cleanup

- If the legacy shell helper scripts are removed entirely later, update `README.md`, `CLI.md`, and `.github/copilot-instructions.md` in the same change.
- If the server command is refactored out of `main.go`, preserve the current `dnsmasq-web server ...` invocation so local docs and tooling stay stable.


# Unified Binary Notes

The project now treats `dnsmasq-web` as the only supported executable surface.

## Runtime Contract

- `dnsmasq-web server ...` runs the API server and daemon mode.
- `dnsmasq-web <subcommand>` runs the Cobra-based client.
- The separate `dnsmasq-web-cli` wrapper is no longer part of the supported build or release process.
- The shell helpers under `cli/` are retained only as legacy reference material and are no longer built or shipped.

## Build Contract

- `bash scripts/build.sh debug` builds the local debug binary at `dist/debug/dnsmasq-web`.
- `bash scripts/build.sh release` builds the supported release artifacts at `dist/release/`.
- `mage` wraps the debug build.
- `mage release` wraps the release build.
- The release build is containerized by default so local artifacts match CI targets and naming.

## Supported Release Targets

- `linux-amd64`
- `linux-arm64`
- `linux-amd64-musl`
- `linux-arm64-musl`

## Verification

- `go test ./...`
- `bash scripts/build.sh debug`
- `bash scripts/build.sh release`
- `./dist/debug/dnsmasq-web --help`
- `./dist/debug/dnsmasq-web server -V`

## Follow-On Cleanup

- If the legacy shell helper scripts are removed entirely later, update `README.md`, `CLI.md`, and `.github/copilot-instructions.md` in the same change.
- If the server command is refactored out of `main.go`, preserve the current `dnsmasq-web server ...` invocation so local docs and tooling stay stable.


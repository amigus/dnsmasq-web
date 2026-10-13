# Build and Release

## Authoritative Sources

- [Go install / modules docs](https://go.dev/doc/modules/)
- [`go build` flags](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)
- [`go test` flags](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [Magefile docs](https://magefile.org/) (the build orchestrator)

## Project Build Layout

| Path | Purpose |
|---|---|
| `Magefile.go` | `//go:build mage` targets for host builds and shell-helper env |
| `bin/` | Host binaries produced by `mage Binaries` |
| `cli/` | Legacy POSIX shell helpers (reference material, not built) |
| `.config/dlv/` | Delve config for the VS Code debugger |
| `.vscode/{launch,tasks}.json` | Debug + build tasks |

## Build Modes

- `go build -o dnsmasq-web .` — host debug binary in the repository root. Fast inner loop.
- `mage` (default `All`) — shell-helper environment plus host binaries.
- `mage Binaries` — `dnsmasq-web-amd64` and `dnsmasq-web-arm64` via cross-`gcc`.
- Release artifacts are built in CI by `.github/workflows/release.yml` via `go-cross/cgo-actions@v1`; there is no supported local multi-arch release path.

## CGO Is Required

`github.com/mattn/go-sqlite3` is a CGO package. Implications:

- **Host builds need a working C compiler** (`gcc`, or whatever `CC` points to).
- **Cross-compilation needs the matching C cross-toolchain** (e.g. `aarch64-suse-linux-gcc` for arm64), as wired in `Magefile.go`.
- Don't override `CGO_ENABLED=0` for these builds.

## Supported Release Targets

```
linux-amd64
linux-amd64-musl
linux-arm64
linux-arm64-musl
linux-arm-7
```

The release workflow matrix (see `.github/workflows/release.yml`) defines which targets CI builds; update that matrix if a target changes.

## Mage Targets

| Target | Does |
|---|---|
| `mage` (default) / `mage All` | shell-helper env + host binaries |
| `mage Binaries` | host binaries only (`dnsmasq-web-amd64`, `dnsmasq-web-arm64`) |
| `mage ClientEnv` / `mage FullClientEnv` | build the legacy `cli/` shell helper environment |
| `mage CleanBinaries` | remove built binaries |
| `mage CleanClientEnv` | remove the shell-helper environment |
| `mage Clean` | both cleans |

The `mage` binary reads the `//go:build mage` constraint on `Magefile.go`.

## What CI Runs

From `.github/workflows/test.yml`:

```bash
go test -v ./...
```

Locally, prove a change is shippable with:

```bash
go test -v ./...
go build -o dnsmasq-web .
./dnsmasq-web --help
./dnsmasq-web server -V
```

## Versioning

- `main.Version` is a package-level string in `main.go` set via `-ldflags '-X 'main.Version=...'` during the build.
- The version string is `"<git describe> built <date> by <user>@<host>"`, produced by `buildInfo` in `Magefile.go` and the release workflow's "Declare version" step. The smoke test only checks `-V` is non-empty.

## Things to Verify After Build Changes

1. `go test -v ./...` succeeds.
2. `go build -o dnsmasq-web .` succeeds (needs a host C compiler).
3. `./dnsmasq-web --help` prints help.
4. `./dnsmasq-web server -V` prints the version.

## Things to Verify After `Magefile.go` Changes

1. `mage -l` lists the target you expect.
2. `mage` (default target) builds.

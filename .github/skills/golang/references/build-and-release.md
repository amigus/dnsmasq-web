# Build and Release

## Authoritative Sources

- [Go install / modules docs](https://go.dev/doc/modules/)
- [`go build` flags](https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies)
- [`go test` flags](https://pkg.go.dev/cmd/go#hdr-Test_packages)
- [Magefile docs](https://magefile.org/) (the build orchestrator)
- Docker Engine reference (for the release path)

## Project Build Layout

| Path | Purpose |
|---|---|
| `Magefile.go` | `//go:build mage` targets wrapping `scripts/build.sh` |
| `scripts/build.sh` | The single source of truth for build modes |
| `scripts/release.Dockerfile` | The reusable release image |
| `dist/debug/` | Local debug binary + runtime artifacts (pid, sock, reservations.d) |
| `dist/release/` | Multi-arch release artifacts (built inside Docker by default) |
| `.toolchains/` | Downloaded musl cross-compilers (host-only) |
| `.config/dlv/` | Delve config for the VS Code debugger |
| `.vscode/{launch,tasks}.json` | Debug + smoke tasks |

## Build Modes

`scripts/build.sh <mode>`:

| Mode | What it does | When to use it |
|---|---|---|
| `debug` | Host-arch CGO build → `dist/debug/dnsmasq-web` | Default. Fast inner loop. |
| `release` | All supported targets via Docker (default) or host | Tagging a release or reproducing CI artifacts |
| `prepare-release-image` | Builds the reusable release image once | Speeds up repeated `release` invocations |
| `release-inner` | Internal entrypoint used by the Dockerized release build | Don't call from your shell |
| `clean` | Removes `dist/` | After toggling between modes |

## CGO Is Required

`github.com/mattn/go-sqlite3` is a CGO package. Implications:

- **Cross-compilation needs a C cross-toolchain.** The release script pulls them in the container (`scripts/release.Dockerfile`); host builds need them too if you flip `BUILD_USE_CONTAINER=0`.
- **`CGO_ENABLED=1` is the default** in `scripts/build.sh` debug and release flows. Don't override it.
- **musl targets use `-extldflags '-static -fpic'`.** See `release_ldflags` in `scripts/build.sh`. This produces a self-contained binary.
- **glibc targets use the matching cross-`gcc`.** `aarch64-linux-gnu-gcc` for arm64.

## Supported Release Targets

```
linux-amd64
linux-arm64
linux-amd64-musl
linux-arm64-musl
```

CI asserts exactly 4 artifacts (see `.github/workflows/release.yml` "Validate artifact set"). If you add a target, update both `RELEASE_TARGETS` in `scripts/build.sh` and the CI count check.

## Mage Targets (Mirrors `scripts/build.sh`)

| Target | Wraps |
|---|---|
| `mage` (default) / `mage Debug` | `scripts/build.sh debug` |
| `mage Release` | `scripts/build.sh release` |
| `mage PrepareReleaseImage` | `scripts/build.sh prepare-release-image` |
| `mage Binaries` | `scripts/build.sh release` (alias) |
| `mage All` | `scripts/build.sh debug` (alias) |
| `mage CleanBinaries` | `scripts/build.sh clean` |
| `mage CleanDaemonFiles` | removes pid, sock, reservations.d from `dist/debug/` |
| `mage Clean` | both cleans |
| `mage Start` | debug-build then run `server -v -f leases.db -h reservations.d -d -S ...sock -P ...pid` |
| `mage Kill` | SIGTERM the daemon from its pid file |
| `mage Stop` | `Kill` then `CleanDaemonFiles` |

The `mage` binary reads the `//go:build mage` constraint on `Magefile.go`. VS Code's `tasks.json` sets `go.buildTags: mage` so intellisense sees the right build.

## The Three Things CI Runs

From `.github/workflows/test.yml`:

```bash
go test -v ./...
bash scripts/build.sh debug
./dist/debug/dnsmasq-web --help
./dist/debug/dnsmasq-web reservations --help
./dist/debug/dnsmasq-web server -V
```

These four lines are the minimum that proves a change is shippable locally. Match them in your validation.

## Versioning

- `main.Version` is a package-level string in `main.go` set via `-ldflags '-X 'main.Version=...'` during the build.
- The debug ldflags only set the version (no `-s -w`); the release ldflags add `-s -w` for stripping.
- The version string is `"<git describe> # <UTC ISO 8601 timestamp>"` produced by `build_version` in `scripts/build.sh`. The format is part of the contract — the smoke test just checks `-V` is non-empty.

## Caches

| Cache | Where | When populated |
|---|---|---|
| Go module cache | Docker volume `dnsmasq-web-go-mod-cache` | First `release` build |
| Go build cache | Docker volume `dnsmasq-web-go-build-cache` | First `release` build |
| Prepared release image | `dnsmasq-web-release:golang-1.23.12-bookworm` (or `DNSMASQ_WEB_GO_IMAGE` override) | First `release` build, or `prepare-release-image` |
| Host musl toolchains | `.toolchains/<prefix>` | First `release` build with `BUILD_USE_CONTAINER=0` |

Set `DNSMASQ_WEB_PREPARED_RELEASE_IMAGE=0` to force the legacy disposable-container behavior. The repo defaults to the cached path for local iteration; CI uses the disposable path because the cache is per-run.

## Things to Verify After Build-Script Changes

1. `bash scripts/build.sh debug` succeeds.
2. `./dist/debug/dnsmasq-web --help` prints help.
3. `./dist/debug/dnsmasq-web server -V` prints the version.
4. `bash scripts/build.sh release` produces exactly 4 files in `dist/release/` matching the artifact naming pattern `dnsmasq-web_<label>_<target>`.
5. The CI workflow's "Validate artifact set" step would still pass.

## Things to Verify After `Magefile.go` Changes

1. `mage -l` lists the target you expect.
2. `mage` (default target) builds.
3. New or renamed targets still wrap the same `scripts/build.sh` mode.

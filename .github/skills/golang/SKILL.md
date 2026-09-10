---
name: golang
description: 'Authoritative Go development guide for this repo, blending Go community best practices (Effective Go, Go Code Review Comments, official tooling) with this codebase''s documented opinions. USE FOR: any change, review, or generation of Go code in this repo; Go style questions; CGO/SQLite/Gin/GORM/cobra questions; handler, package, or test design; build, release, or validation steps. DO NOT USE FOR: pure infrastructure work with no Go code, or generic shell scripting. Built around go 1.23, CGO, and the unified-binary layout described in [IMPLEMENTATION.md](../../../IMPLEMENTATION.md).'
---

# Go Development in This Repo

## Mission

Combine authoritative Go community guidance with this repo's stated conventions so generated code is correct by Go's standards *and* matches the author's style. Where the two diverge, prefer the author's style but call out the divergence in code review so the author can decide.

## Sources

- [Effective Go](https://go.dev/doc/effective_go) — the canonical Go style guide
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) — common style nits
- [Standard library layout](https://pkg.go.dev/std) — what idiomatic Go looks like
- [go.dev/doc/](https://go.dev/doc/) — official language and tooling docs
- This repo's [`.github/copilot-instructions.md`](../../copilot-instructions.md), [`IMPLEMENTATION.md`](../../../IMPLEMENTATION.md), [`README.md`](../../../README.md), [`CLI.md`](../../../CLI.md)

Detailed references live in `./references/`:

| Topic | Reference |
|---|---|
| Code style, naming, control flow, errors | [code-style.md](./references/code-style.md) |
| CGO, build, release, mage, Docker | [build-and-release.md](./references/build-and-release.md) |
| HTTP handlers, Gin, JSON, validation, auth | [http-handlers.md](./references/http-handlers.md) |
| SQLite + GORM query patterns | [database-and-queries.md](./references/database-and-queries.md) |
| Cobra CLI, env/flag precedence, output formatting | [cli-and-output.md](./references/cli-and-output.md) |
| Testing, httptest, testify, helper processes | [testing.md](./references/testing.md) |
| DHCP / dnsmasq integration rules | [domain-rules.md](./references/domain-rules.md) |
| Author-vs-community divergences to call out | [divergences.md](./references/divergences.md) |

## When to Load

Load this skill (or the more specific reference) when:

- Writing or reviewing any `.go` file
- Adding, removing, or refactoring a route, command, query, or package
- Touching `go.mod`, `Magefile.go`, `scripts/build.sh`, `scripts/release.Dockerfile`
- Touching `cmd/`, `internal/`, `main.go`, `daemon.go`, `host_dir.go`, `lease_database.go`, `token*.go`
- Answering a "how does X work here" question that ends in a Go code change

## Repo Facts That Drive Decisions

| Fact | Source | Implication |
|---|---|---|
| `go 1.23.0`, toolchain `go1.23.12` | [`go.mod`](../../../go.mod) | Use modern stdlib (`slices`, `maps`, `any`); no need for `golang.org/x/exp/slices` |
| Uses `github.com/mattn/go-sqlite3` | `go.mod` | CGO is **required**. The build script bootstraps cross-compilers in a container |
| Single unified binary `dnsmasq-web` | [`IMPLEMENTATION.md`](../../../IMPLEMENTATION.md) | `main.go` dispatches `server` vs. Cobra; never add a second top-level entrypoint |
| `main.go` owns CLI flags, startup, daemon vs. foreground | [copilot-instructions.md](../../copilot-instructions.md) | New server flags go in `main.go`; new client flags go in `cmd/*.go` |
| `lease_database.go` is the HTTP query layer | copilot-instructions.md | New query endpoints belong there, not scattered across packages |
| `host_dir.go` is dnsmasq integration code | copilot-instructions.md | Do not generalize it into "file CRUD" |
| Token state is in-memory only | copilot-instructions.md | Do not introduce persistence without an explicit ask |
| Validation stays close to handlers | copilot-instructions.md | Do not introduce a service/repository layer |
| Preserve current JSON shapes and routes | copilot-instructions.md | Additive changes only; update [`README.md`](../../../README.md) examples in the same change |

## Standard Workflow for Any Go Change

1. **Read first.** Before changing behavior, read the files named in [copilot-instructions.md](../../copilot-instructions.md) (Architecture section) plus the specific file(s) you'll touch.
2. **Match the existing style.** See [`code-style.md`](./references/code-style.md). When in doubt, copy the nearest-neighbor file.
3. **Make the change small and explicit.** No new abstraction layers, no premature helpers. If a helper is used once, inline it.
4. **Handle errors with `fmt.Errorf("...: %w", err)`** for wrapping. The current `main.go` uses `fmt.Fprintf(os.Stderr, "...: %v", err)` in startup paths because there is no caller to wrap to — that is acceptable. New code in `internal/` should prefer `%w`.
5. **Validate.**
   ```bash
   go test ./...
   ```
   If you touched build/packaging, also run `bash scripts/build.sh debug` (or `mage`) and the smoke test in `.vscode/tasks.json`.
6. **Update docs** in the same change when the surface changed: [`README.md`](../../../README.md), [`CLI.md`](../../../CLI.md), or [`IMPLEMENTATION.md`](../../../IMPLEMENTATION.md).
7. **Call out divergences** in the PR/response. See [`divergences.md`](./references/divergences.md).

## Quick Style Cheatsheet

The full version is in [`code-style.md`](./references/code-style.md). The non-negotiables:

- **Run `gofmt -s` mentally** on every edit. The repo has a few lingering unformatted files (`internal/auth/checker.go`, `internal/client/http.go`, `internal/reservations/file.go`, `token_gin.go`, `token_test.go`) — this is a known gap; new code you add must be gofmt-clean.
- **`go vet ./...` is a baseline** and currently passes; keep it that way.
- **No new dependencies without justification.** When adding one, the go.mod direct-require block is where it lands; check it's not already covered by an existing dep.
- **Naming**: package names are short, lowercase, no underscores; types are PascalCase; receivers are 1–2 letters; errors are `err`; the sentinel is `ErrXxx` only if exported.
- **Return early.** Validate inputs at the top of the function, return on failure, keep the success path linear.
- **No `interface{}` in new code** — use `any` (this is Go 1.23).
- **Comments**: doc comments on every exported identifier; match the terse, sentence-case style in the existing files. Do not narrate what the code obviously does.

## Quick Build Cheatsheet

The full version is in [`build-and-release.md`](./references/build-and-release.md).

| Want | Run |
|---|---|
| Local debug build | `mage` (or `bash scripts/build.sh debug`) → `dist/debug/dnsmasq-web` |
| Smoke-test the binary | `./dist/debug/dnsmasq-web --help` and `./dist/debug/dnsmasq-web server -V` |
| Multi-arch release | `bash scripts/build.sh release` (uses Docker; reuses prepared image + Go module/build cache volumes) |
| Prebuild the release image once | `bash scripts/build.sh prepare-release-image` or `mage prepareReleaseImage` |
| Run the daemon locally | `mage Start` (or follow the args in `.vscode/launch.json`) |
| Stop the daemon + clean up | `mage Stop` then `mage Clean` |

CGO is required. The release script handles cross toolchains inside the container. A native debug build needs `gcc` on the host (it builds `go-sqlite3`).

## Quick "Where Does This Go?" Cheatsheet

| Adding a... | Put it in | Notes |
|---|---|---|
| HTTP server flag | `main.go` `flag.*Var` block | Update `flag.Usage` text in the same place |
| HTTP route | `lease_database.go` (queries) or `host_dir.go` (reservations) or `token_gin.go` (auth) | Match the JSON-error response pattern already in the file |
| Cobra subcommand | `cmd/<name>.go` (top-level) or `cmd/<group>/<name>.go` (nested) | Wire into `rootCmd` via `AddCommand` in `cmd/root.go` `init()` |
| Shared HTTP client code | `internal/client/` | One file per concern (`config.go`, `auth.go`, `http.go`, `types.go`) |
| Output formatter | `internal/output/formatter.go` (interface + factory) and a `<format>.go` impl | Wire through `NewFormatter(cfg.OutputFormat)` in the command |
| Domain types shared by server and client | `internal/client/types.go` for client, `lease_database.go` for server, or a new `internal/<domain>/` if both grow | Match the existing JSON tag style |
| Test for a handler | `*_test.go` next to the file it tests, using `httptest` + `gin.New()` + `testify/assert` | See [`testing.md`](./references/testing.md) |

## Anti-Patterns to Reject (Author's Stated Rules)

- ❌ Adding a service layer / repository pattern. Validate in the handler.
- ❌ Persisting tokens across restarts. Token state is in-memory only.
- ❌ Renaming existing JSON fields or routes without an explicit ask. Additive changes only.
- ❌ Allowing `/requests` ranges outside the configured subnet. Keep the scope check.
- ❌ Using alternate MAC formats in filenames, JSON, or SQL. Normalize first, then store.
- ❌ Storing reservation files in a format dnsmasq `dhcp-hostsdir` cannot parse. Format is `MAC,set:tag1,set:tag2,IPv4[,hostname[,lease_time]]`.
- ❌ Splitting the unified binary into a separate `dnsmasq-web-cli`. The CLI subcommands live inside `dnsmasq-web`.

## Anti-Patterns to Reject (Community Best Practices)

- ❌ Ignoring `gofmt` / `go vet` warnings. New code is clean by default.
- ❌ `interface{}` in new Go 1.23 code. Use `any`.
- ❌ Naked `interface{}` parameters in exported APIs. Use generics or concrete types.
- ❌ Swallowing errors with `_`. Handle them or document why you don't.
- ❌ `init()` for anything that could be a normal function. (Existing `init()` for Cobra wiring in `cmd/root.go` is fine.)
- ❌ Goroutines without a clear ownership/lifecycle. The codebase has one `go func()` for the token publisher on a Unix socket — that's the pattern, not a license to add more.
- ❌ `panic` for control flow. Reserve for truly unrecoverable programmer errors.
- ❌ `time.Sleep` in tests where `time.After` + a channel works better (current `daemon_test.go` uses a 5s `time.After` after `signal.Notify` — acceptable for a signal test).

## PR / Response Hygiene

When you finish a Go change, end your response with a short list:

1. What changed (file + intent)
2. The validation you ran (`go test ./...`, build, smoke)
3. Any docs you updated ([`README.md`](../../../README.md), [`CLI.md`](../../../CLI.md), [`IMPLEMENTATION.md`](../../../IMPLEMENTATION.md))
4. Any divergences from Go community best practices you made on purpose, citing [`divergences.md`](./references/divergences.md)

Keep it short — the user can always ask for detail.

# Divergences: Author's Style vs. Go Community Best Practices

This file tracks the **intentional** divergences from authoritative Go community guidance. When you write code, prefer the author's style (this repo) unless the divergence would actively harm correctness or readability — in which case call it out.

## Format Drift (Existing)

- **What the community says**: `gofmt` (or `gofmt -s`) must be clean. CI should fail on drift. ([Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments))
- **What this repo does**: ships with five files that fail `gofmt -l`:
  - `internal/auth/checker.go`
  - `internal/client/http.go`
  - `internal/reservations/file.go`
  - `token_gin.go`
  - `token_test.go`
- **Decision**: **new code must be gofmt-clean**. Don't widen the drift. If a change touches one of these files, fix its formatting in the same change — but **do not** perform a sweeping `gofmt -w` PR without an explicit ask.

## No golangci-lint / Staticcheck / etc.

- **What the community says**: the standard Go toolchain is intentionally minimal. Most projects add `golangci-lint` with at least `govet`, `staticcheck`, `errcheck`, and `ineffassign` enabled.
- **What this repo does**: no linter config, no `golangci.yml`, CI only runs `go test -v ./...` and a build smoke test.
- **Decision**: don't add a linter config in the same change as a feature. If you want to add one, propose it as its own change so the diff is reviewable.

## Error Reporting in `main.go` Startup

- **What the community says**: `fmt.Errorf("...: %w", err)` and return; let the caller decide exit. Errors are values.
- **What this repo does**: `main.go` uses `fmt.Fprintf(os.Stderr, "...: %v", err)` then `os.Exit(1)` directly. There is no caller — `main()` is the top of the stack — so wrapping with `%w` would be wasted ceremony.
- **Decision**: this is correct for top-level `main` startup. **In `internal/` and reusable library code, prefer `%w`**. The author uses `%w` in `cmd/reservations/add.go`, `client/http.go`, etc. — the `%v`-only style is reserved for `main` startup paths.

## Goroutine Ownership in the Token Publisher

- **What the community says**: "the go statement has no nice way to express ownership over a goroutine" — every `go func()` should have a clear owner, lifecycle, and stop condition. ([Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments#goroutines))
- **What this repo does**: `main.go` has exactly one `go func() { ... TokenCheckerPublisher(gin.Default(), ttc, tokenEndpointPath).RunFd(4) ... }()` for the token publisher. It exits when the listener closes, because the listener's `Serve` returns.
- **Decision**: this is correct — it has a clear owner (the parent process), a clear lifecycle (until listener shutdown), and no shared state. **Do not add more goroutines** without the same rigor. New "fire and forget" goroutines are an anti-pattern here.

## Use of `slices.Index` Over `flag.Visit` in `daemon.go`

- **What the community says**: when you need to know which flags were set in `flag`, use `flag.Visit` (set flags only) or `flag.VisitAll` (all defined flags). Scanning `os.Args` is fragile and breaks with `=`-style flags and ordering.
- **What this repo does**: `daemon.go` `RunDaemon` uses `slices.Index(os.Args, "-d")` etc. to remove flags before re-execing as the daemon child. This works because the flags are short and well-known, and the `-d`/`-v` (bool) vs `-u`/`-g` (string) split is hard-coded.
- **Decision**: this is a deliberate trade-off — the flag set is small and stable, and the re-exec is a one-shot. **If a new flag is added in `main.go`, update the hard-coded index dance in `daemon.go` in the same change.** Don't refactor to `flag.Visit` here without a wider audit; the current shape is local and easy to grep for.

## Magic Strings for Header Names

- **What the community says**: export constants for shared strings; don't use string literals in multiple places.
- **What this repo does**: `main.go` defines `const tokenHeader = "X-Token"`, but other places use the literal:
  - `internal/client/http.go`: `c.restyClient.SetHeader("X-Token", ...)` and `c.retryCondition` checks `r.StatusCode() == http.StatusUnauthorized` separately
  - `token_gin_test.go`: `req.Header.Set("X-Token", ...)`
- **Decision**: the duplication is small and localized. Don't try to centralize the header name across `main.go` and `internal/client/` — the `internal/` package can't import `main`. A shared constant would need its own package. **Leave the literal as-is.**

## `interface{}` in `UpdateReservation`

- **What the community says (Go 1.18+)**: prefer `any` over `interface{}`. New code should not introduce `interface{}` literals.
- **What this repo does**: `internal/client/http.go` has `update map[string]interface{}` for the `PUT /reservations/:mac` body. The handler side decodes that into a struct (`reservationData`) so the API contract is `json.Unmarshal`-safe.
- **Decision**: this is a deliberate choice — the client accepts an arbitrary partial update. Switching to `any` is cosmetic. **Match the surrounding style if you're in that file.** In other files, use `any`.

## Reflection in `internal/output/table.go`

- **What the community says**: avoid reflection in hot paths; prefer code generation or explicit implementations.
- **What this repo does**: the table formatter uses `reflect` to extract struct tags and field values. The reason: the formatter has to handle any response type from any command without per-type code.
- **Decision**: this is the right tool for the job. The formatter is not on a hot path. **Don't rewrite it** to use generics — the response types aren't all known at one call site.

## PID File Parsing with `strconv.Atoi` in `Magefile.go`

- **What the community says**: prefer `strconv.Atoi` for integer parsing; for PIDs specifically, `os.FindProcess` is the standard way to reach a process without parsing the PID yourself.
- **What this repo does**: `Magefile.go` `Kill` reads the PID file, parses it with `strconv.Atoi`, and calls `syscall.Kill`. The reason: `mage` runs out-of-process and the daemon may have already died; `syscall.Kill(pid, 0)` is the cheap liveness check the script wants.
- **Decision**: correct for the use case. The `os.FindProcess` route would swallow the "no such process" error, which the script wants to surface.

## `if/else` Chains Returning Errors

- **What the community says**: return errors as values; minimize branching.
- **What this repo does**: `daemon.go` `id` uses nested `if/else` to handle name-vs-numeric lookup of users and groups. The branchiness is unavoidable because the API is two different lookups depending on whether the input is a number.
- **Decision**: correct. Don't refactor to a single `parseUint32` helper — the two cases are different enough that the helper would have to dispatch on the same `if` internally.

## No `Makefile` / Uses `mage`

- **What the community says**: there's no single community standard for build orchestration. `make`, `mage`, `task`, and shell scripts are all common.
- **What this repo does**: `mage` with `//go:build mage` on `Magefile.go`; release builds run in CI via `.github/workflows/release.yml`.
- **Decision**: don't replace `mage` with `make` or vice versa. Add build targets in `Magefile.go`.

## Single-File Server vs. Split Binary

- **What the community says**: it's common to have separate `cmd/server/main.go` and `cmd/cli/main.go` for a server+CLI pair.
- **What this repo does**: single unified `dnsmasq-web` binary with `server` dispatched to a different code path. This is documented in [`IMPLEMENTATION.md`](../../../../IMPLEMENTATION.md).
- **Decision**: this is the project's stated contract. Do not split the binary. The `cli/` legacy shell scripts are retained only as reference material.

## When You Add a Divergence

If your change introduces a new divergence (e.g. "I'm using a custom build tag for a feature flag"), add it to this file with the same structure: community guidance, repo behavior, decision rule. Future agents need the rationale, not just the rule.

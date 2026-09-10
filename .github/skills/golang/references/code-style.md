# Go Code Style for This Repo

## Authoritative Sources

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go FAQ](https://go.dev/doc/faq) — especially "Why does Go not have ...?" sections
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) — secondary, use as a tie-breaker, never in conflict with Effective Go

The rest of this document distills those into rules for this repo, then layers this codebase's specific opinions on top.

## Formatting (Non-Negotiable for New Code)

- **`gofmt -s`** is the law. Run it mentally on every edit.
- **The repo currently has unformatted files** (`internal/auth/checker.go`, `internal/client/http.go`, `internal/reservations/file.go`, `token_gin.go`, `token_test.go`). Fix these in the same change if you touch them, otherwise the debt grows. Don't make it worse.
- **Line length is gofmt's call, not yours.** Some long lines in `main.go` are intentional; do not wrap them by hand.
- **`go vet ./...` must stay clean.** If you add code that triggers a vet warning, fix the code, not vet.

## Naming

- **Package names**: short, lowercase, no underscores, no mixed case. Names describe *what the package provides*, not what it contains (`http`, `fmt`, `client`, `auth`, `reservations`).
- **Type names**: PascalCase. No `I` prefix for interfaces. Interface names are one-method descriptions when possible (`Checker` for `TokenChecker` here is an exception because the unexported `token` type and exported `Checker` share a root).
- **Functions/methods**: PascalCase exported, camelCase unexported. Getters drop the `Get` prefix when idiomatic (`token.Get()` is used here — keep that). Methods on a value receiver use the same name as the value (e.g. `*Checker.Check`).
- **Variables**: short in small scopes (`i`, `n`, `err`), longer in larger scopes. Local-loop variables shadowing is fine.
- **Constants**: PascalCase exported, camelCase unexported. This repo does **not** use `SCREAMING_SNAKE_CASE`.
- **Acronyms are consistent**: `MAC`, `IPv4`, `JSON`, `HTTP`, `CIDR`, `URL`, `UUID`. Two-letter acronyms (`IP`) are also uppercase.

## Comments and Docs

- **Every exported identifier gets a doc comment** starting with the identifier name. Existing files in this repo are sparse on doc comments — match the local file's density, but always add one for new exports.
- **Comments are sentences** with periods. Match the terse style in `internal/auth/checker.go` and `cmd/server.go` — they are sentence-case, not Title Case.
- **No narrating-the-obvious comments** (e.g. `// loop over users` above a `for _, u := range users`). Reserve comments for *why*, not *what*.
- **TODO/FIXME**: prefix with `TODO(name):` or `FIXME:` so they are greppable. The repo currently has none; add if you must.

## Control Flow

- **Return early.** The repo does this consistently — see `whereMac`, `whereIPv4`, `whereSince` in `lease_database.go`. The success path is flat.
- **No `else` after `return` or `continue`.** See `host_dir.go` handlers.
- **Errors are values, not control flow.** Don't use sentinel panics for normal cases.
- **Defer for cleanup only.** Don't defer in loops without thinking about it (the file-walk `defer file.Close()` in `host_dir.go` is per-call and fine).

## Errors

- **Wrap with `%w`, format with `%v`.** This is the modern Go rule.
  ```go
  // wrap (preserves error chain for errors.Is/As)
  return fmt.Errorf("failed to open %s: %w", path, err)

  // format (no wrapping needed)
  fmt.Fprintf(os.Stderr, "unable to open %s: %v\n", path, err)
  ```
- **In `main()` startup paths**, the codebase uses `Fprintf(os.Stderr, ...: %v, err)` then `os.Exit(1)`. That is correct for top-level error reporting with no caller. Keep that pattern for new top-level startup code.
- **In `internal/` and library-ish code**, prefer `%w` so callers can use `errors.Is` / `errors.As`.
- **Don't discard errors with `_`** unless the line above documents why (e.g. `defer file.Close()` where the surrounding `Write` is the real check). When you do, leave a one-line `// best-effort` comment.
- **Sentinel errors** use `ErrXxx` only if exported. The repo doesn't use any currently; introducing one needs a `errors.New` declaration.

## Types and Generics

- **Use `any` instead of `interface{}`.** Go 1.18+ idiom; this repo is on 1.23.
  - Exception: existing `map[string]interface{}` in `internal/client/http.go` `UpdateReservation` — keep the style of the file you're in or fix the file's formatting drift while you're there.
- **Use generics for type-parameterized containers** instead of `interface{}` reflection. The repo currently uses reflection in `internal/output/table.go` to read struct tags — that is correct for that use case and should not be rewritten.
- **Prefer concrete types over interfaces** at the point of declaration. Define the interface at the consumer, not the producer, when the producer only has one implementation (this is the "accept interfaces, return structs" rule, applied). The `auth.TokenChecker` interface in `internal/auth/checker.go` is a good example: it is small, single-purpose, and defined next to its sole implementation.

## Slices, Maps, and Memory

- **Use `slices.Contains`, `slices.Index`, `slices.SortFunc`** from the stdlib (`go 1.23`). The repo already uses `slices.Index` in `daemon.go` — keep using it.
- **Pre-allocate slices/maps when the size is known** with `make([]T, 0, n)` or `make(map[K]V, n)`. The repo does this in `lease_database.go` and `internal/reservations/file.go`.
- **Don't leak loop variables into goroutines** (`go func() { use(i) }()`) — Go 1.22+ fixed this; we're on 1.23 so it's a non-issue, but don't add `i := i` workarounds.

## Concurrency

- **One mutex per struct, named `mu`**, taken as the first field. See `auth.Checker.mu`. This is the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments#mutexes) convention.
- **Use `sync.Mutex` for short critical sections.** Switch to `sync.RWMutex` only with evidence of contention — the codebase has none today, and shouldn't pre-optimize.
- **Goroutines have a clear owner** and an explicit stop condition. The only goroutine in production code is the token publisher in `main.go`'s `go func() { ... TokenCheckerPublisher(...).RunFd(4) ... }()` — it exits when the listener closes.

## I/O

- **Buffered I/O with `bufio.Scanner`** for line-oriented reads. The repo does this in `host_dir.go` `readReservationFile`.
- **Always handle `scanner.Err()`** after the loop. The repo does.
- **Use `filepath.Join` not string concatenation** for paths. The repo does.

## Logging and Output

- **Top-level `main` uses `fmt.Fprintf(os.Stderr, ...)`** for status output, not a logging library. Keep that — the daemon is intentionally log-free and a downstream shell can capture stderr.
- **Don't introduce `log` or `slog` in this repo** without an explicit ask. They will conflict with the `-v` verbose-flag style.
- **`-v` controls verbosity.** New verbose output goes behind `if verbose { ... }` blocks, in the same style as `main.go` and `daemon.go`.

## JSON

- **Use struct tags with `json:"snake_case"`.** Every public type with JSON behavior in the repo follows this — see `internal/client/types.go`, `lease_database.go`, `host_dir.go`.
- **Use `omitempty`** on optional fields. Required fields (e.g. `binding:"required"`) do **not** get `omitempty`.
- **Raw `gin.H{"error": "..."}` for handler errors** is the explicit house style. Do not introduce a custom error type for handlers. The author has rejected this in [`copilot-instructions.md`](../../../copilot-instructions.md).

## What to *Not* Do (Project-Specific)

- ❌ No `panic` for control flow. Top-level `os.Exit(1)` is the error-exit mechanism.
- ❌ No global mutable state. (Module-level `var Version` for ldflags injection is the one exception.)
- ❌ No DI containers, no wire-up libraries. Constructor injection by hand.
- ❌ No `flag.Parse()` calls outside `main.go`. Subcommands use Cobra; sub-handlers take parameters from `*gin.Context`.
- ❌ No goroutine without a documented lifecycle.
- ❌ No HTTP client calls in `main.go`. All HTTP work goes through `internal/client/`.

## Divergences to Call Out

See [`divergences.md`](./divergences.md) for the running list. The notable one for this file: the project ships with gofmt drift; new code is still expected to be gofmt-clean, but do not perform a sweeping `gofmt -w` PR without an explicit ask.

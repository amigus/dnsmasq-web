# Cobra CLI, Env/Flag Precedence, Output Formatting

## Authoritative Sources

- [Cobra docs](https://github.com/spf13/cobra)
- [pflag docs](https://github.com/spf13/pflag) (Cobra's flag library)
- [go-resty/v2 docs](https://github.com/go-resty/resty) (the HTTP client)
- [jedib0t/go-pretty/v6/table](https://github.com/jedib0t/go-pretty) (the table formatter)

## Unified Binary Entry Point

`main.go` does this at the top:

```go
if len(os.Args) > 1 && os.Args[1] == "server" {
    os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
    runServerMain()
    return
}
cmdpkg.Execute()
```

The `server` subcommand is special-cased so the legacy server flags (`-d`, `-l`, `-f`, etc.) keep working. The `server` Cobra command in `cmd/server.go` is a stub:

```go
var serverCmd = &cobra.Command{
    Use: "server [server flags]",
    DisableFlagParsing: true,
    RunE: func(cmd *cobra.Command, args []string) error {
        return fmt.Errorf("server mode is handled by the unified binary dispatcher; run: dnsmasq-web server -d ...")
    },
}
```

**Do not change this dispatcher behavior.** Users depend on `dnsmasq-web server ...` reaching the server flags in `main.go`.

## Cobra Layout

```
cmd/
├── root.go              # rootCmd, PersistentPreRunE, Execute, init() with global flags
├── server.go            # serverCmd (stub; see above)
├── leases.go            # leasesCmd
├── clients.go           # clientsCmd
├── addresses.go         # addressesCmd
├── devices.go           # devicesCmd
├── requests.go          # requestsCmd
├── reservations/
│   ├── reservations.go  # ReservationsCmd (parent; AddCommand wires the rest)
│   ├── list.go
│   ├── get.go
│   ├── add.go
│   ├── update.go
│   └── delete.go
└── root_test.go         # applyFlagOverrides tests
```

Each command file declares its `*Cmd` and an `init()` if it has local flags. Subcommands are wired in the parent's `init()`.

## Precedence: Flags > Env > Defaults

`internal/client/config.go` reads env vars, then `cmd/root.go`'s `applyFlagOverrides` only overwrites fields whose flags were *explicitly* set:

```go
if cmd.Flags().Changed("server") {
    cfg.Server, _ = cmd.Flags().GetString("server")
}
```

This is the contract. **Do not change `applyFlagOverrides` to "always overwrite"** — that would clobber env-set config and break the precedence rule documented in [`CLI.md`](../../../../CLI.md) and the `TestApplyFlagOverrides_DoesNotClobberEnvConfig` test.

The CLI-side rule and the test are the spec. New flags follow the same pattern: env-driven default, flag override only if `cmd.Flags().Changed("name")`.

## Environment Variables

| Var | Default | Notes |
|---|---|---|
| `DNSMASQ_WEB_SERVER` | `[::1]:8080` | `[::1]:8675` syntax is accepted for IPv6 |
| `DNSMASQ_WEB_SSH_USER` | `root` | Used when the server is non-local |
| `DNSMASQ_WEB_SSH_SERVER` | (auto) | Derived from `Server` if non-local |
| `DNSMASQ_WEB_TOKEN_SOCKET` | `/run/dnsmasq-web.sock` | For local token retrieval |
| `DNSMASQ_WEB_TOKEN` | (empty) | Skips socket/SSH fetch |
| `DNSMASQ_WEB_OUTPUT` | `table` | `table` or `json` |
| `DNSMASQ_WEB_NO_AUTH` | `false` | Accepts `true`/`1`/case-insensitive |

`getEnvBool` accepts `true` and `1` as truthy (case-insensitive). Anything else is false. Don't extend it without updating tests.

## Command Pattern (the canonical shape)

Every command file follows the same recipe:

```go
var fooCmd = &cobra.Command{
    Use:   "foo [args]",
    Short: "Short description",
    Args:  cobra.ExactArgs(1),  // or MinimumN, MaximumN, RangeArgs, etc.
    RunE: func(cmd *cobra.Command, args []string) error {
        cl := cmd.Context().Value("client").(*client.Client)
        cfg := cmd.Context().Value("config").(*client.Config)

        out, err := cl.Foo(args[0])
        if err != nil {
            return fmt.Errorf("failed to fetch foo: %w", err)
        }

        formatter := output.NewFormatter(cfg.OutputFormat)
        formatted, err := formatter.Format(out)
        if err != nil {
            return err
        }

        fmt.Println(formatted)
        return nil
    },
}
```

Wired into `rootCmd` in `cmd/root.go init()`:

```go
rootCmd.AddCommand(fooCmd)
```

For nested subcommands (under `reservations`), wire them in the parent package's `init()`:

```go
// cmd/reservations/reservations.go
func init() {
    ReservationsCmd.AddCommand(listCmd)
    ReservationsCmd.AddCommand(getCmd)
    ReservationsCmd.AddCommand(addCmd)
    ReservationsCmd.AddCommand(updateCmd)
    ReservationsCmd.AddCommand(deleteCmd)
}
```

## Output Formatter

`internal/output/formatter.go` is a small factory:

```go
func NewFormatter(format string) Formatter {
    switch format {
    case "json":
        return &JSONFormatter{}
    case "table":
        return &TableFormatter{}
    default:
        return &TableFormatter{}
    }
}
```

- **`JSONFormatter`** uses `json.MarshalIndent` with two-space indent. Don't add color or escape characters; it's meant to be piped into `jq`.
- **`TableFormatter`** uses `jedib0t/go-pretty/v6/table` with reflection to read struct tags. Headers come from `json:"snake_case"` tags when present, otherwise from the field name in title case. Column widths are tuned for terminal output and shrink on narrow terminals.

### Reflection Caveat

The table formatter uses reflection. It works on:

- Slices of structs (the common case)
- Maps (mainly for `/requests` results, which return a `map[string][]...`)
- Single structs (rendered as a one-row table)

It does **not** recurse into pointer fields beyond a single deref. Don't add complex types to the response that the formatter can't handle — either flatten them or write a custom formatter.

## HTTP Client Construction

`internal/client/http.go` `NewClient` uses `go-resty/v2`. The interesting bits:

- **Token injection** via `SetHeader("X-Token", token)` only when `cfg.NoAuth` is false.
- **Retry on 401** via `AddRetryCondition` + a method that refreshes the token from `cfg.TokenSocket` (local) or SSH (remote) and re-issues the request.
- **No implicit retries on 5xx.** That's intentional — the client surfaces server errors to the user. Don't add a silent retry policy.

The token retry condition is the most fragile part of the client. If you change it, write a test that simulates a 401 followed by a successful retry.

## Local vs. Remote Token Fetch

`internal/client/auth.go`:

- **Local server** (`[::1]`, `127.0.0.1`, `localhost`): `curl --unix-socket <socket> -s .` returns the token directly.
- **Remote server**: `ssh -nqt <user>@<host> curl --unix-socket <socket> -s .` to fetch from the remote socket.

This is shell-out by design — no SSH library, no key management. The 2s/5s `context.WithTimeout` is the only safeguard. Don't add an SSH library unless the project also picks up an SSH key discovery story; that's a much bigger change.

## Adding a New Subcommand — Checklist

1. Pick a name that matches the existing surface: top-level for a new resource, or nested under an existing group.
2. Create `cmd/<name>.go` (or `cmd/<group>/<name>.go`) with the canonical shape above.
3. Add the file's `init()` if it has local flags.
4. Wire into the parent's `AddCommand` in the same `init()` block.
5. Add the new endpoint to [`README.md`](../../../../README.md) endpoints section and, if it's user-facing, to [`CLI.md`](../../../../CLI.md).
6. If it returns a new type, add it to `internal/client/types.go` with `json:"snake_case"` tags.
7. Add a test if the command has a non-trivial transformation.

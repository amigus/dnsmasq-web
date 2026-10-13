# HTTP Handlers, Gin, JSON, Auth

## Authoritative Sources

- [Gin docs](https://gin-gonic.com/docs/)
- [net/http package](https://pkg.go.dev/net/http)
- [REST API design — Microsoft / Google / OWASP general guidance]
- Project docs: [`README.md`](../../../../README.md) endpoint section, [`copilot-instructions.md`](../../../copilot-instructions.md) "API Design"

## Project Surface

Routes live in three files, each of which exposes a `func X(r *gin.Engine, ...) *gin.Engine` that mutates and returns the same engine:

| File | Routes |
|---|---|
| `lease_database.go` | `GET /leases`, `GET /addresses/:mac`, `GET /devices/:ipv4`, `GET /clients?since=`, `GET /requests?cidr=\|range=` |
| `host_dir.go` | `GET/POST /reservations`, `GET/PUT/DELETE /reservations/:mac` |
| `token_gin.go` | `TokenCheckerHeader` (middleware), `TokenCheckerPublisher` (GET on `/`) |

`main.go` wires them in this order: `DhcpHostDir(LeaseDatabase(r, gormDb, ...), hostDirPath)`. Token middleware is added conditionally only in daemon mode.

## Routing Conventions

- **Resource-oriented, REST-flavored** but pragmatic. The `PUT /reservations/:mac` style is correct; the `GET /addresses/:mac` style is also fine. Don't invent RPC-shaped `/doAction` routes.
- **Status codes:**
  - `200 OK` for successful reads and updates that return a body
  - `201 Created` for `POST /reservations` (matches the `gin.H{"message": "success"}` payload)
  - `400 Bad Request` for validation errors (invalid MAC, missing param, bad CIDR)
  - `404 Not Found` when a MAC isn't in the host directory
  - `409 Conflict` when a reservation already exists on create
  - `401 Unauthorized` for invalid tokens
  - `500 Internal Server Error` for filesystem/SQL failures
- **Additive changes only.** No breaking renames. New fields default to safe values. If you must break the contract, update the examples in [`README.md`](../../../../README.md) in the same change.

## JSON Patterns

**Response error shape (uniform across the project):**

```go
c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mac address"})
```

Use `gin.H` (a `map[string]any`) for ad-hoc error bodies. Do not introduce a custom error type — the author has rejected this in [`copilot-instructions.md`](../../../copilot-instructions.md).

**Success bodies are structs with `json:"snake_case"` tags**, e.g. `lease_database.go`'s `IPHistory`, `MacHistory`, and `Client` types. The unified binary's client reads these via `internal/client/types.go`, which mirrors them.

**`omitempty` is for genuinely optional fields.** Don't add it to fields that always have a value — that just hides empty strings downstream.

## Validation in the Handler

Validation lives next to the request. The pattern is `whereMac`, `whereIPv4`, `whereSince` in `lease_database.go`:

```go
func whereMac(a func(string) string, c *gin.Context, db *gorm.DB, name, field string, required bool) (*gorm.DB, bool) {
    mac := ipaddr.NewMACAddressString(a(name))
    if mac.IsEmpty() {
        if required {
            c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s address is required", name)})
        }
    } else {
        if mac.IsValid() {
            return db.Where(fmt.Sprintf("%s = ?", field), mac.GetAddress().ToColonDelimitedString()), true
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid mac address"})
        }
    }
    return nil, false
}
```

- **Takes a parser abstraction** (`a func(string) string`) so it works for both `c.Query` and `c.Param`.
- **Returns the modified `*gorm.DB` and a `bool ok`** so the caller can short-circuit on validation failure (`if query, ok = whereMac(...); !ok { return }`).
- **Emits the JSON error response itself** — the caller just `return`s.
- This pattern is intentional; new endpoints should follow it. If you find yourself writing the same error JSON by hand in a new handler, ask whether the pattern needs another helper.

## MAC and IPv4 Normalization

`github.com/seancfoley/ipaddress-go/ipaddr` is the only address parser. Rules:

- **MAC: validate first, then normalize to colon-delimited lowercase for JSON, and to the bare normalized form for filenames.** The two are different on purpose:
  - `mac.ToColonDelimitedString()` → `"00:1a:2b:3c:4d:5e"` (in JSON)
  - `mac.ToNormalizedString()` → `"001a2b3c4d5e"` (in `dhcp-hostsdir` filename)
- **IPv4: use `ipaddr.NewIPAddressString(...).ToAddress()`** then `ip.WithoutPrefixLen().ToNormalizedString()` for SQL binding.
- **Never introduce alternate MAC formats** in filenames, JSON, or SQL filters. This is a stated rule in [`copilot-instructions.md`](../../../copilot-instructions.md).

## Subnet Filtering

`/requests?cidr=` and `/requests?range=` must be inside the configured subnet (`-s` flag, default `192.168.0.0/16`). The check is in `ipListFromExpression` in `lease_database.go`. Keep the scope check — it's both a security boundary (don't expose data outside the managed network) and a dnsmasq operational rule.

## Auth: In-Memory Token Checker

Two files, one purpose:

- `internal/auth/checker.go` — `TokenChecker` interface, `Checker` struct (mutex-protected ring of UUIDs with use-count and timeout).
- `token.go` — re-exports the constructor `NewTokenChecker` so the root `main` package can use it without importing `internal/` directly. The alias is a small concession to keep the import graph clean.
- `token_gin.go` — `TokenCheckerHeader` middleware and `TokenCheckerPublisher` route.

Wiring (in `main.go` daemon branch):

```go
ttc := NewTokenChecker(maxTokens, maxTokenUses, tokenTimeout)
if maxTokens > 0 {
    r = TokenCheckerHeader(r, ttc, tokenHeader)         // "X-Token"
    go func() {
        if err := TokenCheckerPublisher(gin.Default(), ttc, tokenEndpointPath).RunFd(4); err != nil {
            // logged but not fatal — the parent may already be shutting down
        }
    }()
}
```

State lives in memory and dies on restart. Do not add a backing store. Do not move the token check into a separate process.

**The `X-Token` header name is a contract.** Changing it is a breaking change for the CLI client in `internal/client/`.

## Daemon vs. Foreground

`main.go` branches on whether `os.Getenv(listenerEnvVar)` is set:

- **Foreground** (`os.Args` didn't include `-d`): just `r.Run(listenOn)`.
- **Child of daemon** (env var set): the inherited file descriptors are used. `r.RunFd(3)` is the TCP listener; the goroutine on `RunFd(4)` is the token publisher. FDs 0/1/2 are stdin/out/err.

If you change how the listener is created or how FDs are inherited, the new shape must be replicated in the release workflow's build flags (`.github/workflows/release.yml`) and the `RunDaemon` setup in `daemon.go`.

## Adding a New Endpoint — Checklist

1. Decide if it belongs in `lease_database.go` (queries) or `host_dir.go` (reservation writes) or `token_gin.go` (auth) — match the existing scope.
2. Write the handler as a closure passed to `r.GET/POST/...` (the current style). Don't refactor to a separate `handler` type.
3. Validate inputs in-handler, using a `whereX` helper if the input is MAC/IPv4/date. Emit `c.JSON(http.StatusBadRequest, gin.H{"error": "..."})` on failure and `return` immediately.
4. Use the response shape from the existing endpoints. If you need a new type, add it in the same file with `json:"snake_case"` tags.
5. If the endpoint needs the daemon auth check, it's already wrapped by `TokenCheckerHeader` — no per-route opt-in needed.
6. Add an entry to the "Endpoints" section in [`README.md`](../../../../README.md) in the same change.
7. Add a test in the file's `*_test.go` using `gin.New()` + `httptest` + `testify/assert`. See [`testing.md`](./testing.md).

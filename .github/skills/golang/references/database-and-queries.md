# SQLite + GORM Query Patterns

## Authoritative Sources

- [GORM docs](https://gorm.io/docs/)
- [Go database/sql tutorial](https://go.dev/doc/database/)
- [SQLite docs](https://www.sqlite.org/docs.html)
- [mattn/go-sqlite3 driver](https://github.com/mattn/go-sqlite3)

## Layering

`main.go` opens `database/sql` first, then wraps it in GORM:

```go
db, err := sql.Open("sqlite3", databaseFilePath)
gormDb, err := gorm.Open(sqlite.Dialector{Conn: db})
```

This is intentional: the raw `*sql.DB` is the only way to do the `db.Ping()` startup check in `main.go` without dragging GORM into the boot path. After that, all queries go through GORM.

`lease_database.go` is the only place that talks to the database. New query endpoints belong there.

## Schema

Three tables, defined via `db.AutoMigrate(&Request{}, &Lease{}, &Client{})` at the top of `LeaseDatabase`. The schema matches the [reference SQLite schema](https://gist.github.com/amigus/6a9e4151d175d04bf05337b815f2213e) used by the `dhcp-script` hook.

| Table | Purpose | Keys |
|---|---|---|
| `requests` | Every DHCP request event | `(received, argument, mac, ipv4, client_id, requested_options)` — natural; no PK |
| `leases` | Active leases | PK `(mac, ipv4)` |
| `clients` | Aggregated client info | PK `mac` |

The test data lives in `lease_database_test_data.go` as a `const testDatabaseSQL`. This is fed into `:memory:` SQLite for handler tests. Don't move it to a fixture file — the SQL blob is the contract and it's easier to keep it versioned in Go.

## Query Patterns in the Codebase

**1. Active leases (joins clients onto leases):**

```go
db.Table("clients as c").
    Select("c.mac, c.hostname, c.client_id, c.vendor_class, leases.*").
    Joins("right join leases on c.mac = leases.mac").
    Scan(&active)
```

Right join is intentional: leases without a matching client row still show up, with `c.*` columns null. Don't switch to `INNER JOIN` "to be safer" — the existing semantics are part of the response shape.

**2. Address history for a MAC (in-Go aggregation):**

```go
query := db.Table("requests as r")
if query, ok = whereMac(c.Param, c, query, "mac", "r.mac", true); !ok {
    return
}
query.Select(...).Joins("RIGHT JOIN clients as c ON r.mac = c.mac").Order("r.received").Scan(&requests)

// Then aggregate into ipHistory map in Go
```

The codebase deliberately does some aggregation in Go (the `first_seen`/`last_seen` rollup for `/addresses/:mac` and `/devices/:ipv4`) rather than using a `GROUP BY`. Reasons:

- Keeps the SQL simple and inspectable
- The per-MAC/per-IPv4 result set is small (bounded by request history)
- Avoids SQLite-specific window functions

Keep the in-Go aggregation for new endpoints unless the row count is unbounded.

**3. `/clients` (LEFT JOIN with subquery):**

```go
subQuery := db.Table("requests").
    Select("mac, GROUP_CONCAT(DISTINCT ipv4 ORDER BY ipv4) as ipv4s").
    Group("mac")

query := db.Table("requests as r").
    Select("clients.*, COUNT(r.mac) as requests, sub.ipv4s").
    Joins("LEFT JOIN clients ON clients.mac = r.mac").
    Joins("LEFT JOIN (?) as sub ON clients.mac = sub.mac", subQuery).
    Group("r.mac").
    Order("requests")
```

`GROUP_CONCAT` is SQLite-specific. Since this codebase ships only on Linux with `go-sqlite3`, that's fine. If you ever add a Postgres backend, this needs to change.

**4. `/requests` (CIDR/range expansion in Go, then SQL `IN`):**

The `ipListFromExpression` helper uses `ipaddr.NewIPAddressString(...).ToAddress()` plus an iterator to enumerate every IP in a CIDR or range. That list is bound to a SQL `IN ?` clause. The list is also subnet-checked upstream — see [`http-handlers.md`](./http-handlers.md).

This is correct but **a very large CIDR (e.g. `/16`) will explode the IN list**. The default subnet is `/16`. The cap is implicit; if you need to harden, add a row-count guard before the bind.

## Scan Targets

The codebase uses anonymous structs as scan targets, not the top-level `Request`/`Lease`/`Client` types:

```go
var active []struct {
    Mac         string `json:"mac"`
    Hostname    string `json:"hostname"`
    ...
}
```

This is fine — the `Request`/`Lease`/`Client` types are there primarily for `AutoMigrate`, not for response shape. New endpoints should follow this pattern: declare the response shape inline.

**Edge case**: anonymous struct embedding for the `/clients` endpoint uses `Client` embedded in a `ClientRequests` wrapper. The embedded fields show up in JSON correctly. If you need an embedded type, follow this pattern.

## Errors

- **`query.Scan(&dst).Error`** is checked explicitly and turned into `500` JSON when non-nil. The codebase does this on every `Scan` after a query that could fail.
- **`query.Where(...).Scan(&dst)`** (no `.Error` check) appears in some of the simpler endpoints (e.g. `/leases`) where the only way `Scan` can fail is a closed connection — the handler then panics rather than returning a useful error. This is a known gap; new code should add `.Error` checks.

## Migrations

`db.AutoMigrate(&Request{}, &Lease{}, &Client{})` is called on every `LeaseDatabase(r, ...)` call. That's a no-op when the schema is already current. It will *not* drop columns, so destructive changes need a separate migration path. The author hasn't needed one yet; do not introduce one without an explicit ask.

## SQLite-Specific Tuning

The current code uses GORM defaults. If you need tuning:

- **`?_journal_mode=WAL`** for concurrent reads while a writer is active — would require switching from `sql.Open` to GORM-only opening
- **`?_busy_timeout=5000`** to wait for a busy lock instead of failing immediately
- **`?_foreign_keys=on`** to enforce referential integrity (currently OFF in tests; see `lease_database_test_data.go`)

These are all reasonable but the author hasn't enabled them. Discuss before changing.

## Testing Query Changes

See [`testing.md`](./testing.md). The pattern is:

1. `gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})`
2. `db.Exec(testDatabaseSQL)` (or your own fixture)
3. Pass the `*gorm.DB` to `LeaseDatabase(r, db, 30, "192.168.1.0/24")`
4. `httptest.NewRecorder()` + `r.ServeHTTP`
5. Assert the JSON body

Do not add tests that hit a real SQLite file. `:memory:` is fast, deterministic, and CI-friendly.

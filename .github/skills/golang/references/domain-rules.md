# DHCP / dnsmasq Domain Rules

This file captures the *domain* rules that apply regardless of how the Go code is structured. They come from [`copilot-instructions.md`](../../../copilot-instructions.md), [`README.md`](../../../../README.md), and the dhcp-hostsdir format spec.

## Reservation File Format

Files under the host directory (`-h` flag, default `/var/lib/misc/dnsmasq.reservations.d`) must be parseable by dnsmasq's `dhcp-hostsdir` directive. The format is:

```
MAC,set:tag1,set:tag2,IPv4[,hostname[,lease_time]]
```

Examples:

```
44:4f:8e:ce:fa:64,192.168.1.143,Adam-s-Phone
bc:32:b2:3b:13:d4,set:unsafe,192.168.1.9,Adam-s-Phone
00:1a:2b:3c:4d:5e,set:tag1,set:tag2,192.168.1.100,test-host,24h
```

Rules:

- **One reservation per file**, named after the normalized MAC.
- **MAC is always first**, colon-delimited.
- **Tags are `set:<name>`**, repeated as many times as needed. They're optional.
- **IPv4 is mandatory and present** (even when it's only a placeholder like `0.0.0.0` for "any address" — but the codebase requires a valid address).
- **Hostname and lease time are optional** and must appear in that order, never reversed.
- **No spaces** around commas.

The parser is in `internal/reservations/file.go`. The formatter is the single source of truth for output shape. New fields are not optional — they would break dnsmasq compatibility.

## MAC Normalization

- **Input formats accepted** by the validator: anything `seancfoley/ipaddress-go` accepts (colon, hyphen, dot). The validator in `host_dir.go` uses `ipaddr.NewMACAddressString(mac).ToAddress()`.
- **JSON output**: `mac.ToColonDelimitedString()` → lowercase, colon-separated. Always.
- **Filename**: `mac.ToNormalizedString()` → no separators, lowercase hex. Always.
- **SQL filter**: `mac.ToColonDelimitedString()` to match the JSON/DB representation. Always.

Mixing formats is a stated anti-pattern in [`copilot-instructions.md`](../../../copilot-instructions.md). If you find yourself wanting a "user-friendly" MAC like `44-4F-8E-CE-FA-64` somewhere, normalize it first, then store the normalized form.

## IPv4 Validation

- **Subnet check**: every request query (`/requests?cidr=`, `/requests?range=`) must be inside the configured `-s` subnet (default `192.168.0.0/16`). The check is in `ipListFromExpression` in `lease_database.go`.
- **No IPv6 anywhere**. The codebase is IPv4-only — `lease_database.go` types are `IPv4`, the URL params are `:ipv4`, and `internal/output/table.go` has a `preferredColumnWidth("ipv4", 15)` tuning. Adding IPv6 is a much bigger change than a column rename; don't do it casually.
- **IPv4 normalization** uses `ip.WithoutPrefixLen().ToNormalizedString()` for SQL binding.

## Token State Is In-Memory

- Tokens are issued by `auth.Checker`, which is a `sync.Mutex`-protected ring buffer of UUIDs with a use-count and a timeout.
- The publisher runs over a Unix domain socket (`-S` flag, default `/run/dnsmasq-web.sock`); the CLI client (`internal/client/auth.go`) reads from that socket locally or via SSH remotely.
- **No persistence**: tokens die on restart. This is a stated rule.
- **No new auth backend**. Don't add OAuth, JWT, or any other auth scheme without an explicit ask. The `X-Token` header is the contract.

## DHCP Request Semantics

The `requests` table mirrors the `dhcp-script` env vars:

- `argument` is one of `add`, `del`, `old`. `add` is a fresh lease; `del` is a release; `old` is a renewal/existing entry.
- `received` is the dnsmasq-supplied timestamp.
- `client_id` and `requested_options` are optional; the schema uses `NULL` when absent.

The current code treats `requests` as an event log. Aggregations (active leases, per-MAC history) are computed on read. The codebase has not implemented retention/TTL; the `-m` flag (max days) is parsed in `main.go` and threaded into `LeaseDatabase` but is currently unused in queries. Don't add retention behavior without checking with the maintainer — there's a risk of clobbering the active `leases` table.

## The SQLite Schema Is External

The `requests`/`leases`/`clients` tables are populated by an external [`dhcp-script` shim](https://gist.github.com/amigus/6a9e4151d175d04bf05337b815f2213e). This repo only **reads** from them and **writes** to the host directory. Two implications:

- **Column names and types are a contract with the shim**, not just with ourselves. Don't rename a column in GORM models without considering the shim.
- **`AutoMigrate` is intentionally additive**. It does not drop columns. The shim controls the schema.

# Getting Started with dnsmasq-web

A step-by-step operator guide for installing, wiring into dnsmasq, and using dnsmasq-web — the unified server and CLI that gives you a dashboard, a JSON API, and a CLI over your dnsmasq DHCP data.

## What you get

- A **dashboard** at `/` — pool utilization, leases, clients, devices (with your annotations), unacknowledged alerts, and reservations, in plain server-rendered HTML (no JavaScript).
- A **JSON API** — unversioned compatibility endpoints plus the canonical `/api/v1` device/annotation/alert/pool surface.
- A **CLI** — the same binary runs the server and the client commands.
- **Safe reservations** — validated, normalized, conflict-checked writes in dnsmasq `dhcp-hostsdir` format.

## Prerequisites

- A host running dnsmasq (the DHCP server) with root access for setup.
- SQLite installed on that host.
- A C compiler (`gcc`) only if you build the binary yourself; release binaries from the [releases page](https://github.com/amigus/dnsmasq-web/releases/latest) need no compiler.

## Step 1 — Install the lease database and script

dnsmasq-web reads what dnsmasq records, so dnsmasq must first keep its lease/request/client history in SQLite.
The [Dnsmasq Lease Database](https://gist.github.com/amigus/6a9e4151d175d04bf05337b815f2213e) gist provides the schema, the `dhcp-script`, and an installer:

```bash
curl -sLo - https://gist.githubusercontent.com/amigus/6a9e4151d175d04bf05337b815f2213e/raw/install.sh | sh
```

Or manually:

```bash
sqlite3 /var/lib/misc/dnsmasq.leases.db /path/to/database.sql
chown dnsmasq:dnsmasq /var/lib/misc/dnsmasq.leases.db
chgrp dnsmasq /var/lib/misc
chmod g+w /var/lib/misc
install -m 550 -o root -g dnsmasq /path/to/dnsmasq-dhcp.sh /usr/local/sbin
```

Then add to `dnsmasq.conf`:

```ini
dhcp-script=/usr/local/sbin/dnsmasq-dhcp.sh
script-on-renewal
leasefile-ro
```

Restart dnsmasq. From now on, every DHCP lease change, client, and request lands in `/var/lib/misc/dnsmasq.leases.db`.

## Step 2 — Install the binary

Download the appropriate binary from the [releases page](https://github.com/amigus/dnsmasq-web/releases/latest) to the DHCP server.

Or build locally:

```bash
bash scripts/build.sh debug    # dist/debug/dnsmasq-web; needs gcc (CGO)
bash scripts/build.sh release  # dist/release/; uses Docker
bash scripts/build.sh release linux-amd64-musl
                               # one target on the host, no Docker
```

## Step 3 — Create the host directory and start the server

Create the directory for reservation files (dnsmasq must read it):

```bash
sudo mkdir -p /etc/dnsmasq.d/reservations
sudo chown dnsmasq:dnsmasq /etc/dnsmasq.d/reservations
sudo chmod 750 /etc/dnsmasq.d/reservations
```

Start the server as a daemon:

```bash
sudo dnsmasq-web server -d -l :80 \
  -f /var/lib/misc/dnsmasq.leases.db \
  -h /etc/dnsmasq.d/reservations \
  -s 192.168.1.0/24 \
  -a 1m \
  -u dnsmasq -g dnsmasq
```

Flag reference (all optional except where noted):

| Flag | Purpose |
| --- | --- |
| `-f database-file` | SQLite lease database (required if `-h` is omitted) |
| `-h host-dir` | Reservation files directory (required if `-f` is omitted) |
| `-l address` | Listen address, e.g. `:80` (required) |
| `-s subnet` | Subnet for `/requests` scope checks and `/pool` utilization, e.g. `192.168.1.0/24` |
| `-a interval` | New-device alert scan interval, e.g. `1m`; `0` (default) disables alerting |
| `-d` | Fork and run as a daemon |
| `-u user` / `-g group` | Drop privileges after binding (requires root) |
| `-T max-tokens` | Max tokens issued at a time; `0` disables token checking |
| `-c max-uses` | Max uses per token; `0` (default) means unlimited |
| `-t timeout` | Token lifetime; `0` (default) means forever |
| `-P pid-file` | PID file (default `/run/dnsmasq-web.pid`) |
| `-S unix-socket` | Token socket (default `/run/dnsmasq-web.sock`) |
| `-E` | Preserve environment when daemonizing |
| `-v` | Verbose output |

Then add to `dnsmasq.conf`:

```ini
# Serve the reservation files dnsmasq-web writes
# (dnsmasq re-reads this directory automatically)
dhcp-hostsdir=/etc/dnsmasq.d/reservations
```

Restart dnsmasq again. Reservations you create through the API, CLI, or dashboard workflow are now live.

## Step 4 — Authenticate

With token checking enabled (the default in daemon mode), fetch a token from the Unix socket:

```bash
token=$(curl -s --unix-socket /run/dnsmasq-web.sock ./)
```

- **API:** send it as the `Authorization` header: `curl -H "Authorization: $token" http://dhcp/leases`
- **CLI:** pass `--token "$token"`, or just run a client command — the CLI fetches a token from the socket automatically (locally, or over SSH for remote servers) and re-authenticates on 401.
- **Dashboard:** open `http://dhcp/login`, paste the token, and you're in a browser-session cookie (`HttpOnly`, `SameSite=Strict`). `POST /logout` clears it. Login does not consume a token use.

Tokens are in-memory only: restarting the server invalidates all of them — fetch a fresh one. `-T 0` disables token checking entirely (then the CLI needs `--no-auth` or `DNSMASQ_WEB_NO_AUTH=1`).

## Step 5 — Take the tour

**Dashboard first.** Open `http://dhcp/` in a browser: pool strip, counters, and tables for leases, clients, devices, alerts, and reservations. Sections show a notice instead of a table when `-s`/`-h` are not configured.

**CLI tour:**

```bash
dnsmasq-web leases
dnsmasq-web clients --since 2024-09-01
dnsmasq-web devices list
dnsmasq-web devices get bc:32:b2:3b:13:d4
dnsmasq-web devices annotation set bc:32:b2:3b:13:d4 --name "Living-room TV" --tag media
dnsmasq-web alerts list --unacknowledged
dnsmasq-web alerts watch --interval 30s
dnsmasq-web pool
dnsmasq-web requests --cidr 192.168.1.0/24
dnsmasq-web addresses bc:32:b2:3b:13:d4 --output json | jq
dnsmasq-web reservations list --output json
```

**The pin workflow** — the core write job:

```bash
# Pin a device from its current lease in one step
# (from_lease is mutually exclusive with --ipv4; no current lease is a 400)
dnsmasq-web reservations pin bc:32:b2:3b:13:d4

# Or reserve explicitly (conflict-checked: reserved conflicts are hard 409,
# leased conflicts return 409 overridable with force=1)
dnsmasq API: POST /reservations {"mac":"...","ipv4":"..."}
```

**API tour:**

```bash
curl -H "Authorization: $token" http://dhcp/api/v1/devices
curl -H "Authorization: $token" 'http://dhcp/api/v1/alerts?unacknowledged=1'
curl -H "Authorization: $token" http://dhcp/api/v1/pool
```

## Step 6 — Day-2 operations

- **Restart the server:** tokens are in-memory only — every dashboard session and CLI token dies; fetch a fresh token from the socket. The PID file is silently overwritten on restart (stale PID files are harmless).
- **Back up reservations:** they are plain files in the host directory — copy the directory. (Export/import is roadmap SP-10, not built.)
- **Back up history:** copy the SQLite database file.
- **Update the binary:** stop the daemon, replace the binary, start it again.
- **Upgrade the database schema:** the server runs migrations at startup (AutoMigrate); no manual step. Your existing gist database is compatible as-is: the gist tables (`requests`, `leases`, `clients`) are only ever extended in place, never altered destructively, and the new tables (`device_annotations`, `alerts`, `device_alert_state`) are created on first start. Back up the file first if you want a rollback point.
- **Monitor:** `GET /api/v1/pool` (or `dnsmasq-web pool`) for utilization; `dnsmasq-web alerts watch` for new-device events. Health/metrics endpoint is roadmap SP-11, not built.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| `failed to get token from socket` | Server running? socket path (`-S`, default `/run/dnsmasq-web.sock`)? socket permissions? |
| `Unauthorized token` | Token expired (`-t`), exhausted (`-c`), or server restarted — fetch a fresh one; CLI re-fetches automatically on 401 |
| Dashboard login rejected | Token invalid/expired/exhausted — fetch a fresh one from the socket and retry |
| `/requests` returns 400 | Query outside `-s` subnet, or no `-s` configured |
| `/pool` returns 400 | No `-s` configured |
| 409 on reservation create | Reserved conflict (hard, no override) or leased conflict (retry with `force=1`) |
| `exactly one of ipv4 or from_lease is required` | Pass `--ipv4` or pin from lease, not both, not neither |
| 400 `no current lease; an explicit IPv4 is required` | The device has no active lease — reserve explicitly with `--ipv4` |
| Empty dashboard sections | No DHCP activity recorded yet — verify dnsmasq wiring from Step 1 (`dhcp-script`, `script-on-renewal`, `leasefile-ro`) |
| Build fails with CGO errors | Missing host C compiler — install `gcc` (or set `CC`); release builds are unaffected (Docker) |

## Where to go next

- Full client reference: [`CLI.md`](CLI.md)
- API surface and examples: [`README.md`](README.md)
- Build/packaging details: [`IMPLEMENTATION.md`](IMPLEMENTATION.md)
- Roadmap (SP-7..SP-11 not yet built): [`dnsmasq-web-review/product-proposals.md`](dnsmasq-web-review/product-proposals.md)

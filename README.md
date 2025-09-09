# Dnsmasq Web

[![Release](https://github.com/amigus/dnsmasq-web/actions/workflows/release.yml/badge.svg)](https://github.com/amigus/dnsmasq-web/actions/workflows/release.yml)
[![Run Tests](https://github.com/amigus/dnsmasq-web/actions/workflows/test.yml/badge.svg)](https://github.com/amigus/dnsmasq-web/actions/workflows/test.yml)

A JSON/HTTP interface for Dnsmasq and client interface written in POSIX shell.
It extends Dnsmasq using the `dhcp-script` and `dhcp-hostsdir` configuration parameters.
The script maintains client, lease and request information in an
[SQLite](https://www.sqlite.org/index.html)
[database](https://gist.github.com/amigus/6a9e4151d175d04bf05337b815f2213e).
The DHCP reservation data is stored in files under the host directory.

## Installation

1. Add the [database](https://gist.github.com/amigus/6a9e4151d175d04bf05337b815f2213e) to the DHCP server.
1. Download the appropriate binary from the [releases](https://github.com/amigus/dnsmasq-web/releases/latest) page to the DHCP server.
1. Run it, e.g., `dnsmasq-web` or as a daemon with `sudo dnsmasq-web -d -l :80 -T 0`.

## Client

The [cli](cli) directory contains a client interface written in POSIX shell.

The scripts define functions when sourced into the environment, e.g.:

```sh
. /path/to/dnsmasq-web.env
dnsmasq_web_curl addresses/bc:32:b2:3b:13:d4 | jq
[
  {
    "ipv4": "192.168.1.9",
    "first_seen": "2024-09-03 10:07:21",
    "last_seen": "2024-09-03 12:37:22",
    "requested_options": "",
    "hostname": "Adam-s-Phone",
    "vendor_class": "android-dhcp-14"
  }
]
```

## Endpoints

| Endpoint       | Method | Query Parameter         | Required | Description                                      |
|----------------|--------|-------------------------|----------|--------------------------------------------------|
| **/reservations** |     |                         |          |                                                  |
|                | GET    | mac                     | No       | Retrieve one or the entire list of reservations  |
|                | POST   |                         |          | Create a new reservation                         |
|                | PUT    | mac                     | Yes      | Update an existing reservation by MAC address    |
|                | DELETE | mac                     | Yes      | Delete a reservation by MAC address              |
| **/leases**    |        |                         |          |                                                  |
|                | GET    |                         |          | Retrieve lease information                       |
| **/clients**   |        |                         |          |                                                  |
|                | GET    | since=YYYY-mm-dd        | No       | Retrieve clients, optionally filtered by a date  |
| **/addresses** |        |                         |          |                                                  |
|                | GET    |                         |          | Retrieve IPv4 addresses used by a MAC address    |
| **/devices**   |        |                         |          |                                                  |
|                | GET    |                         |          | Retrieve MACs that used a specific IPv4 address  |
| **/requests**  |        |                         |          |                                                  |
|                | GET    | cidr                    | Yes      | Retrieve requests filtered by CIDR               |
|                | GET    | range                   | Yes      | Retrieve requests filtered by range              |

## Examples

### Reservations

```bash
echo '{}' |
jq '.mac = "bc:32:b2:3b:13:d4" | .ipv4 = "192.168.1.9" | .hostname = "Adam-s-Phone" | .tags = ["unsafe"]' |
curl -s http://dhcp/reservations -X POST -d @-
{"message":"success"}
curl -s http://dhcp/reservations/bc:32:b2:3b:13:d4 | jq
{
  "mac": "bc:32:b2:3b:13:d4",
  "tags": [
    "unsafe"
  ],
  "ipv4": "192.168.1.9",
  "hostname": "Adam-s-Phone"
}

curl -s http://dhcp/reservations/bc:32:b2:3b:13:d4 |
jq '.tags = ["safe"]' |
curl -s http://dhcp/reservations/bc:32:b2:3b:13:d4 -X PUT -d @-
{"message":"success"}
curl -s http://dhcp/reservations/bc:32:b2:3b:13:d4 | jq
{
  "mac": "bc:32:b2:3b:13:d4",
  "tags": [
    "safe"
  ],
  "ipv4": "192.168.1.9",
  "hostname": "Adam-s-Phone"
}
curl -s http://dhcp/reservations/bc:32:b2:3b:13:d4 -X DELETE
{"message":"success"}
curl -s http://dhcp/reservations/bc:32:b2:3b:13:d4 | jq
{"error":"no such reservation"}
```

### Leases

Iterates the leases table. It takes no parameters.

```bash
curl -s http://dhcp/leases |
jq -r '.[] | [.mac,.ipv4,.renewed,.hostname,.vendor_class] | @tsv' |
column
44:4f:8e:8e:3d:7e       192.168.1.127           wiz_8e3d7e
bc:32:b2:3b:13:d4       192.168.1.9             Adam-s-Phone        android-dhcp-14
6c:29:90:fe:9f:2b       192.168.1.119           wiz_fe9f2b
44:4f:8e:8a:5a:6c       192.168.1.125           wiz_8a5a6c
6c:29:90:4c:7e:1d       192.168.1.121           wiz_4c7e1d
6c:29:90:4c:3b:8f       192.168.1.107           wiz_4c3b8f
6c:29:90:fc:4a:2c       192.168.1.105           wiz_fc4a2c
```

### Clients

Iterates the clients table but adds the total number of requests and requested IP addresses.
It takes a "since" date as the only parameter.

```bash
curl -s http://dhcp/clients |
jq -r '.[] | [.requests, .mac, .hostname, (.ipv4s|join(","))] | @tsv' |
column -t
6  84:28:59:86:57:36  192.168.1.208
6  bc:32:b2:3b:13:d4  Adam-s-Phone   192.168.1.9
7  44:4f:8e:a0:62:46  wiz_a06246     192.168.1.146
7  44:4f:8e:ce:fa:64  wiz_cefa64     192.168.1.143
7  64:b7:08:7a:44:10  amazon         192.168.1.90
7  6c:29:90:2a:a4:03  wiz_2aa403     192.168.1.105
7  6c:29:90:56:f3:b6  wiz_56f3b6     192.168.1.118
7  6c:29:90:69:82:b4  wiz_6982b4     192.168.1.113
7  6c:29:90:75:ac:b5  wiz_75acb5     192.168.1.107
7  6c:29:90:ca:8f:e0  wiz_ca8fe0     192.168.1.108
```

### Addresses and Devices

Iterates the IPv4 addresses requested by the mac and vice versa.
It provides first and last seen times, requested options and the vendor class.

```bash
curl -s 'http://dhcp/addresses/6c:29:90:4c:3b:8f' | jq
[
  {
    "ipv4": "192.168.1.107",
    "first_seen": "2024-04-30 15:57:04",
    "last_seen": "2024-09-05 12:37:33",
    "requested_options": "1,3,28,6,15,44,46,47,31,33,121,43",
    "hostname": "wiz_4c3b8f",
    "vendor_class": ""
  }
]
```

```bash
curl -s http://dhcp/devices/192.168.1.90 | jq
[
  {
    "mac": "64:b7:08:7a:44:10",
    "first_seen": "2024-09-03 10:14:25",
    "last_seen": "2024-09-03 13:02:04",
    "requested_options": "",
    "hostname": "amazon",
    "vendor_class": ""
  }
]
```

### Requests

Itemizes the history of requests for each IPv4 address queried.

Extracting the keys yields a list of currently allocated addresses.

```bash
curl -s 'http://dhcp/requests?cidr=192.168.1.0/24' | jq 'keys'
[
  "192.168.1.105",
  "192.168.1.107",
  "192.168.1.108",
  "192.168.1.113",
  "192.168.1.118",
  "192.168.1.143",
  "192.168.1.146",
  "192.168.1.208",
  "192.168.1.9",
  "192.168.1.90"
]
```

The full output is more detailed.

```bash
curl -s 'http://dhcp/requests?range=192.168.1.1-10' | jq
{
  "192.168.1.9": [
    {
      "mac": "bc:32:b2:3b:13:d4",
      "hostname": "Adam-s-Phone",
      "vendor_class": "android-dhcp-14",
      "requested_options": "",
      "requested": "2024-09-03 12:37:22"
    }
  ]
}
```

## Security

When running as a daemon with `-d`, the `-T`, `-c`, and `-t` options control the _TokenChecker_.

### TokenChecker

The TokenChecker implements a simple token management scheme,
an HTTP header checker,
and a token publishing endpoint.

```bash
-T int
        the maximum number of tokens to issue at a time (0 disables token checking) (default 1)
-c int
        the maximum number of times a token can be used (the default 0 means unlimited)
-t duration
        the duration a token is valid (the default 0 means forever)
```

Thus, by default, it generates a single token with unlimited reuse forever.

```bash
dnsmasq-web -d -l :80
```

For 3 tokens that expire after 8 hours, run:

```bash
dnsmasq-web -d -l :80 -T 3 -t 8h
```

For 10 token single-use tokens that expire after a 30 days, run:

```bash
dnsmasq-web -d -l :80 -T 10 -c 1 -t $((24*30))h
```

### Use

The token is sent via the `X-Token` custom HTTP header.

It publishes tokens on the `/run/dnsmasq-web.sock` UNIX socket by default.

Getting the token from there is straight-forward with cURL:

```bash
curl -s --unix-socket /run/dnsmasq-web.sock ./
ffab9080-0197-43c4-895a-80eff055d428
# save it to a variable
token=$(curl -s --unix-socket /run/dnsmasq-web.sock ./)
# create the header
header = "X-Token: $token"
# use cURL
curl -H "$header" -s http://dhcp/leases
```

These steps can be combined with SSH to allow remote access with tokens:

```bash
curl -H "X-Token: $(ssh -ntq dhcp curl -s --unix-socket /run/dnsmasq-web.sock .)" -s http://dhcp/leases
```

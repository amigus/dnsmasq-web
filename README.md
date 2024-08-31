# DNSMASQ Database Web

A RESTful Web API for
[DNSMASQ Lease Database](https://gist.github.com/amigus/6a9e4151d175d04bf05337b815f2213e).

It has 5 methods:

1. GET /leases
1. GET /clients[?since=YYYY-mm-dd]
1. GET /addresses/_mac_
1. GET /devices/_ip_
1. GET /requests?cidr=_cidr_ or /requests?range=_range_

## Leases

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

## Clients

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

## Addresses and Devies

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

## Requests

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
curl -s 'http://dhcp/requests?cidr=192.168.1.0/24' | jq
{
  "192.168.1.105": [
    {
      "mac": "6c:29:90:2a:a4:03",
      "hostname": "wiz_2aa403",
      "vendor_class": "",
      "requested_options": "",
      "requested": "2024-09-03 12:56:34"
    }
  ],
  "192.168.1.107": [
    {
      "mac": "6c:29:90:75:ac:b5",
      "hostname": "wiz_75acb5",
      "vendor_class": "",
      "requested_options": "",
      "requested": "2024-09-03 12:43:24"
    }
  ],
  "192.168.1.108": [
    {
      "mac": "6c:29:90:ca:8f:e0",
      "hostname": "wiz_ca8fe0",
      "vendor_class": "",
      "requested_options": "",
      "requested": "2024-09-03 13:05:36"
    }
  ],
  "192.168.1.113": [
    {
      "mac": "6c:29:90:69:82:b4",
      "hostname": "wiz_6982b4",
      "vendor_class": "",
      "requested_options": "",
      "requested": "2024-09-03 12:59:33"
    }
  ],
  "192.168.1.118": [
    {
      "mac": "6c:29:90:56:f3:b6",
      "hostname": "wiz_56f3b6",
      "vendor_class": "",
      "requested_options": "",
      "requested": "2024-09-03 12:45:33"
    }
  ],
  "192.168.1.143": [
    {
      "mac": "44:4f:8e:ce:fa:64",
      "hostname": "wiz_cefa64",
      "vendor_class": "",
      "requested_options": "",
      "requested": "2024-09-03 12:57:54"
    }
  ],
  "192.168.1.146": [
    {
      "mac": "44:4f:8e:a0:62:46",
      "hostname": "wiz_a06246",
      "vendor_class": "",
      "requested_options": "1,3,28,6",
      "requested": "2024-09-03 11:55:49"
    },
    {
      "mac": "44:4f:8e:a0:62:46",
      "hostname": "wiz_a06246",
      "vendor_class": "",
      "requested_options": "",
      "requested": "2024-09-03 12:51:03"
    }
  ],
  "192.168.1.208": [
    {
      "mac": "84:28:59:86:57:36",
      "hostname": "",
      "vendor_class": "android-dhcp-11",
      "requested_options": "",
      "requested": "2024-09-03 12:50:58"
    },
    {
      "mac": "84:28:59:86:57:36",
      "hostname": "",
      "vendor_class": "android-dhcp-11",
      "requested_options": "1,3,6,15,26,28,51,58,59,43,114",
      "requested": "2024-09-03 13:00:05"
    }
  ],
  "192.168.1.9": [
    {
      "mac": "bc:32:b2:3b:13:d4",
      "hostname": "Adam-s-Phone",
      "vendor_class": "android-dhcp-14",
      "requested_options": "",
      "requested": "2024-09-03 12:37:22"
    }
  ],
  "192.168.1.90": [
    {
      "mac": "64:b7:08:7a:44:10",
      "hostname": "amazon",
      "vendor_class": "",
      "requested_options": "",
      "requested": "2024-09-03 13:02:04"
    }
  ]
}
```

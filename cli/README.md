# Dnsmasq Web UNIX/Linux shell client

A composable collection of POSIX shell scripts that constitute a client of Dnsmasq Web.

## use.sh

Contains the `dnsmasq_web_use` command.
It checks whether the argument(s) are available as a shell command.
It prevents the definition of functions that have unmet prerequisites.
It must be "dotted" into the shell first.
It will set DNSMASQ_WEB_SERVER to `[::1]`, i.e., _localhost_, and,
DNSMASQ_WEB_SERVER_USER to _root_ unless they are already set.

```sh
. /path/to/cli/use.sh
```

## token.sh

`dnsmasq_web_token` gets a token from the UNIX domain socket.
When `$DNSMASQ_WEB_SERVER` is set to something other than the default,
"[::1]", i.e., the server is remote,
it will use `ssh -ntq` to run the command on `$DNSMASQ_WEB_SERVER`.
Consider configuring SSH to connect to it without prompting for a password/passphrase.

## curl.sh

`dnsmasq_web_curl` uses cURL to access Dnsmasq Web.
It does two things:

1. It uses `$DNSMASQ_WEB_SERVER` instead of requiring the full URL as an argument
1. It uses `$DNSMASQ_WEB_TOKEN` to add the `Authorization` header to each request

There are also two commands to manage _reservations_:

1. dnsmasq_web_reservations_add
1. dnsmasq_web_reservations_delete

## curl_jq.sh

`dnsmasq_web_curl_jq` processes the output of `dnsmasq_web_curl` using `jq`.
If has hooks to inject pre-built `jq` logic for formatting, processing and sorting.

## jq_commands.sh

Contains some pre-built `jq` arguments for _clients_, _leases_, and _reservations_.
It uses `eval` to define shell functions for:

1. dnsmasq_web_curl_clients
1. dnsmasq_web_clients
1. dnsmasq_web_curl_leases
1. dnsmasq_web_leases
1. dnsmasq_web_curl_reservations
1. dnsmasq_web_reservations

The shortened command presents the data as a table.

## reservations.sh

Adds commands to manage _reservations_ that also use `jq`:

1. dnsmasq_web_reservations_add
1. dnsmasq_web_reservations_change
1. dnsmasq_web_reservations_delete

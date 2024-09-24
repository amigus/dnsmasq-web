#!/bin/sh

# shellcheck disable=SC2034
dnsmasq_web_clients_jq_sort='sort_by(.requests)'
# shellcheck disable=SC2034
dnsmasq_web_clients_jq='
    .ipv4s |= (.|join(","))[0:30] |
    .vendor_class |= (.|gsub("[\\s]+"; " "))[0:15] |
    if .hostname == "" then
        .hostname="<empty>"
    end |
    if .vendor_class == "" then
        .vendor_class="<empty>"
    end |
    if (.ipv4s|length) > 28 then
        .ipv4s |= .[0:27] + "..."
    end |
    if (.vendor_class|length) > 13 then
        .vendor_class |= .[0:12] + "..."
    end |
    [.requests,.mac,.hostname,.ipv4s,.vendor_class]'
dnsmasq_sort_by_ipv4='sort_by(.ipv4|split("\\."; null)[3]|tonumber)'
# shellcheck disable=SC2034
dnsmasq_web_leases_jq_sort="$dnsmasq_sort_by_ipv4"
# shellcheck disable=SC2034
dnsmasq_web_leases_jq='
    if .hostname == "" then
        .hostname="<empty>"
    end |
    if .renewed == "" then
        .renewed=.added
    end |
    [.mac,.ipv4,.hostname,.added,.renewed,.age]'
# shellcheck disable=SC2034
dnsmasq_web_reservations_jq_sort="$dnsmasq_sort_by_ipv4"
# shellcheck disable=SC2034
dnsmasq_web_reservations_jq='
    [.mac,.ipv4,.hostname,(.tags|join(","))]'

dnsmasq_web_use dnsmasq_web_curl_jq || return
for noun in clients leases reservations; do
    eval "dnsmasq_web_curl_$noun() { dnsmasq_web_curl_jq $noun \"\$@\"; };
     dnsmasq_web_$noun() { dnsmasq_web_curl_$noun | dnsmasq_web_tabulate; }"
done

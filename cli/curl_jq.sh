#!/bin/sh

dnsmasq_web_use dnsmasq_web_curl jq || {
    echo no dnsmasq_web_curl_jq without jq &&
        return
}
dnsmasq_web_curl_jq() {
    noun="$(echo "$1" | sed -e 's|[/?].*||')"
    sort="$(eval echo "\$dnsmasq_web_${noun}_jq_sort")"
    process="$(eval echo "\$dnsmasq_web_${noun}_jq")"
    expression="."
    test -n "$sort" && expression="$sort | .[]"
    test -n "$process" && expression="$expression | $process"
    dnsmasq_web_curl "$@" | jq "$expression"
}

dnsmasq_web_use dnsmasq_web_curl column || {
    echo using cat in place of column for dnsmasq_web_tabulate >&2
    column() { cat; }
}
dnsmasq_web_tabulate() {
    jq -r '@tsv' | column -t
}

#!/bin/sh

dnsmasq_web_use dnsmasq_web_curl || return
dnsmasq_web_reservation_add() {
    mac="$1"
    shift

    if echo "$1" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'; then
        ipv4="$1"
        shift
        hostname="$1"
        [ -n "$hostname" ] && shift
        tags="$1"
        [ -n "$tags" ] && shift
        lease_time="$1"
    else
        tags="$1"
        [ -n "$tags" ] && shift
        lease_time="$1"
    fi

    {
        echo -n '{"mac":"'"$mac"'"'
        [ -n "$ipv4" ] && echo -n ',"ipv4":"'"$ipv4"'"'
        [ -n "$hostname" ] && echo -n ',"hostname":"'"$hostname"'"'
        [ -n "$tags" ] && echo -n ',"tags":["'"$tags"'"]'
        [ -n "$lease_time" ] && echo -n ',"lease_time":"'"$lease_time"'"'
        echo '}'
    } | dnsmasq_web_curl reservations -X POST -d @-
}

dnsmasq_web_reservation_delete() {
    dnsmasq_web_curl "reservations/$1" -X DELETE
}

dnsmasq_web_use jq || return
dnsmasq_web_reservation_change() {
    mac="$1"
    shift
    dnsmasq_web_curl "reservations/$mac" |
        jq ".$1 = $2" |
        dnsmasq_web_curl "reservations/$mac" -X PUT -d @-
}

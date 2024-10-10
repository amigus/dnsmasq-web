#!/bin/sh

dnsmasq_web_use dnsmasq_web_curl || return
dnsmasq_web_reservation_add() {
    {
        echo '{"mac":"'"$1"'","ipv4":"'"$2"'","hostname":"'"$3"'"'
        test -n "$4" && echo ',"tags":["'"$4"'"]'
        test -n "$5" && echo ',"lease_time":"'"$5"'"'
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

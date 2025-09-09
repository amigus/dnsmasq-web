#!/bin/sh
dnsmasq_web_use curl || {
    echo no dnsmasq_web_curl without curl && return
}

dnsmasq_web_requires_token() {
    test -n "$DNSMASQ_WEB_TOKEN" -o -n "$DNSMASQ_WEB_TOKEN_CMD"
} && ! dnsmasq_web_requires_token || dnsmasq_web_use dnsmasq_web_token || {
    echo dnsmasq_web_curl requires dnsmasq_web_token && return
}

dnsmasq_web_try_curl() {
    url="http://$DNSMASQ_WEB_SERVER/${1%%/}"
    shift
    args='--fail-with-body --silent'
    dnsmasq_web_requires_token && args="$args --header 'Authorization: $(
        if test -n "$DNSMASQ_WEB_TOKEN"; then
            echo "$DNSMASQ_WEB_TOKEN"
        else
            dnsmasq_web_token
        fi
    )'"
    eval curl "$args" "$url" "$*"
}

dnsmasq_web_curl() {
    output=$(dnsmasq_web_try_curl "$@")
    test $? -ne 22 && echo "$output" && return
    if test "$output" = '{"error":"Unauthorized token"}'; then
        printf '%s expired; renewing' "$DNSMASQ_WEB_TOKEN" >&2
    else
        echo "$output" && return
    fi
    DNSMASQ_WEB_TOKEN=$(dnsmasq_web_token) &&
        dnsmasq_web_try_curl "$@"
}

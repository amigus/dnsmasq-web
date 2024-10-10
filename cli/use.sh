#!/bin/sh

dnsmasq_web_use() {
    for c in "$@"; do
        command -v "$c" >/dev/null
        test $? -eq 0 && continue
        echo "$c is not present" >&2
        return 1
    done
}

DNSMASQ_WEB_SERVER="${DNSMASQ_WEB_SERVER:-[::1]}"
DNSMASQ_WEB_SSH_USER="${DNSMASQ_WEB_SSH_USER:-root}"
export DNSMASQ_WEB_SERVER DNSMASQ_WEB_SSH_USER

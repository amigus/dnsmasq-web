#!/bin/sh

DNSMASQ_WEB_SERVER="${DNSMASQ_WEB_SERVER:-[::1]}" &&
export DNSMASQ_WEB_SERVER
DNSMASQ_WEB_TOKEN_SOCKET="${DNSMASQ_WEB_TOKEN_SOCKET:-/run/dnsmasq-web.sock}" &&
export DNSMASQ_WEB_TOKEN_SOCKET

dnsmasq_web_server_ip() {
    ip=$(echo "$1" | sed -e 's/^\[\(.*\)\]:.*$/\1/' -e 's/^\[\(.*\)\]$/\1/')
    test -n "$ip" || ip=$(expr "$1" : '\([0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\)')
    if test -n "$ip"; then echo "$ip" else echo "$1"; fi
}
dnsmasq_web_server_is_local() {
    ip=$(dnsmasq_web_server_ip "$1") && { test "$ip" = "::1" || test "$ip" = "127.0.0.1"; }
}

dnsmasq_web_server_is_local "$DNSMASQ_WEB_SERVER" ||
test -n "$DNSMASQ_WEB_SSH_SERVER" ||
DNSMASQ_WEB_SSH_SERVER=$(echo "$DNSMASQ_WEB_SERVER" | sed -e 's/^\[\(.*\)\]:.*$/\1/' -e 's/^\[\(.*\)\]$/\1/')
test -n "$DNSMASQ_WEB_SSH_SERVER" ||
    DNSMASQ_WEB_SSH_SERVER=$(expr "$DNSMASQ_WEB_SERVER" : '\([0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\)')
test -n "$DNSMASQ_WEB_SSH_SERVER" ||
    DNSMASQ_WEB_SSH_SERVER=$DNSMASQ_WEB_SERVER

test -n "$DNSMASQ_WEB_SSH_SERVER" || test -r "$DNSMASQ_WEB_TOKEN_SOCKET" || return

if test -n "$DNSMASQ_WEB_SSH_SERVER"; then
    ! dnsmasq_web_use ssh &&
    printf 'dnsmasq_web_token requires ssh to access %s' "$DNSMASQ_WEB_SERVER" &&
        return
else
    ! dnsmasq_web_use curl &&
    printf 'dnsmasq_web_token requires curl to access %s' "$DNSMASQ_WEB_TOKEN_SOCKET" &&
        return
fi

# Use cURL to get the token from the (local) UNIX domain socket:
DNSMASQ_WEB_TOKEN_CMD="curl --unix-socket $DNSMASQ_WEB_TOKEN_SOCKET -s ."
# Using ncat is noticeably slower but can be done like this:
# echo -e 'GET / HTTP/1.1\r\nHost: .\r\n' | ncat -U $DNSMASQ_WEB_TOKEN_SOCKET | tail -1

dnsmasq_web_token() {
    if test -n "$DNSMASQ_WEB_SSH_SERVER"; then
        server=$DNSMASQ_WEB_SSH_SERVER
        test -n "$DNSMASQ_WEB_SSH_USER" && server="${DNSMASQ_WEB_SSH_USER}@${server}"
        ssh -nqt "$server" "$DNSMASQ_WEB_TOKEN_CMD"
    else
        eval "$DNSMASQ_WEB_TOKEN_CMD"
    fi
}

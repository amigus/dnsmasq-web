#!/bin/sh

test -f /run/dnsmasq-web.sock -o -n "$DNSMASQ_WEB_SERVER" &&
    test -z "$DNSMASQ_WEB_TOKEN_SOCKET" && DNSMASQ_WEB_TOKEN_SOCKET=/run/dnsmasq-web.sock
test -n "$DNSMASQ_WEB_TOKEN_SOCKET" && {
    # Use cURL to get the token from the (local) UNIX domain socket:
    DNSMASQ_WEB_TOKEN_CMD="curl --unix-socket $DNSMASQ_WEB_TOKEN_SOCKET -s ."
    # Using ncat is noticeably slower but can be done like this:
    # echo -e 'GET / HTTP/1.1\r\nHost: .\r\n' | ncat -U $DNSMASQ_WEB_TOKEN_SOCKET | tail -1
    # Derive the SSH server from the web server:
    DNSMASQ_WEB_SSH_SERVER=$(echo "$DNSMASQ_WEB_SERVER" | sed -e 's|:[0-9]*$||' -e 's|\[\(.*\)\]|\1|')
    test -n "$DNSMASQ_WEB_SSH_SERVER" ||
        DNSMASQ_WEB_SSH_SERVER=$(expr "$DNSMASQ_WEB_SERVER" : '\([0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\)')
    test -n "$DNSMASQ_WEB_SSH_SERVER" ||
        DNSMASQ_WEB_SSH_SERVER=$DNSMASQ_WEB_SERVER

    dnsmasq_web_server_is_local() {
        test "$1" = "::1" || test "$1" = "127.0.0.1"
    } && dnsmasq_web_server_is_local "$server" || dnsmasq_web_use ssh || {
        printf 'dnsmasq_web_token requires ssh to access %s' "$DNSMASQ_WEB_SERVER" && return
    }

    dnsmasq_web_token() {
        if dnsmasq_web_server_is_local "$DNSMASQ_WEB_SSH_SERVER"; then
            eval "$DNSMASQ_WEB_TOKEN_CMD"
        else
            server=$DNSMASQ_WEB_SSH_SERVER
            test -n "$DNSMASQ_WEB_SSH_USER" && server="${DNSMASQ_WEB_SSH_USER}@${server}"
            ssh -nqt "$server" "$DNSMASQ_WEB_TOKEN_CMD"
        fi
    }
}

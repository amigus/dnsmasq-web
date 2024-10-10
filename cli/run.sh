#!/bin/sh

myname=$(basename "$0")
if command -v "$myname" >/dev/null 2>&1 && [ "$(type "$myname")" = "function" ]; then
    "$myname" "$@"
else
    echo "Function '$myname' is not defined."
    return 1
fi

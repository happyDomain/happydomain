#!/bin/sh

[ -z "${HAPPYDNS_SOCKET}" ] &&
    DEST="./happydns.sock" ||
        DEST="${HAPPYDNS_SOCKET}"

[ -S "${DEST}" ] && DEST="--unix-socket $DEST http://localhost"

RET=$(curl -s ${DEST}"$@")
CODE=$?

echo "$RET" | jq . 2> /dev/null ||
    echo "$RET"

exit $CODE

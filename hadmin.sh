#!/bin/sh

[ -z "${HAPPYDOMAIN_SOCKET}" ] &&
    DEST="./happydomain.sock" ||
        DEST="${HAPPYDOMAIN_SOCKET}"

[ -S "${DEST}" ] && DEST="--unix-socket $DEST http://localhost"

RET=$(curl -s ${DEST}"$@")
CODE=$?

echo "$RET" | jq . 2> /dev/null ||
    echo "$RET"

exit $CODE

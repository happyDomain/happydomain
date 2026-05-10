#!/bin/sh

[ -z "${HAPPYDOMAIN_SOCKET}" ] &&
    DEST="./happydomain.sock" ||
        DEST="${HAPPYDOMAIN_SOCKET}"

[ -S "${DEST}" ] && DEST="--unix-socket $DEST http://localhost"

RET=$(curl -s ${DEST}"$@")
CODE=$?

if [ -t 1 ]
then
    which jq > /dev/null 2> /dev/null &&
        echo "${RET}" | jq . 2> /dev/null ||
            echo "$RET"
else
    echo "$RET"
fi

exit $CODE

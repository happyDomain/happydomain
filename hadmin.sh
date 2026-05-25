#!/bin/sh

[ -z "${HAPPYDOMAIN_ADMIN_BIND}" ] &&
    DEST="./happydomain.sock" ||
        DEST="${HAPPYDOMAIN_ADMIN_BIND}"

if [ -S "${DEST}" ]; then
    DEST="--unix-socket $DEST http://localhost"
elif echo "$DEST" | grep -q ":"; then
    case "$DEST" in
        :*)          DEST="http://localhost${DEST}" ;;
        0.0.0.0:*)   DEST="http://localhost${DEST#0.0.0.0}" ;;
        \[::\]:*)    DEST="http://localhost:${DEST##*:}" ;;
        *)           DEST="http://${DEST}" ;;
    esac
fi

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

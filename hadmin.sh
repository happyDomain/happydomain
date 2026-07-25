#!/bin/sh

[ -z "${HAPPYDOMAIN_SOCKET}" ] &&
    DEST="./happydomain.sock" ||
        DEST="${HAPPYDOMAIN_SOCKET}"

[ -S "${DEST}" ] && DEST="--unix-socket $DEST http://localhost"

# When the admin interface is password-protected, exchange the password for a
# short-lived session token and carry it on the actual request.
#
# The variable is deliberately not named HAPPYDOMAIN_*: the server maps every
# HAPPYDOMAIN_* variable to a command line flag and refuses to start when no
# such flag exists, so it could not be shared with the server's environment.
if [ -n "${HADMIN_PASSWORD}" ]; then
    LOGIN=$(curl -s ${DEST}/api/admin-login \
        -H "Content-Type: application/json" \
        -d "{\"password\":$(printf '%s' "${HADMIN_PASSWORD}" | sed 's/\\/\\\\/g; s/"/\\"/g; s/^/"/; s/$/"/')}")

    TOKEN=$(printf '%s' "${LOGIN}" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')

    if [ -z "${TOKEN}" ]; then
        # Report the server message: login may also have been refused because
        # too many attempts were made, in which case the password is fine and
        # the right move is to wait.
        ERRMSG=$(printf '%s' "${LOGIN}" | sed -n 's/.*"errmsg":"\([^"]*\)".*/\1/p')
        echo "hadmin: admin login failed: ${ERRMSG:-check HADMIN_PASSWORD}" >&2
        exit 1
    fi

    RET=$(curl -s -H "Authorization: Bearer ${TOKEN}" ${DEST}"$@")
else
    RET=$(curl -s ${DEST}"$@")
fi
CODE=$?

echo "$RET" | jq . 2> /dev/null ||
    echo "$RET"

exit $CODE

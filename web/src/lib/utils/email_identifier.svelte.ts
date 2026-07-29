// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { getEmailIdentifier } from "$lib/api/email_identifier";
import { digestHex, hasSubtleCrypto } from "$lib/utils/crypto";

/**
 * Owner name prefix used by OPENPGPKEY (RFC 7929) and SMIMEA (RFC 8162)
 * records: the SHA-256 of the local-part, truncated to its 28 leftmost bytes,
 * in hexadecimal.
 *
 * Computed in the browser when possible. Web Crypto is only exposed in a
 * secure context, so an instance served over plain HTTP falls back to the
 * backend, which computes the very same hash.
 */
export async function computeEmailIdentifier(domainId: string, username: string): Promise<string> {
    const local = await digestHex("SHA-256", new TextEncoder().encode(username), 28);
    if (local !== undefined) return local;

    return getEmailIdentifier(domainId, username);
}

/**
 * Keeps an OPENPGPKEY/SMIMEA owner name prefix in sync with a username.
 *
 * The prefix and the username have to agree: the backend recomputes the hash
 * when generating records and rejects the zone with "Invalid prefix" when they
 * don't. So `hash` is only ever one of three things: the hash of the current
 * username, a value the user typed in themselves, or empty. It is never a hash
 * left over from a username that has since changed.
 *
 * `initial` is the prefix already stored on the record and `initialUsername`
 * the username it was derived from, so failing to recompute the hash of an
 * untouched record doesn't destroy a name that is still correct.
 */
export function createEmailIdentifierHasher(
    getUsername: () => string | undefined,
    getDomainId: () => string,
    initial: string,
    initialUsername: string,
) {
    // Hashing locally needs a secure context, unavailable when served over
    // plain HTTP. The backend computes the very same hash, so fall back to it,
    // but debounce: there, every keystroke costs a round trip.
    const local = hasSubtleCrypto();

    let hash = $state(initial);
    let error = $state("");

    // Set once the prefix has been dropped for want of a valid one, so callers
    // know to drop the record's owner name too. An empty `hash` on its own
    // doesn't mean that: it is also the state of a record whose first hash is
    // still being computed, whose name must be left alone meanwhile.
    let dropped = $state(false);

    // The username `hash` corresponds to. While it matches the current
    // username the prefix is trustworthy; as soon as it doesn't, the prefix
    // has to be replaced or dropped.
    let hashedUsername = initialUsername;

    // The last value we assigned to `hash` ourselves. Anything else in there
    // was typed by the user and has to survive a failed recomputation, or they
    // could never converge on a valid record after an error.
    let ours = initial;

    // Identifies the latest request, so a slow response for an outdated
    // username can't overwrite a fresher one.
    let seq = 0;

    // Drop a prefix that no longer matches the username, unless the user
    // supplied it by hand.
    function invalidate() {
        if (hashedUsername === (getUsername() ?? "")) return;
        if (hash !== ours) return;

        hash = "";
        ours = "";
        hashedUsername = "";
        dropped = true;
    }

    $effect(() => {
        const username = getUsername();

        if (!username) {
            // Nothing left to hash: drop any pending result and the stale
            // warning, so it doesn't outlive the condition that raised it.
            seq++;
            error = "";
            invalidate();
            return;
        }

        const timer = setTimeout(
            () => {
                const mine = ++seq;
                computeEmailIdentifier(getDomainId(), username)
                    .then((h) => {
                        if (mine !== seq) return;
                        error = "";
                        hash = h;
                        ours = h;
                        hashedUsername = username;
                        dropped = false;
                    })
                    .catch((e) => {
                        if (mine !== seq) return;
                        error = e.message || String(e);
                        invalidate();
                    });
            },
            local ? 0 : 400,
        );

        return () => clearTimeout(timer);
    });

    return {
        get hash() {
            return hash;
        },
        set hash(v: string) {
            // Typed in by the user: it stands until they change the username
            // again, and it makes the record whole once more.
            hash = v;
            dropped = false;
        },
        get error() {
            return error;
        },
        get dropped() {
            return dropped;
        },
    };
}

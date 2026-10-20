// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

import type { HappydnsSshHostKey } from "$lib/api-base/types.gen";
import { getRrtype, newRR, type dnsTypeSSHFP } from "$lib/dns_rr";

// SSHFP fingerprint types (RFC 4255 and RFC 6594).
export const SHA1 = 1;
export const SHA256 = 2;

/** Builds an SSHFP record attached to the name of the service. */
export function newSSHFP(algorithm: number, type: number, fingerprint: string): dnsTypeSSHFP {
    return {
        ...(newRR("", getRrtype("SSHFP")) as dnsTypeSSHFP),
        Algorithm: algorithm,
        Type: type,
        FingerPrint: fingerprint,
    };
}

/**
 * The host the fingerprints of a service describe: the name the service is
 * attached to, which is the one a client will look up. origin.domain is fully
 * qualified, and a trailing dot would be rejected by a TCP dial.
 */
export function defaultHost(dn: string, origin: string): string {
    return [dn, origin]
        .filter((l) => l && l !== "@")
        .join(".")
        .replace(/\.+$/, "");
}

/**
 * Publishes both fingerprints of each key: the SHA-256 one is what validating
 * clients use, the SHA-1 one is still asked for by a few deployments and costs
 * nothing to carry. Returns how many entries were added to `entries`.
 */
export function importKeys(entries: dnsTypeSSHFP[], keys: HappydnsSshHostKey[]): number {
    let added = 0;

    const add = (algorithm: number, type: number, fingerprint: string | undefined) => {
        const normalized = (fingerprint || "").toLowerCase();
        if (!normalized) return;

        const published = entries.some(
            (e) =>
                e.Algorithm === algorithm &&
                e.Type === type &&
                (e.FingerPrint || "").toLowerCase() === normalized,
        );
        if (published) return;

        entries.push(newSSHFP(algorithm, type, normalized));
        added++;
    };

    for (const key of keys) {
        // Publishing an algorithm 0 would be an invalid record.
        if (key.algorithm == null) continue;

        add(key.algorithm, SHA256, key.sha256);
        add(key.algorithm, SHA1, key.sha1);
    }

    return added;
}

// The statuses the backend reports, other than "ok" (see classifySSHError in
// internal/usecase/ssh_hostkeys.go). Each has a translated sentence.
export const KNOWN_STATUSES = ["blocked", "connect-error", "handshake-error", "timeout"];

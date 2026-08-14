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

import {
    asArray,
    type ComplianceContext,
    type ComplianceIssue,
    checkDigest,
    hasAddress,
    isCnameOwner,
    registerValidators,
} from "$lib/services/compliance";

interface SSHFP {
    algorithm: number;
    type: number;
    fingerprint: string;
}

const RFC1034 = "https://www.rfc-editor.org/rfc/rfc1034#section-3.6.2";
const RFC4255 = "https://www.rfc-editor.org/rfc/rfc4255#section-3.1";
const RFC6594 = "https://www.rfc-editor.org/rfc/rfc6594#section-2";

// RFC 4255, RFC 6594, RFC 8709: key algorithms, and the fingerprint types they
// may be published with.
const ALGORITHMS: Record<number, string> = {
    1: "RSA",
    2: "DSA",
    3: "ECDSA",
    4: "Ed25519",
    6: "Ed448",
};

// Fingerprint type to the length, in hexadecimal characters, of the digest.
const FINGERPRINT_LENGTHS: Record<number, number> = { 1: 40, 2: 64 };
const FINGERPRINT_NAMES: Record<number, string> = { 1: "SHA-1", 2: "SHA-256" };

const SHA1 = 1;
const SHA256 = 2;
const DSA = 2;

/**
 * An SSHFP is the one record a client checks before trusting a machine, and a
 * wrong one is indistinguishable from a man in the middle: the connection is
 * simply refused, with no hint as to why.
 */
function sshfpSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const records = asArray<SSHFP>(raw?.SSHFP).filter((r) => r && typeof r === "object");

    if (records.length === 0) {
        return [{ id: "sshfp.no-record", severity: "info" }];
    }

    // Which fingerprint types each algorithm is published with, to spot the
    // ones left on SHA-1 alone.
    const typesByAlgorithm = new Map<number, Set<number>>();
    const seen = new Set<string>();

    records.forEach((r, idx) => {
        const field = `SSHFP[${idx}]`;
        const fingerprint = (r.fingerprint ?? "").trim().toLowerCase();

        if (ALGORITHMS[r.algorithm] === undefined) {
            issues.push({
                id: "sshfp.unknown-algorithm",
                severity: "error",
                params: { algorithm: String(r.algorithm) },
                field,
                docUrl: RFC4255,
            });
        } else {
            const types = typesByAlgorithm.get(r.algorithm) ?? new Set<number>();
            types.add(r.type);
            typesByAlgorithm.set(r.algorithm, types);

            if (r.algorithm === DSA) {
                issues.push({
                    id: "sshfp.dsa-deprecated",
                    severity: "warning",
                    field,
                });
            }
        }

        const check = checkDigest(fingerprint, r.type, FINGERPRINT_LENGTHS);
        if (check.status === "unknown-type") {
            issues.push({
                id: "sshfp.unknown-type",
                severity: "error",
                params: { type: String(r.type) },
                field,
                docUrl: RFC6594,
            });
        } else if (check.status === "not-hex") {
            issues.push({
                id: "sshfp.invalid-fingerprint",
                severity: "error",
                field,
            });
        } else if (check.status === "bad-length") {
            issues.push({
                id: "sshfp.fingerprint-length",
                severity: "error",
                params: {
                    hash: FINGERPRINT_NAMES[r.type],
                    expected: check.expected,
                    got: check.got,
                },
                field,
            });
        }

        const key = `${r.algorithm}/${r.type}/${fingerprint}`;
        if (seen.has(key)) {
            issues.push({ id: "sshfp.duplicate", severity: "warning", field });
        }
        seen.add(key);
    });

    for (const [algorithm, types] of typesByAlgorithm) {
        if (types.has(SHA1) && !types.has(SHA256)) {
            issues.push({
                id: "sshfp.sha1-only",
                severity: "warning",
                params: { algorithm: ALGORITHMS[algorithm] },
                docUrl: RFC6594,
            });
        }
    }

    // The record is looked up on the name a user types after "ssh".
    if (isCnameOwner(ctx, ctx.dn)) {
        issues.push({
            id: "sshfp.owner-is-cname",
            severity: "error",
            params: { subdomain: ctx.dn || "@" },
            docUrl: RFC1034,
        });
    } else if (!hasAddress(ctx, ctx.dn) && ctx.zone !== null) {
        issues.push({
            id: "sshfp.no-address",
            severity: "warning",
            params: { subdomain: ctx.dn || "@" },
        });
    }

    // RFC 4255 sec. 5 only trusts these records under DNSSEC, but neither the
    // zone nor the domain carries a signing state to key a message off. A note
    // shown on every edit, that the user cannot act on from here, would only
    // drown the issues above.

    return issues;
}

registerValidators("svcs.SSHFPs", { sync: sshfpSync });

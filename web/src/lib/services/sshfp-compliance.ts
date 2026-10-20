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
} from "$lib/services/compliance";
import type { dnsTypeSSHFP } from "$lib/dns_rr";
import { SHA1, SHA256 } from "$lib/services/sshfp";

const RFC1034 = "https://www.rfc-editor.org/rfc/rfc1034#section-3.6.2";
const RFC4255 = "https://www.rfc-editor.org/rfc/rfc4255#section-3.1";
const RFC6594 = "https://www.rfc-editor.org/rfc/rfc6594#section-2";

// Key algorithm numbers (RFC 4255, 6594, 8709).
const ALGORITHMS: Record<number, string> = {
    1: "RSA",
    2: "DSA",
    3: "ECDSA",
    4: "Ed25519",
    6: "Ed448",
};

// Digest size in hexadecimal characters (2 per byte), per fingerprint type.
const FINGERPRINT_LENGTHS: Record<number, number> = { 1: 40, 2: 64 };
const FINGERPRINT_NAMES: Record<number, string> = { 1: "SHA-1", 2: "SHA-256" };

// Key algorithm number of DSA, in the IANA SSHFP registry.
const DSA = 2;

export interface SSHFPCheckOptions {
    /**
     * Reports a service publishing no fingerprint at all. Only svcs.SSHFPs
     * exists for them; on abstract.Server they are an optional extra.
     */
    requireRecord?: boolean;
    /**
     * Checks that the name carries an address. abstract.Server publishes its
     * own, so it skips it.
     */
    checkAddress?: boolean;
}

/**
 * A wrong SSHFP makes clients refuse the connection with no hint, same as a
 * man in the middle. Checks the SSHFP records of a svcs.SSHFPs or an
 * abstract.Server body.
 */
export function sshfpIssues(
    raw: Record<string, unknown>,
    ctx: ComplianceContext,
    { requireRecord = false, checkAddress = false }: SSHFPCheckOptions = {},
): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const records = asArray<dnsTypeSSHFP>(raw?.SSHFP);

    if (!records.some((r) => r && typeof r === "object")) {
        return requireRecord ? [{ id: "sshfp.no-record", severity: "info" }] : [];
    }

    // Which fingerprint types each algorithm is published with, to spot the
    // ones left on SHA-1 alone.
    const typesByAlgorithm = new Map<number, Set<number>>();
    const seen = new Set<string>();

    records.forEach((r, idx) => {
        // Skipped rather than filtered out beforehand, so that SSHFP[idx]
        // keeps pointing at the real position in the edited field.
        if (!r || typeof r !== "object") return;

        const field = `SSHFP[${idx}]`;
        // Normalised so a pasted, upper-case copy still counts as a duplicate.
        const fingerprint = String(r.FingerPrint ?? "")
            .trim()
            .toLowerCase();

        if (ALGORITHMS[r.Algorithm] === undefined) {
            issues.push({
                id: "sshfp.unknown-algorithm",
                severity: "error",
                params: { algorithm: String(r.Algorithm) },
                field,
                docUrl: RFC4255,
            });
        } else {
            // An unknown algorithm is already an error; grouping it would add a
            // misleading SHA-1 warning.
            const types = typesByAlgorithm.get(r.Algorithm) ?? new Set<number>();
            types.add(r.Type);
            typesByAlgorithm.set(r.Algorithm, types);

            if (r.Algorithm === DSA) {
                issues.push({
                    id: "sshfp.dsa-deprecated",
                    severity: "warning",
                    field,
                });
            }
        }

        const check = checkDigest(fingerprint, r.Type, FINGERPRINT_LENGTHS);
        if (check.status === "unknown-type") {
            issues.push({
                id: "sshfp.unknown-type",
                severity: "error",
                params: { type: String(r.Type) },
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
                    hash: FINGERPRINT_NAMES[r.Type],
                    expected: check.expected,
                    got: check.got,
                },
                field,
            });
        }

        const key = `${r.Algorithm}/${r.Type}/${fingerprint}`;
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
    } else if (checkAddress && !hasAddress(ctx, ctx.dn) && ctx.zone !== null) {
        // Without the zone we cannot tell whether an address exists, so stay
        // silent.
        issues.push({
            id: "sshfp.no-address",
            severity: "warning",
            params: { subdomain: ctx.dn || "@" },
        });
    }

    // No DNSSEC note on purpose, see docs/compliance-tests-goals.md.

    return issues;
}

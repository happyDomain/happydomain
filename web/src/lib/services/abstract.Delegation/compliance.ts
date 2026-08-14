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
    hasAddress,
    inZoneSubdomain,
    isCnameOwner,
    isValidHostname,
    normalizeFqdn,
    originFqdn,
    recordFqdn,
    registerValidators,
} from "$lib/services/compliance";

interface NS {
    Ns: string;
    Hdr?: { Name?: string };
}

interface DS {
    KeyTag: number;
    Algorithm: number;
    DigestType: number;
    Digest: string;
}

const RFC1034_NS = "https://www.rfc-editor.org/rfc/rfc1034#section-4.2.1";
const RFC2181 = "https://www.rfc-editor.org/rfc/rfc2181#section-10.3";
const RFC4034 = "https://www.rfc-editor.org/rfc/rfc4034#section-5.1";
const RFC8624 = "https://www.rfc-editor.org/rfc/rfc8624#section-3.3";

const HEX_RE = /^[0-9a-f]*$/i;

// Digest type to the length, in hexadecimal characters, of the digest it
// produces (RFC 4034 app. A.2, RFC 4509, RFC 5933, RFC 6605).
const DIGEST_LENGTHS: Record<number, number> = { 1: 40, 2: 64, 3: 64, 4: 96 };

// DNSKEY algorithms RFC 8624 sec. 3.1 marks as MUST NOT or NOT RECOMMENDED for
// signing: RSAMD5, DSA, RSASHA1, DSA-NSEC3-SHA1, RSASHA1-NSEC3-SHA1, ECC-GOST.
const DEPRECATED_ALGORITHMS = new Set([1, 3, 5, 6, 7, 12]);

function nameserverIssues(records: NS[], ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    // The delegated child, which the targets are compared against to decide
    // whether they need glue.
    const child = normalizeFqdn(recordFqdn(records[0]?.Hdr?.Name, ctx));
    const origin = originFqdn(ctx);

    if (records.length === 1) {
        issues.push({
            id: "ns.single-nameserver",
            severity: "warning",
            params: { subdomain: child },
            docUrl: RFC1034_NS,
        });
    }

    const seen = new Set<string>();
    records.forEach((r, idx) => {
        const target = (r.Ns ?? "").trim();
        const field = `ns[${idx}]`;
        const norm = normalizeFqdn(target);

        if (norm === "" || !isValidHostname(norm)) {
            issues.push({
                id: "ns.invalid-target",
                severity: "error",
                params: { target },
                field,
            });
            return;
        }

        if (seen.has(norm)) {
            issues.push({
                id: "ns.duplicate-target",
                severity: "warning",
                params: { target: norm },
                field,
            });
        }
        seen.add(norm);

        // A nameserver living inside the delegated subtree can only be reached
        // through a glue record published here, in the parent zone.
        const inChild = norm === child || norm.endsWith("." + child);
        const sub = inZoneSubdomain(norm, origin);

        if (inChild) {
            if (sub === null || !hasAddress(ctx, sub)) {
                issues.push({
                    id: "ns.missing-glue",
                    severity: "error",
                    params: { target: norm, subdomain: child },
                    field,
                    docUrl: RFC1034_NS,
                });
            }
            return;
        }

        if (sub === null) return;

        // RFC 2181 sec. 10.3: the target of an NS must be a host name, never an
        // alias.
        if (isCnameOwner(ctx, sub)) {
            issues.push({
                id: "ns.target-is-cname",
                severity: "error",
                params: { target: norm },
                field,
                docUrl: RFC2181,
            });
        } else if (!hasAddress(ctx, sub)) {
            issues.push({
                id: "ns.target-no-address",
                severity: "warning",
                params: { target: norm },
                field,
            });
        }
    });

    return issues;
}

function dsIssues(records: DS[]): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    const seen = new Set<string>();
    records.forEach((r, idx) => {
        const field = `ds[${idx}]`;
        const digest = (r.Digest ?? "").trim();

        if (!Number.isInteger(r.KeyTag) || r.KeyTag < 0 || r.KeyTag > 65535) {
            issues.push({
                id: "ns.ds-invalid-keytag",
                severity: "error",
                params: { keytag: String(r.KeyTag) },
                field,
            });
        }

        if (DEPRECATED_ALGORITHMS.has(r.Algorithm)) {
            issues.push({
                id: "ns.ds-deprecated-algorithm",
                severity: "warning",
                params: { algorithm: r.Algorithm },
                field,
                docUrl: RFC8624,
            });
        }

        const expected = DIGEST_LENGTHS[r.DigestType];
        if (expected === undefined) {
            issues.push({
                id: "ns.ds-unknown-digest-type",
                severity: "error",
                params: { digesttype: String(r.DigestType) },
                field,
                docUrl: RFC4034,
            });
        } else {
            if (r.DigestType === 1) {
                issues.push({
                    id: "ns.ds-sha1",
                    severity: "warning",
                    field,
                    docUrl: RFC8624,
                });
            }
            if (!HEX_RE.test(digest) || digest.length !== expected) {
                issues.push({
                    id: "ns.ds-digest-length",
                    severity: "error",
                    params: { expected, got: digest.length },
                    field,
                    docUrl: RFC4034,
                });
            }
        }

        const key = [r.KeyTag, r.Algorithm, r.DigestType, digest.toLowerCase()].join("/");
        if (seen.has(key)) {
            issues.push({
                id: "ns.ds-duplicate",
                severity: "warning",
                params: { keytag: String(r.KeyTag) },
                field,
            });
        }
        seen.add(key);
    });

    return issues;
}

/**
 * A delegation hands a subdomain over to another set of name servers: get it
 * wrong and the whole subtree disappears, which no amount of correct records
 * inside the child zone can fix.
 */
function delegationSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const nameservers = asArray<NS>(raw?.ns).filter((r) => r && typeof r === "object");
    const ds = asArray<DS>(raw?.ds).filter((r) => r && typeof r === "object");

    if (nameservers.length === 0) {
        // A lone DS gets the more specific message, which says what it breaks.
        const missing: ComplianceIssue =
            ds.length > 0
                ? { id: "ns.ds-without-ns", severity: "error", docUrl: RFC4034 }
                : { id: "ns.no-nameserver", severity: "error", docUrl: RFC1034_NS };

        return [missing, ...dsIssues(ds)];
    }

    return [...nameserverIssues(nameservers, ctx), ...dsIssues(ds)];
}

registerValidators("abstract.Delegation", { sync: delegationSync });

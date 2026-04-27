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

import {
    asArray,
    type ComplianceContext,
    type ComplianceIssue,
    registerValidators,
} from "$lib/services/compliance";

interface MX {
    Mx: string;
    Preference: number;
    Hdr?: { Name?: string };
}

const HOSTNAME_LABEL_RE = /^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$/;
const RFC5321 = "https://www.rfc-editor.org/rfc/rfc5321#section-5.1";
const RFC7505 = "https://www.rfc-editor.org/rfc/rfc7505";

function normalizeFqdn(name: string): string {
    return name.replace(/\.+$/, "").toLowerCase();
}

function isValidHostname(name: string): boolean {
    if (!name || name.length > 253) return false;
    const labels = name.split(".");
    return labels.every((l) => HOSTNAME_LABEL_RE.test(l));
}

/**
 * Returns the in-zone subdomain (relative to origin) for a target FQDN,
 * or null when the target is outside the edited zone.
 */
function inZoneSubdomain(target: string, originFqdn: string): string | null {
    const t = normalizeFqdn(target);
    const o = normalizeFqdn(originFqdn);
    if (!o) return null;
    if (t === o) return "";
    if (t.endsWith("." + o)) return t.slice(0, -(o.length + 1));
    return null;
}

function mxSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const records = asArray<MX>(raw?.mx).filter((r) => r && typeof r === "object");

    if (records.length === 0) return issues;

    const originFqdn: string = (ctx.origin as { domain?: string })?.domain ?? "";

    // RFC 7505 null MX: target ".", preference 0, MUST be the sole record.
    const nullMxes = records.filter((r) => r.Mx?.trim() === "." || r.Mx?.trim() === "");
    const hasNullMx = nullMxes.length > 0;
    if (hasNullMx && records.length > 1) {
        issues.push({
            id: "mx.null-mx-with-others",
            severity: "error",
            docUrl: RFC7505,
        });
    }
    for (const n of nullMxes) {
        if (n.Preference !== 0) {
            issues.push({
                id: "mx.null-mx-non-zero-preference",
                severity: "warning",
                params: { preference: n.Preference },
                docUrl: RFC7505,
            });
        }
    }

    // Per-record checks.
    const seen = new Map<string, number>();
    records.forEach((r, idx) => {
        const target = (r.Mx ?? "").trim();
        const field = `mx[${idx}]`;

        if (target === "" || target === ".") {
            // null MX, validated above.
            return;
        }

        const norm = normalizeFqdn(target);
        if (!isValidHostname(norm)) {
            issues.push({
                id: "mx.invalid-target",
                severity: "error",
                params: { target },
                field,
            });
            return;
        }

        // Preference is uint16 per RFC 1035.
        if (
            typeof r.Preference !== "number" ||
            !Number.isInteger(r.Preference) ||
            r.Preference < 0 ||
            r.Preference > 65535
        ) {
            issues.push({
                id: "mx.invalid-preference",
                severity: "error",
                params: { preference: String(r.Preference) },
                field,
            });
        }

        // Duplicate detection (case-insensitive on target).
        const prev = seen.get(norm);
        if (prev !== undefined) {
            issues.push({
                id: "mx.duplicate-target",
                severity: "warning",
                params: { target: norm, first: prev, second: idx },
                field,
            });
        } else {
            seen.set(norm, idx);
        }

        // Cross-zone checks when the target lives inside the edited zone.
        const sub = inZoneSubdomain(norm, originFqdn);
        if (sub === null) return;

        // RFC 5321 sec. 5.1: MX target must not be a CNAME.
        const cnames = ctx.findServices(sub, "svcs.CNAME");
        if (cnames.length > 0) {
            issues.push({
                id: "mx.target-is-cname",
                severity: "error",
                params: { target: norm },
                field,
                docUrl: RFC5321,
            });
        }

        // Heads-up when the in-zone target has no A/AAAA published.
        const servers = ctx.findServices(sub, "abstract.Server");
        if (servers.length === 0 && cnames.length === 0) {
            issues.push({
                id: "mx.target-no-address",
                severity: "warning",
                params: { target: norm },
                field,
            });
        }
    });

    return issues;
}

registerValidators("svcs.MXs", { sync: mxSync });

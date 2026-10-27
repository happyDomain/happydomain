// This file is part of the happyDomain (R) project.
// Copyright (c) 2026 happyDomain
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
    type ComplianceContext,
    type ComplianceIssue,
    inZoneSubdomain,
    isCnameOwner,
    isValidHostname,
    normalizeFqdn,
    originFqdn,
    registerValidators,
} from "$lib/services/compliance";

const RFC1912 = "https://www.rfc-editor.org/rfc/rfc1912#section-2.2";
const RFC2181 = "https://www.rfc-editor.org/rfc/rfc2181#section-10.3";
const RFC2308 = "https://www.rfc-editor.org/rfc/rfc2308#section-5";

// RFC 1912 sec. 2.2: below 20 minutes, secondaries poll for nothing.
const MIN_REFRESH = 1200;
// RFC 2308 sec. 5 advises one to three hours; past a day, a name added to the
// zone stays unknown to resolvers for longer than anyone expects.
const MAX_NEGATIVE_TTL = 86400;

/** SOA fields carrying a duration in seconds, with the id of their input. */
const TIMERS = [
    ["Refresh", "refresh"],
    ["Retry", "retry"],
    ["Expire", "expire"],
    ["Minttl", "nxttl"],
] as const;

function isUint32(v: unknown): v is number {
    return typeof v === "number" && Number.isInteger(v) && v >= 0 && v <= 0xffffffff;
}

/**
 * Builds the FQDN of a name of the SOA. Names living inside the zone travel
 * relative to its origin, "@" standing for the apex itself.
 */
function soaFqdn(name: string, origin: string): string {
    if (name === "@") return normalizeFqdn(origin);
    if (name.endsWith(".")) return normalizeFqdn(name);
    return normalizeFqdn(name + "." + normalizeFqdn(origin));
}

function mnameIssues(mname: string, ctx: ComplianceContext): ComplianceIssue[] {
    if (mname === "") return [{ id: "soa.empty-mname", severity: "error", field: "mname" }];

    const origin = originFqdn(ctx);
    const fqdn = soaFqdn(mname, origin);
    if (!isValidHostname(fqdn)) {
        return [{ id: "soa.invalid-mname", severity: "error", params: { mname }, field: "mname" }];
    }

    // Only a name of the edited zone can be checked: other zones are out of reach.
    const sub = inZoneSubdomain(fqdn, origin);
    if (sub !== null && isCnameOwner(ctx, sub)) {
        return [
            {
                id: "soa.mname-is-cname",
                severity: "error",
                params: { mname: fqdn },
                field: "mname",
                docUrl: RFC2181,
            },
        ];
    }

    return [];
}

function rnameIssues(rname: string, ctx: ComplianceContext): ComplianceIssue[] {
    if (rname === "") return [{ id: "soa.empty-rname", severity: "error", field: "rname" }];

    // The most common mistake by far: the address typed as is. "@" alone is
    // the apex, not an address.
    if (rname !== "@" && rname.includes("@")) {
        // Dots of the local part become escaped, or they would split it into labels.
        const at = rname.lastIndexOf("@");
        const local = rname.slice(0, at).replace(/\\?\./g, "\\.");
        return [
            {
                id: "soa.rname-has-at",
                severity: "error",
                params: { rname, suggestion: local + "." + rname.slice(at + 1) },
                field: "rname",
                docUrl: RFC1912,
            },
        ];
    }

    // The first label is the local part of the address: it may hold escaped
    // dots and about anything else, so only the domain part is checked.
    const fqdn = soaFqdn(rname, originFqdn(ctx));
    const domain = fqdn.match(/^(?:[^.\\]|\\.)+\.(.+)$/)?.[1];
    if (!domain || !isValidHostname(domain)) {
        return [{ id: "soa.invalid-rname", severity: "error", params: { rname }, field: "rname" }];
    }

    return [];
}

function timerIssues(soa: Record<string, any>): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    const invalid = TIMERS.filter(([key]) => !isUint32(soa[key]));
    for (const [key, field] of invalid) {
        issues.push({
            id: "soa.invalid-timer",
            severity: "error",
            params: { value: String(soa[key]) },
            field,
        });
    }
    // Comparing timers makes no sense until all of them are numbers.
    if (invalid.length > 0) return issues;

    const { Refresh, Retry, Expire, Minttl } = soa as Record<string, number>;

    if (Refresh < MIN_REFRESH) {
        issues.push({
            id: "soa.refresh-too-short",
            severity: "warning",
            params: { refresh: Refresh, min: MIN_REFRESH },
            field: "refresh",
            docUrl: RFC1912,
        });
    }

    if (Retry >= Refresh) {
        issues.push({
            id: "soa.retry-not-below-refresh",
            severity: "warning",
            params: { retry: Retry, refresh: Refresh },
            field: "retry",
            docUrl: RFC1912,
        });
    }

    // A secondary failing a single refresh would then drop the zone before
    // even trying again.
    if (Expire < Refresh + Retry) {
        issues.push({
            id: "soa.expire-too-short",
            severity: "warning",
            params: { expire: Expire, refresh: Refresh, retry: Retry },
            field: "expire",
            docUrl: RFC1912,
        });
    }

    if (Minttl > MAX_NEGATIVE_TTL) {
        issues.push({
            id: "soa.negative-ttl-too-long",
            severity: "warning",
            params: { minttl: Minttl, max: MAX_NEGATIVE_TTL },
            field: "nxttl",
            docUrl: RFC2308,
        });
    }

    return issues;
}

/** Validates the SOA of the zone: the names it carries and the timers secondaries obey. */
function originSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const soa = raw?.soa as Record<string, any> | undefined;
    if (!soa || typeof soa !== "object") return [];

    return [
        ...mnameIssues((soa.Ns ?? "").trim(), ctx),
        ...rnameIssues((soa.Mbox ?? "").trim(), ctx),
        ...timerIssues(soa),
    ];
}

registerValidators("abstract.Origin", { sync: originSync });

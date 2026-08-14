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
    type ComplianceContext,
    type ComplianceIssue,
    inZoneSubdomain,
    isCnameOwner,
    isValidDnsName,
    isValidHostname,
    normalizeFqdn,
    originFqdn,
    recordFqdn,
    registerValidators,
    TYPE_CNAME,
} from "$lib/services/compliance";

const RFC1034 = "https://www.rfc-editor.org/rfc/rfc1034#section-3.6.2";
const RFC6672 = "https://www.rfc-editor.org/rfc/rfc6672#section-2.4";

const TYPE_DNAME = 39;

/**
 * The target lives directly on the record for CNAME and DNAME, and under Data
 * for the pseudo-types (ALIAS, ANAME, R53_ALIAS, ...), which travel as a
 * private use record. Same reading as svcs.Alias/editor.svelte.
 */
export function aliasTarget(record: Record<string, any> | undefined): string {
    if (!record) return "";
    const rdata = record.Data as { Target?: string } | undefined;
    return ((rdata ? rdata.Target : (record.Target as string)) || "").trim();
}

interface CnameTargetOptions {
    /** Path of the target inside the edited value, for inline highlighting. */
    field: string;
    follow?: boolean;
    underscore?: boolean;
}

/**
 * Checks a CNAME-like target: it has to be a resolvable name that is neither
 * the owner itself, nor another alias, nor a name this zone leaves empty.
 * Shared with svcs.SpecialCNAME, which publishes CNAMEs under _service._proto.
 *
 * `follow` tells whether the resolver walks the target as an alias of its own.
 * It does for a CNAME and a DNAME; for the provider-resolved kinds (ALIAS,
 * ANAME, R53_ALIAS, ...) the provider flattens the target into addresses, so
 * neither the chain nor the emptiness of the target is a problem here.
 *
 * `underscore` opens the target to the attribute leaves of RFC 8552: a plain
 * alias points at a host, a SubAlias usually points at another service name.
 */
export function cnameTargetIssues(
    target: string,
    ownerFqdn: string,
    ctx: ComplianceContext,
    { field, follow = true, underscore = false }: CnameTargetOptions,
): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    if (target === "") {
        issues.push({ id: "alias.empty-target", severity: "error", field });
        return issues;
    }

    const norm = normalizeFqdn(target);
    if (!(underscore ? isValidDnsName : isValidHostname)(norm)) {
        issues.push({
            id: "alias.invalid-target",
            severity: "error",
            params: { target },
            field,
        });
        return issues;
    }

    if (norm === normalizeFqdn(ownerFqdn)) {
        issues.push({
            id: "alias.cname-loop",
            severity: "error",
            params: { target: norm },
            field,
            docUrl: RFC1034,
        });
        return issues;
    }

    if (!follow) return issues;

    // Only in-zone targets can be inspected; the rest is up to the runtime
    // checkers.
    const sub = inZoneSubdomain(norm, originFqdn(ctx));
    if (sub === null) return issues;

    if (isCnameOwner(ctx, sub)) {
        issues.push({
            id: "alias.cname-chain",
            severity: "warning",
            params: { target: norm },
            field,
            docUrl: RFC1034,
        });
    } else if (ctx.findServices(sub).length === 0) {
        issues.push({
            id: "alias.dangling-target",
            severity: "warning",
            params: { target: norm },
            field,
        });
    }

    return issues;
}

/**
 * A CNAME must be the only record of its name (RFC 1034 sec. 3.6.2), and a DNAME
 * cannot coexist with a CNAME at the same name (RFC 6672 sec. 2.4).
 *
 * The other kinds of alias exist precisely to lift that restriction: an ALIAS,
 * an R53_ALIAS and their kin are resolved by the provider into addresses, so
 * they may sit at the apex next to the NS, the MX and the TXT. That is why the
 * rule lives here, where the kind of alias is known, rather than in the Alone
 * restriction of the service.
 */
function aliasSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    const record = raw?.record as Record<string, any> | undefined;
    const rrtype = record?.Hdr?.Rrtype;
    if (rrtype === undefined) return issues;

    const follows = rrtype === TYPE_CNAME || rrtype === TYPE_DNAME;

    // Target checks apply to every kind of alias: even a provider-resolved
    // ALIAS needs a name that exists and is not itself.
    issues.push(
        ...cnameTargetIssues(aliasTarget(record), recordFqdn(record?.Hdr?.Name, ctx), ctx, {
            field: "record",
            follow: follows,
        }),
    );

    if (!follows) return issues;

    const siblings = ctx.findServices(ctx.dn).filter((service) => {
        if (service._svctype !== "svcs.Alias") return true;

        // Another alias on the same name is a conflict too, whichever kind.
        return (service.Service as Record<string, any> | undefined)?.record !== raw?.record;
    });

    if (rrtype === TYPE_CNAME) {
        // The apex always carries the SOA and the NS of the zone, so a CNAME
        // can never be alone there. This is exactly what the provider-resolved
        // kinds are for.
        if (!ctx.dn && !record?.Hdr?.Name) {
            issues.push({
                id: "alias.cname-at-apex",
                severity: "error",
                params: { domain: originFqdn(ctx) },
                field: "record",
                docUrl: RFC1034,
            });
        }

        // RFC 1034: nothing else may sit next to a CNAME.
        if (siblings.length > 0) {
            issues.push({
                id: "alias.cname-not-alone",
                severity: "error",
                params: { subdomain: ctx.dn || "@" },
                field: "record",
                docUrl: RFC1034,
            });
        }

        return issues;
    }

    // RFC 6672 sec. 2.4: a DNAME is only in conflict with a CNAME at the very
    // same owner name. It may perfectly well coexist with the other types, so
    // only the aliases of the subdomain are looked at.
    const conflicting = siblings.filter((service) => {
        if (service._svctype !== "svcs.Alias") return false;

        const record = (service.Service as Record<string, any> | undefined)?.record;
        return record?.Hdr?.Rrtype === TYPE_CNAME || record?.Hdr?.Rrtype === TYPE_DNAME;
    });

    if (conflicting.length > 0) {
        issues.push({
            id: "alias.dname-not-alone",
            severity: "error",
            params: { subdomain: ctx.dn || "@" },
            field: "record",
            docUrl: RFC6672,
        });
    }

    return issues;
}

registerValidators("svcs.Alias", { sync: aliasSync });

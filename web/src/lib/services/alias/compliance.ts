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
    registerValidators,
} from "$lib/services/compliance";

const RFC1034 = "https://www.rfc-editor.org/rfc/rfc1034#section-3.6.2";
const RFC6672 = "https://www.rfc-editor.org/rfc/rfc6672#section-2.4";

const TYPE_CNAME = 5;
const TYPE_DNAME = 39;

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

    const rrtype = raw?.record?.Hdr?.Rrtype;
    if (rrtype !== TYPE_CNAME && rrtype !== TYPE_DNAME) return issues;

    const siblings = ctx.findServices(ctx.dn).filter((service) => {
        if (service._svctype !== "svcs.Alias") return true;

        // Another alias on the same name is a conflict too, whichever kind.
        return (service.Service as Record<string, any> | undefined)?.record !== raw?.record;
    });

    if (rrtype === TYPE_CNAME) {
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

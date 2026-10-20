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
    recordFqdn,
    registerValidators,
} from "$lib/services/compliance";
import { cnameTargetIssues } from "$lib/services/svcs.Alias/compliance";

const RFC1034 = "https://www.rfc-editor.org/rfc/rfc1034#section-3.6.2";

/**
 * A SubAlias is a plain CNAME published under a _service._proto name, so it
 * obeys the very same rules as any other CNAME: a valid target, no loop, and
 * nothing else at its owner name.
 */
function specialCnameSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const record = raw?.cname as Record<string, any> | undefined;
    if (!record) return [];

    const owner: string = record.Hdr?.Name ?? "";
    const issues = cnameTargetIssues((record.Target ?? "").trim(), recordFqdn(owner, ctx), ctx, {
        field: "cname",
        underscore: true,
    });

    // The owner is the underscore name carried in the header, not the subdomain
    // the service is attached to, so ctx.dn cannot be reused as svcs.Alias does:
    // a SubAlias of _sip._tcp and one of _xmpp._tcp share the subdomain but are
    // not in conflict.
    const siblings = ctx.findServices(ctx.dn).filter((service) => {
        const other = (service.Service as Record<string, any> | undefined)?.cname;
        if (service._svctype === "svcs.SpecialCNAME") {
            return (
                other &&
                // reference check: skip the record being validated
                other !== record &&
                (other.Hdr?.Name ?? "").toLowerCase() === owner.toLowerCase()
            );
        }

        // Any other service of the subdomain shares the owner name only when it
        // does not carry an underscore name of its own.
        return owner === "";
    });

    if (siblings.length > 0) {
        issues.push({
            id: "alias.special-cname-not-alone",
            severity: "error",
            params: { subdomain: owner || ctx.dn || "@" },
            field: "cname",
            docUrl: RFC1034,
        });
    }

    return issues;
}

registerValidators("svcs.SpecialCNAME", { sync: specialCnameSync });

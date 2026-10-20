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
    isValidHostname,
    normalizeFqdn,
    originFqdn,
    recordFqdn,
    registerValidators,
} from "$lib/services/compliance";

const RFC1034 = "https://www.rfc-editor.org/rfc/rfc1034#section-3.6.2";
const RFC1035 = "https://www.rfc-editor.org/rfc/rfc1035#section-3.5";
const RFC3596 = "https://www.rfc-editor.org/rfc/rfc3596#section-2.5";

const IPV4_ARPA = "in-addr.arpa";
const IPV6_ARPA = "ip6.arpa";

const DECIMAL_RE = /^(0|[1-9][0-9]{0,2})$/;
const NIBBLE_RE = /^[0-9a-f]$/i;

/** The reverse tree the given FQDN belongs to, or null when it is a forward name. */
function reverseKind(fqdn: string): "ipv4" | "ipv6" | null {
    const name = normalizeFqdn(fqdn);
    if (name === IPV4_ARPA || name.endsWith("." + IPV4_ARPA)) return "ipv4";
    if (name === IPV6_ARPA || name.endsWith("." + IPV6_ARPA)) return "ipv6";
    return null;
}

/** A PTR mistake is invisible from the zone: it only shows up in mail servers, logs and ACLs. */
function ptrSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    const record = raw?.Record as Record<string, any> | undefined;
    if (!record) return issues;

    const owner = recordFqdn(record.Hdr?.Name, ctx);
    const kind = reverseKind(owner);

    if (kind === null) {
        // Perfectly legal, and occasionally deliberate, but almost always a
        // record that landed in the wrong zone.
        issues.push({
            id: "ptr.forward-zone",
            severity: "info",
            params: { subdomain: owner },
            docUrl: RFC1035,
        });
    } else {
        // The suffix is stripped by length, leaving a trailing dot that
        // filter(Boolean) drops. The IPv4 regex only bounds the digit count,
        // the range is checked numerically, and leading zeros are rejected
        // because tools disagree on whether they are octal.
        // RFC 2317 classless labels (0/25) are flagged on purpose: deferred.
        const suffix = kind === "ipv4" ? IPV4_ARPA : IPV6_ARPA;
        const labels = normalizeFqdn(owner).slice(0, -suffix.length).split(".").filter(Boolean);
        const valid = kind === "ipv4" ? DECIMAL_RE : NIBBLE_RE;
        const bad = labels.filter((l) => !valid.test(l) || (kind === "ipv4" && Number(l) > 255));

        if (bad.length > 0) {
            issues.push({
                id: "ptr.bad-reverse-label",
                severity: "warning",
                params: { label: bad[0], subdomain: owner },
                docUrl: kind === "ipv4" ? RFC1035 : RFC3596,
            });
        }
    }

    // The edited service is already in the zone, so drop it by identity to
    // keep it from counting as its own sibling. Another PTR at the same name
    // does count.
    const siblings = ctx.findServices(ctx.dn).filter((service) => {
        if (service._svctype !== "svcs.PTR") return true;
        return (service.Service as Record<string, any> | undefined)?.Record !== record;
    });
    if (siblings.length > 0) {
        issues.push({
            id: "ptr.not-alone",
            severity: "warning",
            params: { subdomain: ctx.dn || "@" },
            docUrl: RFC1035,
        });
    }

    const target = (record.Ptr ?? "").trim();
    // Nothing below makes sense without a valid target.
    if (target === "") {
        issues.push({ id: "ptr.empty-target", severity: "error", field: "ptr" });
        return issues;
    }

    const norm = normalizeFqdn(target);
    if (!isValidHostname(norm)) {
        issues.push({
            id: "ptr.invalid-target",
            severity: "error",
            params: { target },
            field: "ptr",
        });
        return issues;
    }

    if (!target.endsWith(".")) {
        // Without the final dot, some editors and some providers append the
        // reverse zone to the name, which turns it into nonsense.
        issues.push({
            id: "ptr.relative-target",
            severity: "info",
            params: { target },
            field: "ptr",
        });
    }

    // Only a target inside the edited zone can be checked: other zones are out of reach.
    const sub = inZoneSubdomain(norm, originFqdn(ctx));
    if (sub !== null && isCnameOwner(ctx, sub)) {
        issues.push({
            id: "ptr.target-is-cname",
            severity: "error",
            params: { target: norm },
            field: "ptr",
            docUrl: RFC1034,
        });
    }

    return issues;
}

registerValidators("svcs.PTR", { sync: ptrSync });

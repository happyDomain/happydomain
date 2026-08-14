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

import { fetchMTAStsPolicy } from "$lib/api/resolver";
import {
    type ComplianceContext,
    type ComplianceIssue,
    registerValidators,
} from "$lib/services/compliance";
import {
    RFC,
    fetchFailureIssue,
    policyIssues,
    policyURL,
    stripTrailingDot,
    txtRecordIssues,
} from "$lib/services/mta_sts";
import { DEFAULT_MAX_AGE, parseMTASTS } from "./model";

function cleanDomainOf(ctx: ComplianceContext): string {
    return stripTrailingDot(ctx.origin?.domain ?? "");
}

function policyCNAMEName(raw: Record<string, any>): string {
    return raw?.policyCNAME?.Hdr?.Name ?? "";
}

function hostingEnabled(raw: Record<string, any>): boolean {
    return Boolean(policyCNAMEName(raw));
}

function mtaStsHostedSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const url = policyURL(cleanDomainOf(ctx));
    const issues: ComplianceIssue[] = txtRecordIssues(raw?.txt, parseMTASTS);

    // Without the CNAME nothing points at happyDomain, so the policy below is
    // written but never served — the domain has half an MTA-STS setup.
    const cnameName = policyCNAMEName(raw);
    if (!cnameName) {
        issues.push({
            id: "mta_sts.hosted-no-policy-host",
            severity: "warning",
            docUrl: RFC + "#section-3.3",
        });
    } else if (!/^mta-sts(\.|$)/i.test(cnameName)) {
        issues.push({
            id: "mta_sts.hosted-wrong-policy-host",
            severity: "error",
            params: { name: cnameName },
            docUrl: RFC + "#section-3.3",
        });
    }

    // The policy happyDomain is going to serve gets the same scrutiny as one
    // fetched from the network: better to hear about it before publishing.
    issues.push(
        ...policyIssues(
            {
                version: "STSv1",
                mode: raw?.mode || "testing",
                mx: ((raw?.mx ?? []) as string[]).map((mx) => mx.trim()).filter(Boolean),
                maxAge: raw?.maxAge || DEFAULT_MAX_AGE,
            },
            ctx,
            url,
        ),
    );

    return issues;
}

/**
 * Compares the policy happyDomain would serve with the one actually reachable
 * on the network, which catches a CNAME that has not propagated, a missing
 * certificate, or an edit that was never applied to the zone.
 */
async function mtaStsHostedAsync(
    raw: Record<string, any>,
    ctx: ComplianceContext,
    signal: AbortSignal,
): Promise<ComplianceIssue[]> {
    // Nothing is expected to answer until the CNAME is published.
    if (!hostingEnabled(raw)) return [];

    const domain = cleanDomainOf(ctx);
    if (!domain) return [];

    const resp = await fetchMTAStsPolicy({ domain }, signal);

    const failure = fetchFailureIssue(resp);
    if (failure) return [failure];
    if (resp.status !== "ok") return [];

    const url = resp.url ?? policyURL(domain);
    const issues: ComplianceIssue[] = [];

    const wantMode = raw?.mode || "testing";
    if ((resp.mode ?? "") !== wantMode) {
        issues.push({
            id: "mta_sts.hosted-mode-mismatch",
            severity: "warning",
            params: { url, served: resp.mode ?? "", configured: wantMode },
            docUrl: RFC + "#section-3.2",
        });
    }

    // Both sides are patterns, not host names, so they are compared as sets of
    // normalized strings rather than matched against each other. Comparing as
    // actual Sets (not just arrays with a length check) matters: a duplicated
    // entry on one side must not make an otherwise-mismatched list look equal.
    const normalize = (mx: string) => mx.trim().toLowerCase().replace(/\.$/, "");
    const wantMx: string[] = (raw?.mx ?? []).map(normalize).filter(Boolean);
    const servedMx = (resp.mx ?? []).map(normalize).filter(Boolean);
    const wantMxSet = new Set(wantMx);
    const servedMxSet = new Set(servedMx);
    const sameMx =
        wantMxSet.size === servedMxSet.size && [...wantMxSet].every((mx) => servedMxSet.has(mx));
    if (!sameMx) {
        issues.push({
            id: "mta_sts.hosted-mx-mismatch",
            severity: "warning",
            params: { url, served: servedMx.join(", "), configured: wantMx.join(", ") },
            docUrl: RFC + "#section-3.2",
        });
    }

    return issues;
}

registerValidators("abstract.MTASTS", { sync: mtaStsHostedSync, async: mtaStsHostedAsync });

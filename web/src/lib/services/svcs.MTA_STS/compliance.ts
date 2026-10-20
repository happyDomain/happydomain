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
import { fetchFailureIssue, policyIssues, txtRecordIssues } from "$lib/services/mta_sts";
import { parseMTASTS } from "./model";

function mtaStsSync(raw: Record<string, any>, _ctx: ComplianceContext): ComplianceIssue[] {
    return txtRecordIssues(raw?.txt, parseMTASTS);
}

async function mtaStsAsync(
    _raw: Record<string, any>,
    ctx: ComplianceContext,
    signal: AbortSignal,
): Promise<ComplianceIssue[]> {
    const domain = ctx.origin?.domain;
    if (!domain) return [];
    const cleanDomain = domain.replace(/\.$/, "");
    if (!cleanDomain) return [];

    const resp = await fetchMTAStsPolicy({ domain: cleanDomain }, signal);

    // Anything but a successful fetch is the whole story: there are no policy
    // fields to validate behind it.
    const failure = fetchFailureIssue(resp);
    if (failure) return [failure];
    if (resp.status !== "ok") return [];

    return policyIssues(
        {
            version: resp.version,
            mode: resp.mode,
            mx: resp.mx,
            maxAge: resp.maxAge,
        },
        ctx,
        resp.url ?? "",
    );
}

registerValidators("svcs.MTA_STS", { sync: mtaStsSync, async: mtaStsAsync });

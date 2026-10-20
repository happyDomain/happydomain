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
    type ComplianceContext,
    type ComplianceIssue,
    registerValidators,
} from "$lib/services/compliance";
import { BIMI_DRAFT, dmarcIssues, selectorIssues } from "$lib/services/bimi";
import { HOST_CNAME_NAME, decodeBase64 } from "./model";

function bimiHostedSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    const name = raw?.txt?.Hdr?.Name;
    if (typeof name === "string" && name) issues.push(...selectorIssues(name));

    // Without the CNAME nothing points at happyDomain: the record would send
    // receivers to a host that does not serve the files.
    const cnameName = raw?.hostCNAME?.Hdr?.Name ?? "";
    if (cnameName !== HOST_CNAME_NAME) {
        issues.push({ id: "bimi.hosted-no-host", severity: "error", docUrl: BIMI_DRAFT });
    }

    // After a zone import, the record is there but the files are not: they
    // cannot be recovered from DNS.
    let logo = "";
    try {
        logo = new TextDecoder().decode(decodeBase64(raw?.logo));
    } catch {
        // Not base64: as good as no logo.
    }
    if (!logo) {
        issues.push({
            id: "bimi.hosted-no-logo",
            severity: "error",
            field: "logo",
            docUrl: BIMI_DRAFT,
        });
    } else if (!/<title[\s>]/i.test(logo)) {
        issues.push({
            id: "bimi.hosted-logo-no-title",
            severity: "warning",
            field: "logo",
            docUrl: BIMI_DRAFT,
        });
    }

    if (!raw?.vmc) {
        issues.push({ id: "bimi.missing-vmc", severity: "info", field: "vmc", docUrl: BIMI_DRAFT });
    }

    issues.push(...dmarcIssues(ctx));

    return issues;
}

registerValidators("abstract.BIMI", { sync: bimiHostedSync });

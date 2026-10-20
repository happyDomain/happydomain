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

/**
 * BIMI checks shared by svcs.BIMI, which only carries the DNS record, and
 * abstract.BIMI, which also has happyDomain host the logo and certificate.
 */

import type { ComplianceContext, ComplianceIssue } from "$lib/services/compliance";
import { parseDMARC } from "$lib/services/svcs.DMARC/model";

export const BIMI_DRAFT =
    "https://datatracker.ietf.org/doc/draft-brand-indicators-for-message-identification/";

const SELECTOR_LABEL_RE = /^[A-Za-z0-9_-]+$/;

/**
 * Checks the owner name of a BIMI record, which must be `<selector>._bimi`
 * relative to the domain, with a selector made of a single plain label.
 */
export function selectorIssues(name: string): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    let selector = "";
    if (name.endsWith("._bimi")) {
        selector = name.slice(0, -"._bimi".length);
    } else if (name !== "_bimi") {
        issues.push({
            id: "bimi.wrong-owner-name",
            severity: "error",
            params: { name },
            docUrl: BIMI_DRAFT,
        });
    }

    if (!selector) {
        issues.push({
            id: "bimi.missing-selector",
            severity: "error",
            field: "selector",
            docUrl: BIMI_DRAFT,
        });
    } else if (!SELECTOR_LABEL_RE.test(selector)) {
        issues.push({
            id: "bimi.invalid-selector",
            severity: "error",
            params: { selector },
            field: "selector",
            docUrl: BIMI_DRAFT,
        });
    }

    return issues;
}

/**
 * Cross-checks the DMARC policy of the zone: receivers only display a BIMI
 * logo for mail passing an enforcing (quarantine or reject) DMARC policy.
 *
 * Without the zone at hand, nothing can be said.
 */
export function dmarcIssues(ctx: ComplianceContext): ComplianceIssue[] {
    if (!ctx.zone) return [];

    const dmarcs = ctx.findAllServices("svcs.DMARC");
    if (dmarcs.length === 0) {
        return [{ id: "bimi.no-dmarc", severity: "warning", docUrl: BIMI_DRAFT }];
    }

    const policies = dmarcs
        .map((s) => {
            const t = (s.Service as Record<string, any>)?.txt;
            const tv = typeof t?.Txt === "string" ? t.Txt : "";
            return tv ? parseDMARC(tv).p : undefined;
        })
        .filter((p): p is string => Boolean(p));
    if (policies.length > 0 && policies.every((p) => p === "none")) {
        return [{ id: "bimi.weak-dmarc-policy", severity: "warning", docUrl: BIMI_DRAFT }];
    }

    return [];
}

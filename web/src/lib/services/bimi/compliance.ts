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
import { isBIMIDeclination, parseBIMI } from "$lib/services/bimi";
import { parseDMARC } from "$lib/services/dmarc";

const SELECTOR_LABEL_RE = /^[A-Za-z0-9_-]+$/;
const DRAFT = "https://datatracker.ietf.org/doc/draft-brand-indicators-for-message-identification/";

function isHttps(uri: string): boolean {
    return /^https:\/\//i.test(uri.trim());
}

function bimiSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const txt = raw?.txt;
    if (!txt) return issues;

    const txtValue: string = typeof txt.Txt === "string" ? txt.Txt : "";
    const name: string = typeof txt.Hdr?.Name === "string" ? txt.Hdr.Name : "";

    // Selector: owner name must be "<selector>._bimi" (relative to the apex).
    let selector = "";
    if (name.endsWith("._bimi")) {
        selector = name.slice(0, -"._bimi".length);
    } else if (name === "_bimi") {
        selector = "";
    } else {
        issues.push({
            id: "bimi.wrong-owner-name",
            severity: "error",
            params: { name },
            docUrl: DRAFT,
        });
    }

    if (!selector) {
        issues.push({
            id: "bimi.missing-selector",
            severity: "error",
            field: "selector",
            docUrl: DRAFT,
        });
    } else if (!SELECTOR_LABEL_RE.test(selector)) {
        issues.push({
            id: "bimi.invalid-selector",
            severity: "error",
            params: { selector },
            field: "selector",
            docUrl: DRAFT,
        });
    }

    if (!txtValue.trim()) return issues;

    const val = parseBIMI(txtValue);

    if (val.v !== undefined && val.v !== "" && val.v !== "BIMI1") {
        issues.push({
            id: "bimi.invalid-version",
            severity: "error",
            params: { version: val.v },
            field: "v",
            docUrl: DRAFT,
        });
    }

    // Declination: a domain that does not wish to participate publishes
    // v=BIMI1 with an empty l= tag. The URL/VMC checks no longer apply,
    // but the DMARC cross-check still does: a declining domain still has
    // to back the declination with an enforcing DMARC policy, otherwise
    // an attacker could spoof the domain and override the declination.
    const declination = isBIMIDeclination(txtValue);
    if (declination) {
        issues.push({
            id: "bimi.declination",
            severity: "info",
            docUrl: DRAFT,
        });
    }

    // l= (Location) is mandatory outside of declination.
    if (!declination) {
        if (!val.l) {
            issues.push({
                id: "bimi.missing-location",
                severity: "error",
                field: "l",
                docUrl: DRAFT,
            });
        } else if (!isHttps(val.l)) {
            issues.push({
                id: "bimi.location-not-https",
                severity: "error",
                field: "l",
                docUrl: DRAFT,
            });
        } else if (!/\.svg(\?|#|$)/i.test(val.l)) {
            issues.push({
                id: "bimi.location-not-svg",
                severity: "warning",
                field: "l",
                docUrl: DRAFT,
            });
        }
    }

    // a= (Authority / VMC) is optional but strongly recommended.
    if (declination) {
        // No-op: VMC is meaningless on a declination record.
    } else if (val.a) {
        if (!isHttps(val.a)) {
            issues.push({
                id: "bimi.authority-not-https",
                severity: "error",
                field: "a",
                docUrl: DRAFT,
            });
        } else if (!/\.pem(\?|#|$)/i.test(val.a)) {
            issues.push({
                id: "bimi.authority-not-pem",
                severity: "info",
                field: "a",
                docUrl: DRAFT,
            });
        }
    } else {
        issues.push({
            id: "bimi.missing-vmc",
            severity: "info",
            field: "a",
            docUrl: DRAFT,
        });
    }

    // e= (Evidence) optional, must be HTTPS if present.
    if (!declination && val.e && !isHttps(val.e)) {
        issues.push({
            id: "bimi.evidence-not-https",
            severity: "warning",
            field: "e",
            docUrl: DRAFT,
        });
    }

    // Cross-record check: BIMI requires DMARC with an enforcing policy.
    if (ctx.zone) {
        const dmarcs = ctx.findAllServices("svcs.DMARC");
        if (dmarcs.length === 0) {
            issues.push({
                id: "bimi.no-dmarc",
                severity: "warning",
                docUrl: DRAFT,
            });
        } else {
            const policies = dmarcs
                .map((s) => {
                    const t = (s.Service as Record<string, any>)?.txt;
                    const tv = typeof t?.Txt === "string" ? t.Txt : "";
                    return tv ? parseDMARC(tv).p : undefined;
                })
                .filter((p): p is string => Boolean(p));
            if (policies.length > 0 && policies.every((p) => p === "none")) {
                issues.push({
                    id: "bimi.weak-dmarc-policy",
                    severity: "warning",
                    docUrl: DRAFT,
                });
            }
        }
    }

    return issues;
}

registerValidators("svcs.BIMI", { sync: bimiSync });

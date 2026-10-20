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
import { BIMI_DRAFT as DRAFT, dmarcIssues, selectorIssues } from "$lib/services/bimi";
import { isBIMIDeclination, parseBIMI } from "./model";

function isHttps(uri: string): boolean {
    return /^https:\/\//i.test(uri.trim());
}

function bimiSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const txt = raw?.txt;
    if (!txt) return issues;

    const txtValue: string = typeof txt.Txt === "string" ? txt.Txt : "";
    const name: string = typeof txt.Hdr?.Name === "string" ? txt.Hdr.Name : "";

    issues.push(...selectorIssues(name));

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

    issues.push(...dmarcIssues(ctx));

    return issues;
}

registerValidators("svcs.BIMI", { sync: bimiSync });

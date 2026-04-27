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
import { parseDMARC, type DMARCValue } from "$lib/services/dmarc";

const POLICY_VALUES = new Set(["none", "quarantine", "reject"]);
const ALIGNMENT_VALUES = new Set(["r", "s"]);
const FO_VALUES = new Set(["0", "1", "d", "s"]);
const RF_VALUES = new Set(["afrf"]);
const RFC = "https://www.rfc-editor.org/rfc/rfc7489";

function isMailto(uri: string): boolean {
    return /^mailto:/i.test(uri.trim());
}

function isHttp(uri: string): boolean {
    return /^https?:/i.test(uri.trim());
}

function dmarcSync(raw: Record<string, any>, _ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const txt = raw?.txt;
    if (!txt) return issues;

    const txtValue: string = typeof txt.Txt === "string" ? txt.Txt : "";
    const name: string = typeof txt.Hdr?.Name === "string" ? txt.Hdr.Name : "";

    // The DMARC TXT must live at _dmarc.<domain>. The editor controls the
    // owner name, but a rename could land it elsewhere.
    if (name && !/^_dmarc(\.|$)/i.test(name)) {
        issues.push({
            id: "dmarc.wrong-owner-name",
            severity: "error",
            params: { name },
            docUrl: RFC + "#section-6.1",
        });
    }

    if (!txtValue.trim()) return issues;

    let val: DMARCValue;
    try {
        val = parseDMARC(txtValue);
    } catch {
        issues.push({ id: "dmarc.parse-error", severity: "error", field: "txt" });
        return issues;
    }

    // v=DMARC1 must be present and first (RFC 7489 sec. 6.3).
    if (!val.v) {
        issues.push({
            id: "dmarc.missing-version",
            severity: "error",
            field: "v",
            docUrl: RFC + "#section-6.3",
        });
    } else if (val.v !== "DMARC1") {
        issues.push({
            id: "dmarc.invalid-version",
            severity: "error",
            params: { version: val.v },
            field: "v",
            docUrl: RFC + "#section-6.3",
        });
    }

    // p= is mandatory (RFC 7489 sec. 6.3).
    if (!val.p) {
        issues.push({
            id: "dmarc.missing-policy",
            severity: "error",
            field: "p",
            docUrl: RFC + "#section-6.3",
        });
    } else if (!POLICY_VALUES.has(val.p)) {
        issues.push({
            id: "dmarc.invalid-policy",
            severity: "error",
            params: { policy: val.p },
            field: "p",
            docUrl: RFC + "#section-6.3",
        });
    } else if (val.p === "none") {
        issues.push({
            id: "dmarc.monitoring-only",
            severity: "info",
            field: "p",
            docUrl: RFC + "#section-6.3",
        });
    }

    // sp= subdomain policy.
    if (val.sp !== undefined && val.sp !== "" && !POLICY_VALUES.has(val.sp)) {
        issues.push({
            id: "dmarc.invalid-sp",
            severity: "error",
            params: { policy: val.sp },
            field: "sp",
        });
    }

    // adkim / aspf alignment.
    if (val.adkim !== undefined && val.adkim !== "" && !ALIGNMENT_VALUES.has(val.adkim)) {
        issues.push({
            id: "dmarc.invalid-alignment",
            severity: "error",
            params: { tag: "adkim", value: val.adkim },
            field: "adkim",
        });
    }
    if (val.aspf !== undefined && val.aspf !== "" && !ALIGNMENT_VALUES.has(val.aspf)) {
        issues.push({
            id: "dmarc.invalid-alignment",
            severity: "error",
            params: { tag: "aspf", value: val.aspf },
            field: "aspf",
        });
    }

    // pct must be 0..100.
    if (val.pct !== undefined && val.pct !== "" && val.pct !== null) {
        const pct = typeof val.pct === "number" ? val.pct : Number.parseInt(String(val.pct), 10);
        if (!Number.isInteger(pct) || pct < 0 || pct > 100) {
            issues.push({
                id: "dmarc.invalid-pct",
                severity: "error",
                params: { pct: String(val.pct) },
                field: "pct",
            });
        } else if (pct < 100) {
            issues.push({
                id: "dmarc.partial-deployment",
                severity: "info",
                params: { pct },
                field: "pct",
                docUrl: RFC + "#section-6.6.4",
            });
        }
    }

    // ri must be a positive integer.
    if (val.ri !== undefined && val.ri !== "") {
        const ri = Number.parseInt(String(val.ri), 10);
        if (!Number.isInteger(ri) || ri <= 0) {
            issues.push({
                id: "dmarc.invalid-ri",
                severity: "error",
                params: { ri: String(val.ri) },
                field: "ri",
            });
        }
    }

    // fo values must be in {0,1,d,s}. Combinations like "d:s" are allowed.
    for (const f of val.fo ?? []) {
        const trimmed = f.trim();
        if (!trimmed) continue;
        if (!FO_VALUES.has(trimmed)) {
            issues.push({
                id: "dmarc.invalid-fo",
                severity: "warning",
                params: { value: trimmed },
                field: "fo",
            });
        }
    }

    // rf format. Only "afrf" is defined.
    for (const r of val.rf ?? []) {
        const trimmed = r.trim();
        if (!trimmed) continue;
        if (!RF_VALUES.has(trimmed)) {
            issues.push({
                id: "dmarc.unknown-rf",
                severity: "warning",
                params: { value: trimmed },
                field: "rf",
            });
        }
    }

    // rua / ruf URIs.
    const uriCheck = (uri: string, tag: "rua" | "ruf") => {
        const u = uri.trim();
        if (!u) {
            issues.push({
                id: "dmarc.empty-uri",
                severity: "warning",
                params: { tag },
                field: tag,
            });
            return;
        }
        if (!isMailto(u) && !isHttp(u)) {
            issues.push({
                id: "dmarc.invalid-uri-scheme",
                severity: "error",
                params: { tag, uri: u },
                field: tag,
                docUrl: RFC + "#section-6.2",
            });
            return;
        }
        if (isMailto(u)) {
            const addr = u.replace(/^mailto:/i, "");
            // Strip optional !size suffix (RFC 7489 sec. 6.2 allows "!10m" etc.).
            const local = addr.split("!")[0];
            if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(local)) {
                issues.push({
                    id: "dmarc.invalid-mailto",
                    severity: "error",
                    params: { tag, uri: u },
                    field: tag,
                });
            }
        }
    };
    for (const u of val.rua ?? []) uriCheck(u, "rua");
    for (const u of val.ruf ?? []) uriCheck(u, "ruf");

    return issues;
}

registerValidators("svcs.DMARC", { sync: dmarcSync });

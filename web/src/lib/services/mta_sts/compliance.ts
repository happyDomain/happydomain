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
import { parseMTASTS, type MTASTSValue } from "$lib/services/mta_sts";

const RFC = "https://www.rfc-editor.org/rfc/rfc8461";
// RFC 8461 sec. 3.1: id is 1..32 alphanumeric characters.
const ID_RE = /^[A-Za-z0-9]{1,32}$/;
const VALID_MODES = new Set(["enforce", "testing", "none"]);
// RFC 8461 sec. 3.2: max_age is in [0, 31557600] (1 year).
const MAX_AGE_HARD_LIMIT = 31557600;
const MAX_AGE_RECOMMENDED_MIN = 86400; // sec. 3.2 recommends "at least one week" but anything below a day is suspicious.

function mtaStsSync(raw: Record<string, any>, _ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const txt = raw?.txt;
    if (!txt) return issues;

    const txtValue: string = typeof txt.Txt === "string" ? txt.Txt : "";
    const name: string = typeof txt.Hdr?.Name === "string" ? txt.Hdr.Name : "";

    // Owner name must be _mta-sts.<domain>.
    if (name && !/^_mta-sts(\.|$)/i.test(name)) {
        issues.push({
            id: "mta_sts.wrong-owner-name",
            severity: "error",
            params: { name },
            docUrl: RFC + "#section-3.1",
        });
    }

    if (!txtValue.trim()) return issues;

    let val: MTASTSValue;
    try {
        val = parseMTASTS(txtValue);
    } catch {
        issues.push({ id: "mta_sts.parse-error", severity: "error", field: "txt" });
        return issues;
    }

    if (!val.v) {
        issues.push({
            id: "mta_sts.missing-version",
            severity: "error",
            field: "v",
            docUrl: RFC + "#section-3.1",
        });
    } else if (val.v !== "STSv1") {
        issues.push({
            id: "mta_sts.invalid-version",
            severity: "error",
            params: { version: val.v },
            field: "v",
            docUrl: RFC + "#section-3.1",
        });
    }

    if (val.id === undefined || val.id === "") {
        issues.push({
            id: "mta_sts.missing-id",
            severity: "error",
            field: "id",
            docUrl: RFC + "#section-3.1",
        });
    } else if (!ID_RE.test(val.id)) {
        issues.push({
            id: "mta_sts.invalid-id",
            severity: "error",
            params: { id: val.id },
            field: "id",
            docUrl: RFC + "#section-3.1",
        });
    }

    return issues;
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

    const issues: ComplianceIssue[] = [];
    const resp = await fetchMTAStsPolicy({ domain: cleanDomain }, signal);
    const url = resp.url ?? "";

    switch (resp.status) {
        case "ok":
            break;
        case "dns-error":
            issues.push({
                id: "mta_sts.policy-dns-error",
                severity: "error",
                params: { url },
                docUrl: RFC + "#section-3.3",
            });
            return issues;
        case "tls-error":
            issues.push({
                id: "mta_sts.policy-tls-error",
                severity: "error",
                params: { url, error: resp.errorMsg ?? "" },
                docUrl: RFC + "#section-3.3",
            });
            return issues;
        case "not-found":
            issues.push({
                id: "mta_sts.policy-not-found",
                severity: "error",
                params: { url },
                docUrl: RFC + "#section-3.3",
            });
            return issues;
        case "http-error":
            issues.push({
                id: resp.redirected ? "mta_sts.policy-redirect" : "mta_sts.policy-http-error",
                severity: "warning",
                params: { url, code: resp.httpCode ?? 0 },
                docUrl: RFC + "#section-3.3",
            });
            return issues;
        case "fetch-error":
            issues.push({
                id: "mta_sts.policy-fetch-error",
                severity: "warning",
                params: { url, error: resp.errorMsg ?? "" },
            });
            return issues;
        case "too-large":
            issues.push({
                id: "mta_sts.policy-too-large",
                severity: "error",
                params: { url },
            });
            return issues;
        default:
            // Unknown status: ignore so a future backend addition does not
            // surface a localized "undefined" string.
            return issues;
    }

    // status === "ok": validate parsed policy fields.
    if (!resp.version) {
        issues.push({
            id: "mta_sts.policy-missing-version",
            severity: "error",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    } else if (resp.version !== "STSv1") {
        issues.push({
            id: "mta_sts.policy-invalid-version",
            severity: "error",
            params: { url, version: resp.version },
            docUrl: RFC + "#section-3.2",
        });
    }

    const mode = resp.mode ?? "";
    if (!mode) {
        issues.push({
            id: "mta_sts.policy-missing-mode",
            severity: "error",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    } else if (!VALID_MODES.has(mode)) {
        issues.push({
            id: "mta_sts.policy-invalid-mode",
            severity: "error",
            params: { url, mode },
            docUrl: RFC + "#section-3.2",
        });
    } else if (mode === "none") {
        issues.push({
            id: "mta_sts.policy-mode-none",
            severity: "warning",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    } else if (mode === "testing") {
        issues.push({
            id: "mta_sts.policy-mode-testing",
            severity: "info",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    }

    const mxList = resp.mx ?? [];
    if ((mode === "enforce" || mode === "testing") && mxList.length === 0) {
        issues.push({
            id: "mta_sts.policy-missing-mx",
            severity: "error",
            params: { url, mode },
            docUrl: RFC + "#section-3.2",
        });
    }

    const maxAge = resp.maxAge ?? 0;
    if (!maxAge) {
        issues.push({
            id: "mta_sts.policy-missing-max-age",
            severity: "error",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    } else if (maxAge < 0 || maxAge > MAX_AGE_HARD_LIMIT) {
        issues.push({
            id: "mta_sts.policy-invalid-max-age",
            severity: "error",
            params: { url, maxAge },
            docUrl: RFC + "#section-3.2",
        });
    } else if (maxAge < MAX_AGE_RECOMMENDED_MIN) {
        issues.push({
            id: "mta_sts.policy-short-max-age",
            severity: "warning",
            params: { url, maxAge },
            docUrl: RFC + "#section-3.2",
        });
    }

    return issues;
}

registerValidators("svcs.MTA_STS", { sync: mtaStsSync, async: mtaStsAsync });

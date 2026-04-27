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
import { parseTLSRPT } from "$lib/services/tlsrpt.svelte";

const RFC = "https://www.rfc-editor.org/rfc/rfc8460";

function isMailto(uri: string): boolean {
    return /^mailto:/i.test(uri.trim());
}
function isHttp(uri: string): boolean {
    return /^https?:/i.test(uri.trim());
}

function tlsrptSync(raw: Record<string, any>, _ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const txt = raw?.txt;
    if (!txt) return issues;

    const txtValue: string = typeof txt.Txt === "string" ? txt.Txt : "";
    const name: string = typeof txt.Hdr?.Name === "string" ? txt.Hdr.Name : "";

    // Owner name must be _smtp._tls.<domain> (RFC 8460 sec. 3).
    if (name && !/^_smtp\._tls(\.|$)/i.test(name)) {
        issues.push({
            id: "tlsrpt.wrong-owner-name",
            severity: "error",
            params: { name },
            docUrl: RFC + "#section-3",
        });
    }

    if (!txtValue.trim()) return issues;

    let val: ReturnType<typeof parseTLSRPT>;
    try {
        val = parseTLSRPT(txtValue);
    } catch {
        issues.push({ id: "tlsrpt.parse-error", severity: "error", field: "txt" });
        return issues;
    }

    if (!val.v) {
        issues.push({
            id: "tlsrpt.missing-version",
            severity: "error",
            field: "v",
            docUrl: RFC + "#section-3",
        });
    } else if (val.v !== "TLSRPTv1") {
        issues.push({
            id: "tlsrpt.invalid-version",
            severity: "error",
            params: { version: val.v },
            field: "v",
            docUrl: RFC + "#section-3",
        });
    }

    const rua = val.rua ?? [];
    if (rua.length === 0) {
        issues.push({
            id: "tlsrpt.missing-rua",
            severity: "error",
            field: "rua",
            docUrl: RFC + "#section-3",
        });
    }

    for (const uri of rua) {
        const u = uri.trim();
        if (!u) {
            issues.push({
                id: "tlsrpt.empty-rua",
                severity: "warning",
                field: "rua",
            });
            continue;
        }
        if (!isMailto(u) && !isHttp(u)) {
            issues.push({
                id: "tlsrpt.invalid-rua-scheme",
                severity: "error",
                params: { uri: u },
                field: "rua",
                docUrl: RFC + "#section-3",
            });
            continue;
        }
        if (isMailto(u)) {
            const local = u.replace(/^mailto:/i, "");
            if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(local)) {
                issues.push({
                    id: "tlsrpt.invalid-mailto",
                    severity: "error",
                    params: { uri: u },
                    field: "rua",
                });
            }
        }
    }

    return issues;
}

registerValidators("svcs.TLS_RPT", { sync: tlsrptSync });

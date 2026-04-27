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
import { parseMTASTS, type MTASTSValue } from "$lib/services/mta_sts";

const RFC = "https://www.rfc-editor.org/rfc/rfc8461";
// RFC 8461 sec. 3.1: id is 1..32 alphanumeric characters.
const ID_RE = /^[A-Za-z0-9]{1,32}$/;

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

registerValidators("svcs.MTA_STS", { sync: mtaStsSync });

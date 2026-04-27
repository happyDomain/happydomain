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

import { describe, it, expect } from "vitest";
import "./compliance";
import { buildContext, getValidators, type ComplianceIssue } from "$lib/services/compliance";
import type { Domain } from "$lib/model/domain";

const ORIGIN = { domain: "example.com." } as unknown as Domain;
const CTX = buildContext("_mta-sts", ORIGIN, null);

function run(txt: string, name = "_mta-sts.example.com."): ComplianceIssue[] {
    const v = getValidators("svcs.MTA_STS");
    return v!.sync!({ txt: { Hdr: { Name: name }, Txt: txt } }, CTX);
}
const ids = (issues: ComplianceIssue[]) => issues.map((i) => i.id);

describe("MTA-STS compliance", () => {
    it("accepts a clean record", () => {
        expect(ids(run("v=STSv1;id=20240101"))).toEqual([]);
    });
    it("flags a wrong owner name", () => {
        expect(ids(run("v=STSv1;id=2024", "example.com."))).toContain("mta_sts.wrong-owner-name");
    });
    it("flags missing version", () => {
        expect(ids(run("id=2024"))).toContain("mta_sts.missing-version");
    });
    it("flags non-STSv1 version", () => {
        expect(ids(run("v=STSv2;id=2024"))).toContain("mta_sts.invalid-version");
    });
    it("flags missing id", () => {
        expect(ids(run("v=STSv1"))).toContain("mta_sts.missing-id");
    });
    it("flags an id with non-alphanumeric chars", () => {
        expect(ids(run("v=STSv1;id=2024-01-01"))).toContain("mta_sts.invalid-id");
    });
    it("flags an id longer than 32 chars", () => {
        expect(ids(run("v=STSv1;id=" + "a".repeat(33)))).toContain("mta_sts.invalid-id");
    });
    it("returns no issue on empty TXT", () => {
        expect(run("")).toEqual([]);
    });
});

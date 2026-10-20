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
import { buildContext, type ComplianceIssue } from "$lib/services/compliance";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";
import { dmarcIssues, selectorIssues } from "./bimi";

function ids(issues: ComplianceIssue[]): string[] {
    return issues.map((i) => i.id);
}

function contextWithDMARC(...policies: string[]) {
    const services = policies.map((p) =>
        makeService("svcs.DMARC", { txt: { Txt: "v=DMARC1; p=" + p, Hdr: { Name: "_dmarc" } } }),
    );
    return buildContext("", makeDomain(), makeZone({ services: { "": services } }));
}

describe("selectorIssues", () => {
    it("accepts a plain selector", () => {
        expect(selectorIssues("default._bimi")).toEqual([]);
        expect(selectorIssues("brand-2024._bimi")).toEqual([]);
    });
    it("flags a missing selector", () => {
        expect(ids(selectorIssues("_bimi"))).toEqual(["bimi.missing-selector"]);
    });
    it("flags a selector with a character outside a plain label", () => {
        expect(ids(selectorIssues("my selector._bimi"))).toEqual(["bimi.invalid-selector"]);
    });
    it("flags an empty selector", () => {
        expect(ids(selectorIssues("._bimi"))).toEqual(["bimi.missing-selector"]);
    });
    it("flags an invalid selector", () => {
        expect(ids(selectorIssues("a.b._bimi"))).toEqual(["bimi.invalid-selector"]);
    });
    it("flags a record outside _bimi", () => {
        expect(ids(selectorIssues("default"))).toEqual([
            "bimi.wrong-owner-name",
            "bimi.missing-selector",
        ]);
    });
});

describe("dmarcIssues", () => {
    it("says nothing without the zone", () => {
        expect(dmarcIssues(buildContext("", makeDomain(), null))).toEqual([]);
    });
    it("flags a zone without DMARC", () => {
        expect(ids(dmarcIssues(contextWithDMARC()))).toEqual(["bimi.no-dmarc"]);
    });
    it("flags a DMARC policy that does not enforce", () => {
        expect(ids(dmarcIssues(contextWithDMARC("none")))).toEqual(["bimi.weak-dmarc-policy"]);
    });
    it("says nothing about a DMARC record without content", () => {
        const empty = makeService("svcs.DMARC", { txt: { Txt: "", Hdr: { Name: "_dmarc" } } });
        const ctx = buildContext("", makeDomain(), makeZone({ services: { "": [empty] } }));
        expect(dmarcIssues(ctx)).toEqual([]);
    });
    it("accepts an enforcing DMARC policy", () => {
        expect(dmarcIssues(contextWithDMARC("quarantine"))).toEqual([]);
        expect(dmarcIssues(contextWithDMARC("none", "reject"))).toEqual([]);
    });
});

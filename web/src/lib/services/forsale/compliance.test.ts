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
const CTX = buildContext("", ORIGIN, null);

function rr(txt: string, ttl = 3600, name = "_for-sale") {
    return { Hdr: { Name: name, Ttl: ttl }, Txt: txt };
}

function run(records: ReturnType<typeof rr>[]): ComplianceIssue[] {
    return getValidators("svcs.ForSale")!.sync!({ txt: records }, CTX);
}

const ids = (issues: ComplianceIssue[]) => issues.map((i) => i.id);

describe("For Sale compliance", () => {
    it("accepts the RFC 10023 example RRset", () => {
        expect(
            ids(
                run([
                    rr("v=FORSALE1;fcod=EXCO-S2lscm95IHdhcyBoZXJl"),
                    rr("v=FORSALE1;ftxt=Call for info."),
                    rr("v=FORSALE1;furi=https://example.com/fs"),
                    rr("v=FORSALE1;fval=USD750"),
                ]),
            ),
        ).toEqual([]);
    });

    it("accepts mailto and tel contact URIs", () => {
        expect(ids(run([rr("v=FORSALE1;furi=mailto:sales@example.com")]))).toEqual([]);
        expect(ids(run([rr("v=FORSALE1;furi=tel:+15550123")]))).toEqual([]);
    });

    it("flags a record published outside the _for-sale node", () => {
        expect(ids(run([rr("v=FORSALE1;fval=USD750", 3600, "@")]))).toContain(
            "forsale.wrong-owner-name",
        );
    });

    it("flags a missing version tag", () => {
        expect(ids(run([rr("fval=USD750")]))).toContain("forsale.missing-version");
    });

    it("flags content that is not a tag-value pair", () => {
        expect(ids(run([rr("v=FORSALE1;garbage")]))).toContain("forsale.malformed-content");
    });

    it("flags more than one pair in a single record", () => {
        expect(ids(run([rr("v=FORSALE1;fval=USD750;ftxt=Call me")]))).toContain(
            "forsale.multiple-pairs",
        );
    });

    it("flags a duplicated tag-value pair", () => {
        expect(ids(run([rr("v=FORSALE1;ftxt=Call me"), rr("v=FORSALE1;ftxt=Call me")]))).toContain(
            "forsale.duplicate-pair",
        );
    });

    it("accepts the same tag twice with different values", () => {
        expect(ids(run([rr("v=FORSALE1;fval=USD750"), rr("v=FORSALE1;fval=EUR700")]))).toEqual([]);
    });

    it("flags a value longer than 239 octets", () => {
        expect(ids(run([rr("v=FORSALE1;ftxt=" + "a".repeat(240))]))).toContain(
            "forsale.value-too-long",
        );
        // Counted in octets, not code points.
        expect(ids(run([rr("v=FORSALE1;ftxt=" + "é".repeat(120))]))).toContain(
            "forsale.value-too-long",
        );
        expect(ids(run([rr("v=FORSALE1;ftxt=" + "a".repeat(239))]))).toEqual([]);
    });

    it("flags a malformed price", () => {
        for (const value of ["750", "usd750", "USD", "USD1.2.3"]) {
            expect(ids(run([rr("v=FORSALE1;fval=" + value)]))).toContain("forsale.invalid-price");
        }
    });

    it("flags an unparseable URI", () => {
        expect(ids(run([rr("v=FORSALE1;furi=example.com/fs")]))).toContain("forsale.invalid-uri");
    });

    it("warns on an unusual URI scheme", () => {
        expect(ids(run([rr("v=FORSALE1;furi=ftp://example.com/fs")]))).toContain(
            "forsale.unusual-uri-scheme",
        );
    });

    it("warns on an unknown tag but keeps it", () => {
        expect(ids(run([rr("v=FORSALE1;fxyz=future")]))).toEqual(["forsale.unknown-tag"]);
    });

    it("flags an empty value", () => {
        expect(ids(run([rr("v=FORSALE1;ftxt=")]))).toContain("forsale.empty-value");
    });

    it("warns on a TTL above one hour", () => {
        expect(ids(run([rr("v=FORSALE1;fval=USD750", 86400)]))).toContain("forsale.ttl-too-high");
    });

    it("warns when the RRset mixes TTLs", () => {
        expect(
            ids(run([rr("v=FORSALE1;fval=USD750", 3600), rr("v=FORSALE1;ftxt=Hi", 600)])),
        ).toContain("forsale.inconsistent-ttl");
    });

    it("reports a version-only RRset as informational", () => {
        expect(ids(run([rr("v=FORSALE1;")]))).toEqual(["forsale.no-content"]);
    });

    it("returns no issue on an empty RRset", () => {
        expect(run([])).toEqual([]);
    });
});

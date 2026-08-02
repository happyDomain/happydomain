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
import {
    byteLength,
    FORSALE_LABEL,
    FORSALE_VERSION,
    ForSaleService,
    type ForSaleValue,
    isForSaleTag,
    isValidAmount,
    isValidCurrency,
    newForSaleRecord,
    parseForSale,
    parsePrice,
    stringifyForSale,
    stringifyPrice,
} from "./model.svelte";
import type { dnsTypeTXT } from "$lib/dns_rr";

function txtRecord(txt: string): dnsTypeTXT {
    return {
        Hdr: { Name: FORSALE_LABEL, Rrtype: 16, Class: 1, Ttl: 3600, Rdlength: 0 },
        Txt: txt,
    };
}

describe("parseForSale", () => {
    it("extracts the tag and the value", () => {
        expect(parseForSale("v=FORSALE1;fval=USD750")).toEqual({
            tag: "fval",
            value: "USD750",
            invalidVersion: false,
            malformed: false,
        });
    });

    it("tolerates a single space after the version tag", () => {
        expect(parseForSale("v=FORSALE1; ftxt=Call for info.")).toEqual({
            tag: "ftxt",
            value: "Call for info.",
            invalidVersion: false,
            malformed: false,
        });
    });

    it("keeps every = of the value", () => {
        expect(parseForSale("v=FORSALE1;furi=https://example.com/fs?a=b").value).toBe(
            "https://example.com/fs?a=b",
        );
    });

    it("reports a version-only record", () => {
        expect(parseForSale("v=FORSALE1;")).toEqual({
            tag: null,
            value: "",
            invalidVersion: false,
            malformed: false,
        });
    });

    it("reports a missing version tag", () => {
        expect(parseForSale("ftxt=no version").invalidVersion).toBe(true);
        expect(parseForSale("v=FORSALE2;ftxt=nope").invalidVersion).toBe(true);
    });

    it("reports content that is not a tag-value pair", () => {
        expect(parseForSale("v=FORSALE1;garbage").malformed).toBe(true);
    });
});

describe("stringifyForSale", () => {
    it("round-trips every tag", () => {
        for (const [tag, value] of [
            ["fcod", "EXCO-1"],
            ["ftxt", "Call for info."],
            ["furi", "https://example.com/fs"],
            ["fval", "USD750"],
        ] as const) {
            const txt = stringifyForSale(tag, value);
            expect(txt.startsWith(FORSALE_VERSION)).toBe(true);
            expect(parseForSale(txt)).toEqual({
                tag,
                value,
                invalidVersion: false,
                malformed: false,
            });
        }
    });

    it("emits a bare version record when there is no tag", () => {
        expect(stringifyForSale(null, "")).toBe(FORSALE_VERSION);
    });
});

describe("prices", () => {
    it("splits the currency from the amount", () => {
        expect(parsePrice("USD750")).toEqual({ currency: "USD", amount: "750" });
        expect(parsePrice("EUR1234.56")).toEqual({ currency: "EUR", amount: "1234.56" });
        expect(parsePrice("750")).toEqual({ currency: "", amount: "750" });
    });

    it("round-trips", () => {
        const p = parsePrice("EUR1234.56");
        expect(stringifyPrice(p.currency, p.amount)).toBe("EUR1234.56");
    });

    it("validates the RFC shapes", () => {
        expect(isValidCurrency("USD")).toBe(true);
        expect(isValidCurrency("usd")).toBe(false);
        expect(isValidCurrency("")).toBe(false);
        expect(isValidAmount("750")).toBe(true);
        expect(isValidAmount("750.50")).toBe(true);
        expect(isValidAmount("750.")).toBe(false);
        expect(isValidAmount(".5")).toBe(false);
        expect(isValidAmount("1.2.3")).toBe(false);
    });
});

describe("byteLength", () => {
    it("counts UTF-8 octets, not code points", () => {
        expect(byteLength("abc")).toBe(3);
        expect(byteLength("é")).toBe(2);
        expect(byteLength("🏠")).toBe(4);
    });
});

describe("isForSaleTag", () => {
    it("recognizes the four RFC 10023 tags", () => {
        expect(isForSaleTag("fcod")).toBe(true);
        expect(isForSaleTag("fval")).toBe(true);
        expect(isForSaleTag("fxyz")).toBe(false);
        expect(isForSaleTag(null)).toBe(false);
    });
});

describe("newForSaleRecord", () => {
    it("uses the _for-sale label as the relative owner name", () => {
        const rr = newForSaleRecord("ftxt", "Buy me");
        expect(rr.Hdr.Name).toBe(FORSALE_LABEL);
        expect(rr.Txt).toBe("v=FORSALE1;ftxt=Buy me");
    });
});

describe("ForSaleService", () => {
    const value = (): ForSaleValue => ({
        txt: [
            txtRecord("v=FORSALE1;fval=USD750"),
            txtRecord("v=FORSALE1;ftxt=Call for info."),
            txtRecord("v=FORSALE1;ftxt=Appelez-nous"),
            txtRecord("v=FORSALE1;fxyz=future"),
        ],
    });

    it("exposes every entry in RRset order, unknown tags included", () => {
        const svc = new ForSaleService(value());

        expect(svc.entries.map((e) => e.pair.tag)).toEqual(["fval", "ftxt", "ftxt", "fxyz"]);
        expect(svc.entries.map((e) => e.index)).toEqual([0, 1, 2, 3]);
    });

    it("hides the bare version record from the editable entries", () => {
        const svc = new ForSaleService({
            txt: [txtRecord("v=FORSALE1;"), txtRecord("v=FORSALE1;fval=USD750")],
        });

        expect(svc.entries).toHaveLength(2);
        expect(svc.editableEntries.map((e) => e.index)).toEqual([1]);
    });

    it("keeps a broken record editable so it can be repaired", () => {
        const svc = new ForSaleService({
            txt: [txtRecord("v=FORSALE1;garbage")],
        });

        expect(svc.editableEntries).toHaveLength(1);

        svc.setRaw(0, "ftxt=fixed");

        expect(svc.records[0].Txt).toBe("v=FORSALE1;ftxt=fixed");
        expect(svc.editableEntries[0].pair.tag).toBe("ftxt");
    });

    it("accepts a single record instead of an array", () => {
        const svc = new ForSaleService({ txt: txtRecord("v=FORSALE1;") });

        expect(svc.records).toHaveLength(1);
        expect(svc.editableEntries).toHaveLength(0);
    });

    it("replaces the bare version record when the first pair is added", () => {
        const svc = new ForSaleService({
            txt: [txtRecord("v=FORSALE1;")],
        });

        svc.add("fval", "USD750");

        expect(svc.records).toHaveLength(1);
        expect(svc.records[0].Txt).toBe("v=FORSALE1;fval=USD750");
    });

    it("falls back to a bare version record when the last pair is removed", () => {
        const svc = new ForSaleService({
            txt: [txtRecord("v=FORSALE1;fval=USD750")],
        });

        svc.remove(0);

        expect(svc.records).toHaveLength(1);
        expect(svc.records[0].Txt).toBe(FORSALE_VERSION);
    });

    it("rewrites the value while keeping the tag", () => {
        const svc = new ForSaleService(value());

        svc.setValue(1, "New message");

        expect(svc.records[1].Txt).toBe("v=FORSALE1;ftxt=New message");
        expect(svc.getValue(1)).toBe("New message");
    });
});

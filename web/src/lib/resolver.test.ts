// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { describe, it, expect, vi, afterEach } from "vitest";

import { resolvers, recordsFields } from "./resolver";

describe("resolvers", () => {
    it("has an Unfiltered and a Filtered category", () => {
        expect(Object.keys(resolvers)).toEqual(["Unfiltered", "Filtered"]);
    });

    it("each category is a non-empty array", () => {
        expect(resolvers.Unfiltered.length).toBeGreaterThan(0);
        expect(resolvers.Filtered.length).toBeGreaterThan(0);
    });

    it("every entry has a value and a text", () => {
        for (const category of Object.values(resolvers)) {
            for (const entry of category) {
                expect(typeof entry.value).toBe("string");
                expect(entry.value.length).toBeGreaterThan(0);
                expect(typeof entry.text).toBe("string");
                expect(entry.text.length).toBeGreaterThan(0);
            }
        }
    });

    it("includes the local resolver in Unfiltered", () => {
        expect(resolvers.Unfiltered).toContainEqual({ value: "local", text: "Local resolver" });
    });

    it("includes well-known public resolvers", () => {
        expect(resolvers.Unfiltered).toContainEqual({
            value: "1.1.1.1",
            text: "Cloudflare DNS resolver",
        });
        expect(resolvers.Unfiltered).toContainEqual({
            value: "8.8.8.8",
            text: "Google Public DNS resolver",
        });
    });

    it("includes filtered resolvers", () => {
        expect(resolvers.Filtered).toContainEqual({
            value: "9.9.9.9",
            text: "Quad9 DNS resolver",
        });
    });
});

describe("recordsFields", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("returns the fields for A records (1)", () => {
        expect(recordsFields(1)).toEqual(["A"]);
    });

    it("returns the fields for NS records (2)", () => {
        expect(recordsFields(2)).toEqual(["Ns"]);
    });

    it("returns the fields for CNAME records (5)", () => {
        expect(recordsFields(5)).toEqual(["Target"]);
    });

    it("returns the fields for SOA records (6)", () => {
        expect(recordsFields(6)).toEqual([
            "Ns",
            "Mbox",
            "Serial",
            "Refresh",
            "Retry",
            "Expire",
            "Minttl",
        ]);
    });

    it("returns the fields for PTR records (12)", () => {
        expect(recordsFields(12)).toEqual(["Ptr"]);
    });

    it("returns the fields for HINFO records (13)", () => {
        expect(recordsFields(13)).toEqual(["Cpu", "Os"]);
    });

    it("returns the fields for MX records (15)", () => {
        expect(recordsFields(15)).toEqual(["Mx", "Preference"]);
    });

    it("returns Txt for TXT records (16)", () => {
        expect(recordsFields(16)).toEqual(["Txt"]);
    });

    it("returns Txt for SPF records (99)", () => {
        expect(recordsFields(99)).toEqual(["Txt"]);
    });

    it("returns the fields for AAAA records (28)", () => {
        expect(recordsFields(28)).toEqual(["AAAA"]);
    });

    it("returns the fields for SRV records (33)", () => {
        expect(recordsFields(33)).toEqual(["Target", "Port", "Priority", "Weight"]);
    });

    it("returns the fields for DS records (43)", () => {
        expect(recordsFields(43)).toEqual(["KeyTag", "Algorithm", "DigestType", "Digest"]);
    });

    it("returns the fields for SSHFP records (44)", () => {
        expect(recordsFields(44)).toEqual(["Algorithm", "Type", "FingerPrint"]);
    });

    it("returns the fields for RRSIG records (46)", () => {
        expect(recordsFields(46)).toEqual([
            "TypeCovered",
            "Algorithm",
            "Labels",
            "OrigTtl",
            "Expiration",
            "Inception",
            "KeyTag",
            "SignerName",
            "Signature",
        ]);
    });

    it("returns the fields for TLSA records (52)", () => {
        expect(recordsFields(52)).toEqual(["Usage", "Selector", "MatchingType", "Certificate"]);
    });

    it("returns an empty array and warns for an unknown rrtype", () => {
        const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

        expect(recordsFields(9999)).toEqual([]);
        expect(warnSpy).toHaveBeenCalledWith("Unknown RRtype asked fields: ", 9999);

        warnSpy.mockRestore();
    });

    it("returns an empty array for a negative rrtype", () => {
        const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

        expect(recordsFields(-1)).toEqual([]);

        warnSpy.mockRestore();
    });
});

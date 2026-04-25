import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { resolvers, recordsFields } from "./resolver";

describe("resolvers data", () => {
    it("exposes Unfiltered and Filtered groups", () => {
        expect(Object.keys(resolvers).sort()).toEqual(["Filtered", "Unfiltered"]);
    });

    it("each Unfiltered entry has a non-empty value and text", () => {
        for (const r of resolvers.Unfiltered) {
            expect(typeof r.value).toBe("string");
            expect(r.value.length).toBeGreaterThan(0);
            expect(typeof r.text).toBe("string");
            expect(r.text.length).toBeGreaterThan(0);
        }
    });

    it("each Filtered entry has a non-empty value and text", () => {
        for (const r of resolvers.Filtered) {
            expect(typeof r.value).toBe("string");
            expect(r.value.length).toBeGreaterThan(0);
            expect(typeof r.text).toBe("string");
            expect(r.text.length).toBeGreaterThan(0);
        }
    });

    it("includes the local resolver as the first Unfiltered entry", () => {
        expect(resolvers.Unfiltered[0]).toEqual({
            value: "local",
            text: "Local resolver",
        });
    });

    it("includes well-known public resolvers", () => {
        const unfiltered = resolvers.Unfiltered.map((r) => r.value);
        expect(unfiltered).toContain("1.1.1.1"); // Cloudflare
        expect(unfiltered).toContain("8.8.8.8"); // Google
        expect(unfiltered).toContain("9.9.9.10"); // Quad9 unsec
        const filtered = resolvers.Filtered.map((r) => r.value);
        expect(filtered).toContain("9.9.9.9"); // Quad9 default
    });
});

describe("recordsFields", () => {
    it.each([
        [1, ["A"]],
        [2, ["Ns"]],
        [5, ["Target"]],
        [6, ["Ns", "Mbox", "Serial", "Refresh", "Retry", "Expire", "Minttl"]],
        [12, ["Ptr"]],
        [13, ["Cpu", "Os"]],
        [15, ["Mx", "Preference"]],
        [16, ["Txt"]],
        [28, ["AAAA"]],
        [33, ["Target", "Port", "Priority", "Weight"]],
        [43, ["KeyTag", "Algorithm", "DigestType", "Digest"]],
        [44, ["Algorithm", "Type", "FingerPrint"]],
        [52, ["Usage", "Selector", "MatchingType", "Certificate"]],
        [99, ["Txt"]], // SPF aliases TXT
    ])("returns the expected fields for RR type %i", (rrtype, expected) => {
        expect(recordsFields(rrtype)).toEqual(expected);
    });

    it("returns the full RRSIG field set", () => {
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

    describe("for unknown RR types", () => {
        let warnSpy: ReturnType<typeof vi.spyOn>;
        beforeEach(() => {
            warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
        });
        afterEach(() => {
            warnSpy.mockRestore();
        });

        it("returns an empty array and warns", () => {
            expect(recordsFields(9999)).toEqual([]);
            expect(warnSpy).toHaveBeenCalledTimes(1);
        });

        it("does not throw on negative or zero types", () => {
            expect(recordsFields(0)).toEqual([]);
            expect(recordsFields(-1)).toEqual([]);
        });
    });
});

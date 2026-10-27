import { describe, expect, it } from "vitest";

import { rdatanamefields } from "./dns_rr";

// rdatanamefields is generated from what the backend makes relative to the
// zone: a field listed here is shown and typed relative to it.
describe("rdatanamefields", () => {
    it("lists the names of the common records", () => {
        expect(rdatanamefields("NS")).toEqual(["Ns"]);
        expect(rdatanamefields("CNAME")).toEqual(["Target"]);
        expect(rdatanamefields("MX")).toEqual(["Mx"]);
        expect(rdatanamefields("PTR")).toEqual(["Ptr"]);
        expect(rdatanamefields("SRV")).toEqual(["Target"]);
        expect(rdatanamefields("NAPTR")).toEqual(["Replacement"]);
        expect(rdatanamefields("DNAME")).toEqual(["Target"]);
        expect(rdatanamefields("SOA")).toEqual(["Ns", "Mbox"]);
    });

    it("answers the same for the numeric type", () => {
        expect(rdatanamefields(15)).toEqual(rdatanamefields("MX"));
        expect(rdatanamefields(33)).toEqual(rdatanamefields("SRV"));
    });

    it("lists the target of the service bindings", () => {
        expect(rdatanamefields("SVCB")).toEqual(["Target"]);
        expect(rdatanamefields("HTTPS")).toEqual(["Target"]);
    });

    it("leaves out the algorithm names", () => {
        expect(rdatanamefields("TSIG")).toEqual([]);
        expect(rdatanamefields("TKEY")).toEqual([]);
    });

    it("has no name in records holding none", () => {
        expect(rdatanamefields("A")).toEqual([]);
        expect(rdatanamefields("AAAA")).toEqual([]);
        expect(rdatanamefields("TXT")).toEqual([]);
        expect(rdatanamefields("CAA")).toEqual([]);
        expect(rdatanamefields("TLSA")).toEqual([]);
    });
});

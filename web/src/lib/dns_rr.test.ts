import { describe, it, expect } from "vitest";
import {
    getRrtype,
    newRR,
    nsrrtype,
    rdatatostr,
    rdatafields,
    type dnsRR,
    type dnsTypeA,
    type dnsTypeAAAA,
    type dnsTypeSOA,
    type dnsTypeMX,
    type dnsTypeTXT,
    type dnsTypeSRV,
    type dnsTypeCAA,
    type dnsTypeDS,
    type dnsTypeSSHFP,
    type dnsTypeTLSA,
    type dnsTypeNAPTR,
    type dnsTypeURI,
} from "./dns_rr";

// A representative subset of well-supported RR types that exist in all four
// functions (getRrtype/nsrrtype/newRR/rdatafields). Used to verify cross-function
// consistency: drift in any one switch statement breaks one of these invariants.
const KNOWN_TYPES: ReadonlyArray<{ name: string; lower: string; num: number }> = [
    { name: "A", lower: "a", num: 1 },
    { name: "NS", lower: "ns", num: 2 },
    { name: "CNAME", lower: "cname", num: 5 },
    { name: "SOA", lower: "soa", num: 6 },
    { name: "PTR", lower: "ptr", num: 12 },
    { name: "HINFO", lower: "hinfo", num: 13 },
    { name: "MX", lower: "mx", num: 15 },
    { name: "TXT", lower: "txt", num: 16 },
    { name: "AAAA", lower: "aaaa", num: 28 },
    { name: "LOC", lower: "loc", num: 29 },
    { name: "SRV", lower: "srv", num: 33 },
    { name: "NAPTR", lower: "naptr", num: 35 },
    { name: "DNAME", lower: "dname", num: 39 },
    { name: "DS", lower: "ds", num: 43 },
    { name: "SSHFP", lower: "sshfp", num: 44 },
    { name: "RRSIG", lower: "rrsig", num: 46 },
    { name: "DNSKEY", lower: "dnskey", num: 48 },
    { name: "TLSA", lower: "tlsa", num: 52 },
    { name: "SPF", lower: "spf", num: 99 },
    { name: "URI", lower: "uri", num: 256 },
    { name: "CAA", lower: "caa", num: 257 },
];

describe("getRrtype", () => {
    it.each(KNOWN_TYPES)("maps $name (uppercase) to $num", ({ name, num }) => {
        expect(getRrtype(name)).toBe(num);
    });

    it.each(KNOWN_TYPES)("maps $lower (lowercase) to $num", ({ lower, num }) => {
        expect(getRrtype(lower)).toBe(num);
    });

    it("supports the NSAP-PTR hyphenated name", () => {
        expect(getRrtype("NSAP-PTR")).toBe(23);
        expect(getRrtype("nsap-ptr")).toBe(23);
    });

    it("throws for an unknown type name", () => {
        expect(() => getRrtype("NOTAREALTYPE")).toThrow();
    });

    it("does not accept random-cased variants (only upper or all-lower)", () => {
        expect(() => getRrtype("Aaaa")).toThrow();
    });
});

describe("nsrrtype", () => {
    it.each(KNOWN_TYPES)("maps numeric $num to $name", ({ num, name }) => {
        expect(nsrrtype(num)).toBe(name);
    });

    it.each(KNOWN_TYPES)("maps stringified $num to $name", ({ num, name }) => {
        expect(nsrrtype(String(num))).toBe(name);
    });

    it("does not throw for unknown numeric types", () => {
        expect(() => nsrrtype(99999)).not.toThrow();
    });
});

describe("getRrtype <-> nsrrtype round trip", () => {
    it.each(KNOWN_TYPES)("getRrtype(nsrrtype($num)) === $num", ({ num }) => {
        expect(getRrtype(nsrrtype(num))).toBe(num);
    });
});

describe("newRR", () => {
    it("populates the standard header with TTL 3600 and IN class", () => {
        const rr = newRR("example.com.", 1);
        expect(rr.Hdr).toMatchObject({
            Name: "example.com.",
            Rrtype: 1,
            Class: 1,
            Ttl: 3600,
        });
    });

    it("zero-initializes A.A as empty string", () => {
        const rr = newRR("example.com.", 1) as dnsTypeA;
        expect(rr.A).toBe("");
    });

    it("zero-initializes SOA numeric fields as 0", () => {
        const rr = newRR("example.com.", 6) as dnsTypeSOA;
        expect(rr).toMatchObject({
            Ns: "",
            Mbox: "",
            Serial: 0,
            Refresh: 0,
            Retry: 0,
            Expire: 0,
            Minttl: 0,
        });
    });

    it("zero-initializes MX with Preference 0 and empty Mx", () => {
        const rr = newRR("example.com.", 15) as dnsTypeMX;
        expect(rr).toMatchObject({ Preference: 0, Mx: "" });
    });

    it("zero-initializes SRV with all four fields", () => {
        const rr = newRR("example.com.", 33) as dnsTypeSRV;
        expect(rr).toMatchObject({ Priority: 0, Weight: 0, Port: 0, Target: "" });
    });

    it("returns a record with only the header for unknown rrtype", () => {
        const rr = newRR("example.com.", 99999);
        expect(Object.keys(rr)).toEqual(["Hdr"]);
        expect(rr.Hdr.Rrtype).toBe(99999);
    });
});

describe("rdatatostr", () => {
    it("renders A as the bare address string", () => {
        const rr = newRR("example.com.", 1) as dnsTypeA;
        rr.A = "192.0.2.1";
        expect(rdatatostr(rr)).toBe("192.0.2.1");
    });

    it("renders AAAA as the bare address string", () => {
        const rr = newRR("example.com.", 28) as dnsTypeAAAA;
        rr.AAAA = "2001:db8::1";
        expect(rdatatostr(rr)).toBe("2001:db8::1");
    });

    it("renders MX as 'preference exchange'", () => {
        const rr = newRR("example.com.", 15) as dnsTypeMX;
        rr.Preference = 10;
        rr.Mx = "mail.example.com.";
        expect(rdatatostr(rr)).toBe("10 mail.example.com.");
    });

    it("renders SOA with all seven fields space-separated", () => {
        const rr = newRR("example.com.", 6) as dnsTypeSOA;
        rr.Ns = "ns1.example.com.";
        rr.Mbox = "hostmaster.example.com.";
        rr.Serial = 2024010101;
        rr.Refresh = 7200;
        rr.Retry = 3600;
        rr.Expire = 1209600;
        rr.Minttl = 3600;
        expect(rdatatostr(rr)).toBe(
            "ns1.example.com. hostmaster.example.com. 2024010101 7200 3600 1209600 3600",
        );
    });

    it("quotes TXT data containing whitespace", () => {
        const rr = newRR("example.com.", 16) as dnsTypeTXT;
        rr.Txt = "v=spf1 include:_spf.google.com ~all";
        expect(rdatatostr(rr)).toBe('"v=spf1 include:_spf.google.com ~all"');
    });

    it("does not quote TXT data without whitespace or special chars", () => {
        const rr = newRR("example.com.", 16) as dnsTypeTXT;
        rr.Txt = "no-spaces";
        expect(rdatatostr(rr)).toBe("no-spaces");
    });

    it("escapes embedded double quotes in TXT data", () => {
        const rr = newRR("example.com.", 16) as dnsTypeTXT;
        rr.Txt = 'has "quotes"';
        expect(rdatatostr(rr)).toBe('"has \\"quotes\\""');
    });

    it("renders SRV as 'priority weight port target'", () => {
        const rr = newRR("_sip._tcp.example.com.", 33) as dnsTypeSRV;
        rr.Priority = 10;
        rr.Weight = 60;
        rr.Port = 5060;
        rr.Target = "sipserver.example.com.";
        expect(rdatatostr(rr)).toBe("10 60 5060 sipserver.example.com.");
    });

    it("renders CAA as 'flag tag value' (value quoted if needed)", () => {
        const rr = newRR("example.com.", 257) as dnsTypeCAA;
        rr.Flag = 0;
        rr.Tag = "issue";
        rr.Value = "letsencrypt.org";
        expect(rdatatostr(rr)).toBe("0 issue letsencrypt.org");
    });

    it("renders DS with 'keytag algorithm digesttype digest'", () => {
        const rr = newRR("example.com.", 43) as dnsTypeDS;
        rr.KeyTag = 12345;
        rr.Algorithm = 8;
        rr.DigestType = 2;
        rr.Digest = "ABCDEF1234567890";
        expect(rdatatostr(rr)).toBe("12345 8 2 ABCDEF1234567890");
    });

    it("renders SSHFP with 'algorithm type fingerprint'", () => {
        const rr = newRR("host.example.com.", 44) as dnsTypeSSHFP;
        rr.Algorithm = 4;
        rr.Type = 2;
        rr.FingerPrint = "deadbeefcafef00d";
        expect(rdatatostr(rr)).toBe("4 2 deadbeefcafef00d");
    });

    it("renders TLSA with 'usage selector matchingtype certificate'", () => {
        const rr = newRR("_443._tcp.example.com.", 52) as dnsTypeTLSA;
        rr.Usage = 3;
        rr.Selector = 1;
        rr.MatchingType = 1;
        rr.Certificate = "abcdef";
        expect(rdatatostr(rr)).toBe("3 1 1 abcdef");
    });

    it("renders NAPTR with quoted string fields", () => {
        const rr = newRR("example.com.", 35) as dnsTypeNAPTR;
        rr.Order = 100;
        rr.Preference = 10;
        rr.Flags = "U";
        rr.Service = "E2U+sip";
        rr.Regexp = "!^.*$!sip:info@example.com!";
        rr.Replacement = ".";
        expect(rdatatostr(rr)).toBe('100 10 "U" "E2U+sip" "!^.*$!sip:info@example.com!" .');
    });

    it("renders URI with 'priority weight target'", () => {
        const rr = newRR("_http._tcp.example.com.", 256) as dnsTypeURI;
        rr.Priority = 10;
        rr.Weight = 1;
        rr.Target = "https://example.com/";
        expect(rdatatostr(rr)).toBe("10 1 https://example.com/");
    });

    it("returns 'unknown #N' for an unknown rrtype", () => {
        const rr = { Hdr: { Name: ".", Rrtype: 99999, Class: 1, Ttl: 0 } } as dnsRR;
        expect(rdatatostr(rr)).toBe("unknown #99999");
    });
});

describe("rdatafields", () => {
    it.each(KNOWN_TYPES)(
        "returns the same field set when called by name or number for $name",
        ({ name, num }) => {
            expect(rdatafields(num)).toEqual(rdatafields(name));
        },
    );

    it("returns the documented field set for SOA", () => {
        expect(rdatafields("SOA")).toEqual([
            "Ns",
            "Mbox",
            "Serial",
            "Refresh",
            "Retry",
            "Expire",
            "Minttl",
        ]);
    });

    it("returns the documented field set for SRV", () => {
        expect(rdatafields(33)).toEqual(["Priority", "Weight", "Port", "Target"]);
    });

    it("returns the documented field set for CAA", () => {
        expect(rdatafields(257)).toEqual(["Flag", "Tag", "Value"]);
    });

    it("returns an empty array for unknown rrtype number", () => {
        expect(rdatafields(99999)).toEqual([]);
    });

    it("returns an empty array for unknown rrtype name", () => {
        expect(rdatafields("NOTAREALTYPE")).toEqual([]);
    });
});

describe("cross-function consistency", () => {
    it("rdatafields keys are present on a freshly created RR (newRR)", () => {
        for (const { num } of KNOWN_TYPES) {
            const rr = newRR("example.com.", num);
            const fields = rdatafields(num);
            for (const f of fields) {
                expect(rr).toHaveProperty(f);
            }
        }
    });

    it("rdatatostr does not throw when given a freshly built RR", () => {
        for (const { num } of KNOWN_TYPES) {
            const rr = newRR("example.com.", num);
            expect(() => rdatatostr(rr)).not.toThrow();
        }
    });
});

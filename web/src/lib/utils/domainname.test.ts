import { describe, expect, it } from "vitest";

import { looksAbsolute, textToDomainName } from "./domainname";

describe("textToDomainName", () => {
    it("stores what has no final dot as relative", () => {
        expect(textToDomainName("ns1")).toBe("ns1");
        expect(textToDomainName(" www.blog ")).toBe("www.blog");
        expect(textToDomainName("ns1.ovh.net")).toBe("ns1.ovh.net");
    });

    it("keeps the origin typed in full relative without the final dot", () => {
        expect(textToDomainName("ns1.example.com")).toBe("ns1.example.com");
        expect(textToDomainName("Example.com")).toBe("Example.com");
    });

    it("keeps the final dot of absolute names", () => {
        expect(textToDomainName("ns1.ovh.net.")).toBe("ns1.ovh.net.");
        expect(textToDomainName("ns1.example.com.")).toBe("ns1.example.com.");
    });

    it("keeps empty and apex as is", () => {
        expect(textToDomainName("")).toBe("");
        expect(textToDomainName("@")).toBe("@");
    });

});

describe("looksAbsolute", () => {
    it("flags names ending with a common or a country TLD", () => {
        expect(looksAbsolute("ns1.ovh.net")).toBe(true);
        expect(looksAbsolute("mail.provider.fr")).toBe(true);
        expect(looksAbsolute("ghs.googlehosted.com")).toBe(true);
    });

    it("leaves ordinary subdomains alone", () => {
        expect(looksAbsolute("ns1")).toBe(false);
        expect(looksAbsolute("www.blog")).toBe(false);
        expect(looksAbsolute("_sip._tcp")).toBe(false);
        expect(looksAbsolute("@")).toBe(false);
        expect(looksAbsolute("ns1.ovh.net.")).toBe(false);
    });
});

describe("edge cases", () => {
    it("keeps the root as is", () => {
        expect(textToDomainName(".")).toBe(".");
    });

    it("keeps a wildcard relative", () => {
        expect(textToDomainName("*.dev")).toBe("*.dev");
    });

    it("flags TLDs whatever their case", () => {
        expect(looksAbsolute("ns1.OVH.NET")).toBe(true);
        expect(looksAbsolute("mx.provider.De")).toBe(true);
    });

    it("leaves alone a last label that is not a TLD", () => {
        expect(looksAbsolute("www.blog")).toBe(false);
        expect(looksAbsolute("a.b1")).toBe(false);
        expect(looksAbsolute("srv.internal")).toBe(false);
        expect(looksAbsolute("x.abc")).toBe(false);
    });
});

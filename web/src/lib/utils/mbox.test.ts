import { describe, expect, it } from "vitest";

import { emailToMbox, mboxToEmail } from "./mbox";

describe("mboxToEmail", () => {
    it("reads a mailbox relative to the zone", () => {
        expect(mboxToEmail("dnsmaster", "example.com.")).toEqual({
            local: "dnsmaster",
            domain: "example.com",
        });
        expect(mboxToEmail("hostmaster.sub", "example.com.")).toEqual({
            local: "hostmaster",
            domain: "sub.example.com",
        });
    });

    it("reads an absolute mailbox", () => {
        expect(mboxToEmail("hostmaster.other.org.", "example.com.")).toEqual({
            local: "hostmaster",
            domain: "other.org",
        });
    });

    it("unescapes the dots of the local part", () => {
        expect(mboxToEmail("john\\.doe", "example.com.")).toEqual({
            local: "john.doe",
            domain: "example.com",
        });
        expect(mboxToEmail("john\\.doe.other.org.", "example.com")).toEqual({
            local: "john.doe",
            domain: "other.org",
        });
    });

    it("has nothing to say about an empty mailbox", () => {
        expect(mboxToEmail("", "example.com.")).toBeNull();
        expect(mboxToEmail("@", "example.com.")).toBeNull();
    });
});

describe("emailToMbox", () => {
    it("takes a bare local part to be in the zone", () => {
        expect(emailToMbox("dnsmaster", "example.com.")).toBe("dnsmaster");
        expect(emailToMbox(" dnsmaster ", "example.com.")).toBe("dnsmaster");
    });

    it("keeps the domain typed with the address, absolute", () => {
        expect(emailToMbox("dnsmaster@example.com", "example.com.")).toBe("dnsmaster.example.com.");
        expect(emailToMbox("dnsmaster@Example.COM.", "example.com.")).toBe("dnsmaster.Example.COM.");
        expect(emailToMbox("hostmaster@sub.example.com", "example.com.")).toBe(
            "hostmaster.sub.example.com.",
        );
    });

    it("makes an address of another domain absolute", () => {
        expect(emailToMbox("hostmaster@other.org", "example.com.")).toBe("hostmaster.other.org.");
        expect(emailToMbox("hostmaster@notexample.com", "example.com.")).toBe(
            "hostmaster.notexample.com.",
        );
    });

    it("escapes the dots of the local part", () => {
        expect(emailToMbox("john.doe", "example.com.")).toBe("john\\.doe");
        expect(emailToMbox("john.doe@other.org", "example.com.")).toBe("john\\.doe.other.org.");
    });

    it("rejects what is not an address", () => {
        expect(emailToMbox("", "example.com.")).toBeNull();
        expect(emailToMbox("@example.com", "example.com.")).toBeNull();
        expect(emailToMbox("john@", "example.com.")).toBeNull();
        expect(emailToMbox("john doe", "example.com.")).toBeNull();
        expect(emailToMbox("a@b@c.org", "example.com.")).toBeNull();
        expect(emailToMbox("john@other..org", "example.com.")).toBeNull();
    });

    it("round-trips through mboxToEmail", () => {
        for (const email of ["dnsmaster@example.com", "john.doe@other.org", "a@sub.example.com"]) {
            const e = mboxToEmail(emailToMbox(email, "example.com.")!, "example.com.")!;
            expect(e.local + "@" + e.domain).toBe(email);
        }
    });
});

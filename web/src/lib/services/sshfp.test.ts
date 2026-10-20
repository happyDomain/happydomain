import { describe, it, expect } from "vitest";

import type { dnsTypeSSHFP } from "$lib/dns_rr";
import { KNOWN_STATUSES, defaultHost, importKeys } from "./sshfp";
import en from "./svcs.SSHFPs/locales/en.json";
import fr from "./svcs.SSHFPs/locales/fr.json";

describe("defaultHost", () => {
    it("joins the subdomain and the origin, without the trailing dot", () => {
        expect(defaultHost("www", "example.com.")).toBe("www.example.com");
        expect(defaultHost("www", "example.com")).toBe("www.example.com");
    });

    it("is the origin itself at the apex", () => {
        expect(defaultHost("", "example.com.")).toBe("example.com");
        expect(defaultHost("@", "example.com.")).toBe("example.com");
    });
});

// A published record, as the editor holds it.
const sshfp = (fields: Pick<dnsTypeSSHFP, "Algorithm" | "Type" | "FingerPrint">) =>
    ({ Hdr: { Name: "", Rrtype: 44, Class: 1, Ttl: 3600 }, ...fields }) as dnsTypeSSHFP;

describe("importKeys", () => {
    it("adds both fingerprints of a key, lowercased", () => {
        const entries: dnsTypeSSHFP[] = [];
        const added = importKeys(entries, [{ algorithm: 4, sha256: "AB", sha1: "cd" }]);

        expect(added).toBe(2);
        expect(entries).toEqual([
            {
                Hdr: { Name: "", Rrtype: 44, Class: 1, Ttl: 3600 },
                Algorithm: 4,
                Type: 2,
                FingerPrint: "ab",
            },
            {
                Hdr: { Name: "", Rrtype: 44, Class: 1, Ttl: 3600 },
                Algorithm: 4,
                Type: 1,
                FingerPrint: "cd",
            },
        ]);
    });

    it("skips what is already published, whatever its case", () => {
        const entries = [{ Algorithm: 4, Type: 2, FingerPrint: "AB" }].map(sshfp);

        expect(importKeys(entries, [{ algorithm: 4, sha256: "ab", sha1: "cd" }])).toBe(1);
        expect(entries).toHaveLength(2);
    });

    it("adds nothing when everything is already published", () => {
        const entries = [
            { Algorithm: 4, Type: 2, FingerPrint: "ab" },
            { Algorithm: 4, Type: 1, FingerPrint: "cd" },
        ].map(sshfp);

        expect(importKeys(entries, [{ algorithm: 4, sha256: "ab", sha1: "cd" }])).toBe(0);
    });

    it("keeps the same fingerprint under another algorithm", () => {
        const entries = [{ Algorithm: 3, Type: 2, FingerPrint: "ab" }].map(sshfp);

        expect(importKeys(entries, [{ algorithm: 4, sha256: "ab" }])).toBe(1);
    });

    it("skips an empty or missing fingerprint", () => {
        const entries: dnsTypeSSHFP[] = [];

        expect(importKeys(entries, [{ algorithm: 4, sha256: "ab", sha1: "" }])).toBe(1);
        expect(importKeys(entries, [{ algorithm: 1 }])).toBe(0);
    });

    it("skips a key without algorithm instead of publishing algorithm 0", () => {
        const entries: dnsTypeSSHFP[] = [];

        expect(importKeys(entries, [{ sha256: "ab", sha1: "cd" }])).toBe(0);
        expect(entries).toEqual([]);
    });
});

describe("backend statuses", () => {
    it.each(KNOWN_STATUSES)("%s has a sentence in every language", (status) => {
        for (const locale of [en, fr]) {
            expect((locale.sshfp.retrieve.status as Record<string, string>)[status]).toBeTruthy();
        }
    });
});

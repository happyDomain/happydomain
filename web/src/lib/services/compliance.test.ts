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

import { describe, it, expect, vi, beforeEach } from "vitest";

import {
    asArray,
    buildContext,
    checkDigest,
    getValidators,
    hasAddress,
    hasValidators,
    inZoneSubdomain,
    isCnameOwner,
    isUint16,
    isValidDnsName,
    isValidHostname,
    normalizeFqdn,
    originFqdn,
    recordFqdn,
    registerValidators,
    runAsyncValidators,
    runSyncValidators,
    type ComplianceIssue,
} from "./compliance";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";
import type { Zone } from "$lib/model/zone";

const ORIGIN = makeDomain({ domain: "example.com." });

describe("asArray", () => {
    it("returns an empty array for null", () => {
        expect(asArray(null)).toEqual([]);
    });

    it("returns an empty array for undefined", () => {
        expect(asArray(undefined)).toEqual([]);
    });

    it("returns an empty array for an empty string", () => {
        expect(asArray("")).toEqual([]);
    });

    it("returns an empty array for 0", () => {
        expect(asArray(0)).toEqual([]);
    });

    it("wraps a single non-array value into a one-element array", () => {
        expect(asArray({ a: 1 })).toEqual([{ a: 1 }]);
    });

    it("passes arrays through unchanged (same values, not necessarily same reference)", () => {
        const input = [{ a: 1 }, { a: 2 }];
        expect(asArray(input)).toEqual(input);
    });

    it("preserves an empty array as-is", () => {
        expect(asArray([])).toEqual([]);
    });
});

describe("buildContext", () => {
    it("exposes dn, origin and zone unchanged", () => {
        const zone = makeZone();
        const ctx = buildContext("www", ORIGIN, zone);
        expect(ctx.dn).toBe("www");
        expect(ctx.origin).toBe(ORIGIN);
        expect(ctx.zone).toBe(zone);
    });

    describe("findServices", () => {
        it("returns an empty array when zone is null", () => {
            const ctx = buildContext("@", ORIGIN, null);
            expect(ctx.findServices("@")).toEqual([]);
        });

        it("returns an empty array when the subdomain has no services", () => {
            const zone = makeZone({ services: { "@": [makeService("svcs.SPF")] } });
            const ctx = buildContext("@", ORIGIN, zone);
            expect(ctx.findServices("www")).toEqual([]);
        });

        it("returns all services for a subdomain when no type filter is given", () => {
            const spf = makeService("svcs.SPF");
            const mx = makeService("svcs.MXs");
            const zone = makeZone({ services: { "@": [spf, mx] } });
            const ctx = buildContext("@", ORIGIN, zone);
            expect(ctx.findServices("@")).toEqual([spf, mx]);
        });

        it("filters services by type when given", () => {
            const spf = makeService("svcs.SPF");
            const mx = makeService("svcs.MXs");
            const zone = makeZone({ services: { "@": [spf, mx] } });
            const ctx = buildContext("@", ORIGIN, zone);
            expect(ctx.findServices("@", "svcs.SPF")).toEqual([spf]);
        });

        it("returns an empty array when the type filter matches nothing", () => {
            const zone = makeZone({ services: { "@": [makeService("svcs.SPF")] } });
            const ctx = buildContext("@", ORIGIN, zone);
            expect(ctx.findServices("@", "svcs.DKIMRecord")).toEqual([]);
        });

        it("returns a copy, not the live array, when no type filter is given", () => {
            const spf = makeService("svcs.SPF");
            const services = [spf];
            const zone = makeZone({ services: { "@": services } });
            const ctx = buildContext("@", ORIGIN, zone);
            const found = ctx.findServices("@");
            found.push(makeService("svcs.MXs"));
            expect(services).toHaveLength(1);
        });
    });

    describe("findAllServices", () => {
        it("returns an empty array when zone is null", () => {
            const ctx = buildContext("@", ORIGIN, null);
            expect(ctx.findAllServices()).toEqual([]);
        });

        it("returns an empty array when zone.services is undefined", () => {
            const zone = makeZone({ services: undefined as unknown as Zone["services"] });
            const ctx = buildContext("@", ORIGIN, zone);
            expect(ctx.findAllServices()).toEqual([]);
        });

        it("flattens services across every subdomain when no type filter is given", () => {
            const spf = makeService("svcs.SPF");
            const mx = makeService("svcs.MXs");
            const dkim = makeService("svcs.DKIMRecord");
            const zone = makeZone({
                services: {
                    "@": [spf, mx],
                    mail: [dkim],
                },
            });
            const ctx = buildContext("@", ORIGIN, zone);
            expect(ctx.findAllServices()).toEqual([spf, mx, dkim]);
        });

        it("filters across every subdomain by type when given", () => {
            const spf = makeService("svcs.SPF");
            const mx = makeService("svcs.MXs");
            const dkim = makeService("svcs.DKIMRecord");
            const dkim2 = makeService("svcs.DKIMRecord");
            const zone = makeZone({
                services: {
                    "@": [spf, mx, dkim],
                    mail: [dkim2],
                },
            });
            const ctx = buildContext("@", ORIGIN, zone);
            expect(ctx.findAllServices("svcs.DKIMRecord")).toEqual([dkim, dkim2]);
        });

        it("returns an empty array when the type filter matches nothing across the zone", () => {
            const zone = makeZone({ services: { "@": [makeService("svcs.SPF")] } });
            const ctx = buildContext("@", ORIGIN, zone);
            expect(ctx.findAllServices("svcs.DMARC")).toEqual([]);
        });
    });
});

describe("registry: registerValidators / getValidators / hasValidators", () => {
    const TYPE = "svcs.__TestType__";

    it("reports hasValidators(false) for an unregistered type", () => {
        expect(hasValidators("svcs.__NeverRegistered__")).toBe(false);
        expect(getValidators("svcs.__NeverRegistered__")).toBeUndefined();
    });

    it("registers and retrieves sync + async validators for a type", () => {
        const sync = vi.fn(() => []);
        const async = vi.fn(async () => []);
        registerValidators(TYPE, { sync, async });

        expect(hasValidators(TYPE)).toBe(true);
        const v = getValidators(TYPE);
        expect(v?.sync).toBe(sync);
        expect(v?.async).toBe(async);
    });

    it("allows registering only a sync validator", () => {
        const TYPE2 = "svcs.__SyncOnly__";
        const sync = vi.fn(() => []);
        registerValidators(TYPE2, { sync });

        const v = getValidators(TYPE2);
        expect(v?.sync).toBe(sync);
        expect(v?.async).toBeUndefined();
    });

    it("allows registering only an async validator", () => {
        const TYPE3 = "svcs.__AsyncOnly__";
        const async = vi.fn(async () => []);
        registerValidators(TYPE3, { async });

        const v = getValidators(TYPE3);
        expect(v?.async).toBe(async);
        expect(v?.sync).toBeUndefined();
    });

    it("overwrites a previous registration for the same type", () => {
        const TYPE4 = "svcs.__Overwrite__";
        const firstSync = vi.fn(() => []);
        const secondSync = vi.fn(() => []);
        registerValidators(TYPE4, { sync: firstSync });
        registerValidators(TYPE4, { sync: secondSync });

        expect(getValidators(TYPE4)?.sync).toBe(secondSync);
    });
});

describe("runSyncValidators", () => {
    const TYPE = "svcs.__RunSync__";
    const CTX = buildContext("@", ORIGIN, null);

    it("returns an empty array when the type has no registered validators", () => {
        expect(runSyncValidators("svcs.__Unregistered__", {}, CTX)).toEqual([]);
    });

    it("returns an empty array when the type has an async validator but no sync one", () => {
        registerValidators("svcs.__AsyncOnlyRun__", { async: vi.fn(async () => []) });
        expect(runSyncValidators("svcs.__AsyncOnlyRun__", {}, CTX)).toEqual([]);
    });

    it("delegates to the registered sync validator and returns its issues", () => {
        const issues: ComplianceIssue[] = [{ id: "test.issue", severity: "warning" }];
        const sync = vi.fn(() => issues);
        registerValidators(TYPE, { sync });

        const raw = { foo: "bar" };
        const result = runSyncValidators(TYPE, raw, CTX);

        expect(sync).toHaveBeenCalledWith(raw, CTX);
        expect(result).toBe(issues);
    });

    it("catches exceptions thrown by the sync validator and returns an empty array", () => {
        const consoleErr = vi.spyOn(console, "error").mockImplementation(() => {});
        registerValidators(TYPE, {
            sync: () => {
                throw new Error("boom");
            },
        });

        expect(runSyncValidators(TYPE, {}, CTX)).toEqual([]);
        expect(consoleErr).toHaveBeenCalled();
        consoleErr.mockRestore();
    });
});

describe("runAsyncValidators", () => {
    const TYPE = "svcs.__RunAsync__";
    const CTX = buildContext("@", ORIGIN, null);
    let controller: AbortController;

    beforeEach(() => {
        controller = new AbortController();
    });

    it("returns an empty array when the type has no registered validators", async () => {
        await expect(
            runAsyncValidators("svcs.__Unregistered__", {}, CTX, controller.signal),
        ).resolves.toEqual([]);
    });

    it("returns an empty array when the type has a sync validator but no async one", async () => {
        registerValidators("svcs.__SyncOnlyRun__", { sync: vi.fn(() => []) });
        await expect(
            runAsyncValidators("svcs.__SyncOnlyRun__", {}, CTX, controller.signal),
        ).resolves.toEqual([]);
    });

    it("delegates to the registered async validator and returns its issues", async () => {
        const issues: ComplianceIssue[] = [{ id: "test.async-issue", severity: "error" }];
        const asyncFn = vi.fn(async () => issues);
        registerValidators(TYPE, { async: asyncFn });

        const raw = { foo: "bar" };
        const result = await runAsyncValidators(TYPE, raw, CTX, controller.signal);

        expect(asyncFn).toHaveBeenCalledWith(raw, CTX, controller.signal);
        expect(result).toBe(issues);
    });

    it("swallows an AbortError and returns an empty array without logging", async () => {
        const consoleErr = vi.spyOn(console, "error").mockImplementation(() => {});
        const abortError = new DOMException("aborted", "AbortError");
        registerValidators(TYPE, {
            async: async () => {
                throw abortError;
            },
        });

        await expect(runAsyncValidators(TYPE, {}, CTX, controller.signal)).resolves.toEqual([]);
        expect(consoleErr).not.toHaveBeenCalled();
        consoleErr.mockRestore();
    });

    it("catches non-abort exceptions and returns an empty array, logging the error", async () => {
        const consoleErr = vi.spyOn(console, "error").mockImplementation(() => {});
        registerValidators(TYPE, {
            async: async () => {
                throw new Error("network down");
            },
        });

        await expect(runAsyncValidators(TYPE, {}, CTX, controller.signal)).resolves.toEqual([]);
        expect(consoleErr).toHaveBeenCalled();
        consoleErr.mockRestore();
    });
});

describe("isUint16", () => {
    it.each([0, 1, 65535])("accepts %s", (v) => expect(isUint16(v)).toBe(true));
    it.each([-1, 65536, 1.5, NaN, Infinity, "10", null, undefined, {}])("rejects %s", (v) =>
        expect(isUint16(v)).toBe(false),
    );
});

describe("normalizeFqdn", () => {
    it("drops one or several trailing dots", () => {
        expect(normalizeFqdn("a.example.com.")).toBe("a.example.com");
        expect(normalizeFqdn("a.example.com..")).toBe("a.example.com");
    });

    it("lowercases", () => {
        expect(normalizeFqdn("WWW.Example.COM")).toBe("www.example.com");
    });

    it("turns the root and the empty string into the empty string", () => {
        expect(normalizeFqdn(".")).toBe("");
        expect(normalizeFqdn("")).toBe("");
    });
});

describe("isValidHostname", () => {
    it.each(["a", "mx1.example.com", "xn--bcher-kva.example", "a-b.c", "1.2.3"])(
        "accepts %s",
        (n) => expect(isValidHostname(n)).toBe(true),
    );

    it.each([
        "",
        "a..b",
        ".a",
        "-a.com",
        "a-.com",
        "a_b.com",
        "_dmarc.example.com",
        "a b.com",
        "é.com",
        "a.com.",
    ])("rejects %j", (n) => expect(isValidHostname(n)).toBe(false));

    it("enforces the 63 character label limit", () => {
        expect(isValidHostname("a".repeat(63) + ".com")).toBe(true);
        expect(isValidHostname("a".repeat(64) + ".com")).toBe(false);
    });

    it("enforces the 253 character name limit", () => {
        const label = "a".repeat(61);
        const ok = [label, label, label, label].join("."); // 247
        expect(isValidHostname(ok)).toBe(true);
        expect(isValidHostname(ok + ".bbbbbbb")).toBe(false); // 255
    });
});

describe("inZoneSubdomain", () => {
    it("returns '' for the apex, ignoring case and trailing dots", () => {
        expect(inZoneSubdomain("Example.com.", "example.com")).toBe("");
    });

    it("returns the relative name, possibly multi-label", () => {
        expect(inZoneSubdomain("mx.example.com.", "example.com.")).toBe("mx");
        expect(inZoneSubdomain("a.b.example.com", "example.com.")).toBe("a.b");
    });

    it("returns null outside the zone", () => {
        expect(inZoneSubdomain("mx.other.org.", "example.com.")).toBeNull();
    });

    it("does not match a suffix that is not on a label boundary", () => {
        expect(inZoneSubdomain("notexample.com.", "example.com.")).toBeNull();
    });

    it("does not match a parent of the origin", () => {
        expect(inZoneSubdomain("com.", "example.com.")).toBeNull();
    });

    it("returns null when the origin is empty or the root", () => {
        expect(inZoneSubdomain("example.com.", "")).toBeNull();
        expect(inZoneSubdomain("example.com.", ".")).toBeNull();
    });
});

describe("originFqdn", () => {
    it("returns the domain of the origin", () => {
        expect(originFqdn(buildContext("@", ORIGIN, null))).toBe("example.com.");
    });

    it("returns '' when the origin has no domain or is missing", () => {
        expect(originFqdn(buildContext("@", makeDomain({ domain: undefined as any }), null))).toBe(
            "",
        );
        expect(originFqdn(buildContext("@", null as any, null))).toBe("");
    });
});

describe("isCnameOwner and hasAddress", () => {
    const alias = (rrtype?: number) =>
        makeService(
            "svcs.Alias",
            rrtype === undefined ? {} : { record: { Hdr: { Rrtype: rrtype } } },
        );
    const ctxWith = (services: Record<string, any[]>) =>
        buildContext("@", ORIGIN, makeZone({ services }));

    it("isCnameOwner is true for a CNAME alias", () => {
        expect(isCnameOwner(ctxWith({ mx: [alias(5)] }), "mx")).toBe(true);
    });

    it("isCnameOwner is true when a CNAME sits among other aliases", () => {
        expect(isCnameOwner(ctxWith({ mx: [alias(39), alias(5)] }), "mx")).toBe(true);
    });

    it("isCnameOwner is false for ALIAS and DNAME", () => {
        expect(isCnameOwner(ctxWith({ mx: [alias(65401)] }), "mx")).toBe(false);
        expect(isCnameOwner(ctxWith({ mx: [alias(39)] }), "mx")).toBe(false);
    });

    it("isCnameOwner is false without record, service, zone or name", () => {
        expect(isCnameOwner(ctxWith({ mx: [alias()] }), "mx")).toBe(false);
        expect(isCnameOwner(ctxWith({ mx: [] }), "mx")).toBe(false);
        expect(isCnameOwner(ctxWith({}), "mx")).toBe(false);
        expect(isCnameOwner(buildContext("@", ORIGIN, null), "mx")).toBe(false);
    });

    it("hasAddress is true with an abstract.Server or any alias", () => {
        expect(hasAddress(ctxWith({ mx: [makeService("abstract.Server")] }), "mx")).toBe(true);
        expect(hasAddress(ctxWith({ mx: [alias(5)] }), "mx")).toBe(true);
    });

    it("hasAddress is false with unrelated services, nothing or no zone", () => {
        expect(hasAddress(ctxWith({ mx: [makeService("svcs.TXT")] }), "mx")).toBe(false);
        expect(hasAddress(ctxWith({}), "mx")).toBe(false);
        expect(hasAddress(buildContext("@", ORIGIN, null), "mx")).toBe(false);
    });
});

describe("isValidDnsName", () => {
    it("accepts underscore labels", () => {
        expect(isValidDnsName("_sip._tcp.example.com")).toBe(true);
        expect(isValidDnsName("_dmarc.example.com")).toBe(true);
    });

    it("still rejects bad labels", () => {
        expect(isValidDnsName("")).toBe(false);
        expect(isValidDnsName("not a name")).toBe(false);
        expect(isValidDnsName("-bad.example.com")).toBe(false);
        expect(isValidDnsName("a".repeat(64) + ".example.com")).toBe(false);
        expect(isValidDnsName("a.".repeat(130) + "com")).toBe(false);
    });

    it("strips only one leading underscore", () => {
        expect(isValidDnsName("__x.example.com")).toBe(false);
    });
});

describe("recordFqdn", () => {
    const ctx = (dn: string) => buildContext(dn, ORIGIN, null);

    it("falls back on the service subdomain when the header name is empty", () => {
        expect(recordFqdn("", ctx("www"))).toBe("www.example.com");
        expect(recordFqdn(undefined, ctx("www"))).toBe("www.example.com");
    });

    it("is the apex for an empty or @ name on an empty subdomain", () => {
        expect(recordFqdn("", ctx(""))).toBe("example.com");
        expect(recordFqdn("@", ctx(""))).toBe("example.com");
    });

    it("joins the header name and the subdomain", () => {
        expect(recordFqdn("_sip._tcp", ctx("sub"))).toBe("_sip._tcp.sub.example.com");
        expect(recordFqdn("_sip._tcp", ctx(""))).toBe("_sip._tcp.example.com");
    });
});

describe("isCnameOwner with a SubAlias", () => {
    const sub = (Name: string) =>
        buildContext(
            "",
            ORIGIN,
            makeZone({
                services: { x: [makeService("svcs.SpecialCNAME", { cname: { Hdr: { Name } } })] },
            }),
        );

    it("is true for an empty header name", () => {
        expect(isCnameOwner(sub(""), "x")).toBe(true);
    });

    it("is false under an underscore name", () => {
        expect(isCnameOwner(sub("_sip._tcp"), "x")).toBe(false);
    });
});

describe("checkDigest", () => {
    const L = { 1: 40, 2: 64 };

    it("reports an unknown type without lengths", () => {
        expect(checkDigest("aa", 9, L)).toEqual({ status: "unknown-type" });
    });

    it("accepts a digest of the right length, whatever its case", () => {
        expect(checkDigest("A".repeat(40), 1, L)).toEqual({ status: "ok", expected: 40, got: 40 });
    });

    it("tells a truncated digest from a non hexadecimal one", () => {
        expect(checkDigest("abcd", 1, L)).toEqual({ status: "bad-length", expected: 40, got: 4 });
        expect(checkDigest("z".repeat(40), 1, L)).toEqual({
            status: "not-hex",
            expected: 40,
            got: 40,
        });
    });

    it("treats an empty digest as not hexadecimal", () => {
        expect(checkDigest("", 2, L)).toEqual({ status: "not-hex", expected: 64, got: 0 });
    });
});

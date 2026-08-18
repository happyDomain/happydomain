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
    getValidators,
    hasValidators,
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

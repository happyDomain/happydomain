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

import { beforeEach, describe, expect, it, vi } from "vitest";
import { get } from "svelte/store";

const diffZoneMock = vi.fn();

vi.mock("$lib/api/zone", () => ({
    diffZone: (...args: unknown[]) => diffZoneMock(...args),
}));

import { thisZone } from "$lib/stores/thiszone";
import { makeDomain, makeZone } from "$lib/test-utils/fixtures";
import {
    getCachedDiffZone,
    getCachedDiffZoneSummary,
    invalidateZoneDiff,
    zoneDiffVersion,
} from "./zonediff";

// Corrections of the given kinds: 1 addition, 2 update, 3 deletion, 99 other.
function corrections(...kinds: number[]) {
    return kinds.map((kind, i) => ({ id: `c${i}`, msg: `change ${i}`, kind }));
}

describe("zonediff store", () => {
    const domain = makeDomain();

    beforeEach(() => {
        diffZoneMock.mockReset();
        invalidateZoneDiff();
        thisZone.set(null);
    });

    describe("getCachedDiffZone", () => {
        it("fetches the full diff from the API", async () => {
            const diff = corrections(1);
            diffZoneMock.mockResolvedValue(diff);

            const result = await getCachedDiffZone(domain, "from", "to");

            expect(result).toBe(diff);
            expect(diffZoneMock).toHaveBeenCalledWith(domain, "from", "to");
        });

        it("caches identical requests instead of calling the API twice", async () => {
            diffZoneMock.mockResolvedValue([]);

            await getCachedDiffZone(domain, "from", "to");
            await getCachedDiffZone(domain, "from", "to");

            expect(diffZoneMock).toHaveBeenCalledTimes(1);
        });

        it("shares the in-flight promise between concurrent callers", async () => {
            let resolve!: (v: unknown) => void;
            diffZoneMock.mockReturnValue(
                new Promise((r) => {
                    resolve = r;
                }),
            );

            const p1 = getCachedDiffZone(domain, "from", "to");
            const p2 = getCachedDiffZone(domain, "from", "to");

            expect(diffZoneMock).toHaveBeenCalledTimes(1);
            const diff = corrections(1, 3);
            resolve(diff);

            expect(await p1).toBe(diff);
            expect(await p2).toBe(diff);
        });

        it("distinguishes requests by domain id, zoneFrom, and zoneTo", async () => {
            diffZoneMock.mockResolvedValue([]);

            await getCachedDiffZone(domain, "from", "to");
            await getCachedDiffZone(domain, "other-from", "to");
            await getCachedDiffZone(domain, "from", "other-to");
            await getCachedDiffZone(makeDomain({ id: "other-domain" }), "from", "to");

            expect(diffZoneMock).toHaveBeenCalledTimes(4);
        });

        it("does not cache a rejected request, so a retry hits the API again", async () => {
            diffZoneMock.mockRejectedValueOnce(new Error("boom"));
            diffZoneMock.mockResolvedValueOnce(corrections(3));

            await expect(getCachedDiffZone(domain, "from", "to")).rejects.toThrow("boom");
            const result = await getCachedDiffZone(domain, "from", "to");

            expect(result).toEqual(corrections(3));
            expect(diffZoneMock).toHaveBeenCalledTimes(2);
        });

        it("does not evict a fresher entry when an older rejected promise settles late", async () => {
            let rejectFirst!: (e: Error) => void;
            diffZoneMock.mockReturnValueOnce(
                new Promise((_, reject) => {
                    rejectFirst = reject;
                }),
            );

            const firstCall = getCachedDiffZone(domain, "from", "to");
            firstCall.catch(() => {});

            // A cache clear in between (e.g. zone switch) replaces the pending
            // entry; when the stale request finally rejects it must not delete
            // the newer promise sitting under the same key.
            invalidateZoneDiff();

            const fresh = corrections(1);
            diffZoneMock.mockResolvedValueOnce(fresh);
            const secondCall = getCachedDiffZone(domain, "from", "to");

            rejectFirst(new Error("stale failure"));
            await expect(firstCall).rejects.toThrow("stale failure");

            expect(await secondCall).toBe(fresh);
            // The cache should still return the second promise's result afterwards.
            expect(await getCachedDiffZone(domain, "from", "to")).toBe(fresh);
            expect(diffZoneMock).toHaveBeenCalledTimes(2);
        });
    });

    describe("getCachedDiffZoneSummary", () => {
        it("counts the changes of the full diff, leaving out the other messages", async () => {
            diffZoneMock.mockResolvedValue(corrections(1, 2, 3, 99, 1));

            const result = await getCachedDiffZoneSummary(domain, "from", "to");

            expect(result).toEqual({ nbDiffs: 4 });
            expect(diffZoneMock).toHaveBeenCalledWith(domain, "from", "to");
        });

        it("shares a pending request with the full diff", async () => {
            let resolve!: (v: unknown) => void;
            diffZoneMock.mockReturnValue(
                new Promise((r) => {
                    resolve = r;
                }),
            );

            const summary = getCachedDiffZoneSummary(domain, "from", "to");
            const full = getCachedDiffZone(domain, "from", "to");

            expect(diffZoneMock).toHaveBeenCalledTimes(1);
            const diff = corrections(1, 2);
            resolve(diff);

            expect(await summary).toEqual({ nbDiffs: 2 });
            expect(await full).toBe(diff);
        });

        it("reuses a full diff already fetched", async () => {
            diffZoneMock.mockResolvedValue(corrections(3));

            await getCachedDiffZone(domain, "from", "to");
            const result = await getCachedDiffZoneSummary(domain, "from", "to");

            expect(result).toEqual({ nbDiffs: 1 });
            expect(diffZoneMock).toHaveBeenCalledTimes(1);
        });

        it("rejects like the full diff, and a retry hits the API again", async () => {
            diffZoneMock.mockRejectedValueOnce(new Error("network down"));
            diffZoneMock.mockResolvedValueOnce(corrections(1, 1));

            await expect(getCachedDiffZoneSummary(domain, "from", "to")).rejects.toThrow(
                "network down",
            );
            const result = await getCachedDiffZoneSummary(domain, "from", "to");

            expect(result).toEqual({ nbDiffs: 2 });
            expect(diffZoneMock).toHaveBeenCalledTimes(2);
        });
    });

    describe("invalidateZoneDiff", () => {
        it("clears the cache so the next call refetches", async () => {
            diffZoneMock.mockResolvedValue([]);

            await getCachedDiffZoneSummary(domain, "from", "to");
            await getCachedDiffZone(domain, "from", "to");

            invalidateZoneDiff();

            await getCachedDiffZoneSummary(domain, "from", "to");
            await getCachedDiffZone(domain, "from", "to");

            expect(diffZoneMock).toHaveBeenCalledTimes(2);
        });

        it("bumps zoneDiffVersion", () => {
            const before = get(zoneDiffVersion);

            invalidateZoneDiff();

            expect(get(zoneDiffVersion)).toBe(before + 1);
        });
    });

    describe("auto-invalidation on thisZone changes", () => {
        it("clears the caches and bumps the version when the zone reference changes", async () => {
            diffZoneMock.mockResolvedValue([]);
            await getCachedDiffZoneSummary(domain, "from", "to");

            const versionBefore = get(zoneDiffVersion);

            thisZone.set(makeZone());

            expect(get(zoneDiffVersion)).toBe(versionBefore + 1);

            await getCachedDiffZoneSummary(domain, "from", "to");
            expect(diffZoneMock).toHaveBeenCalledTimes(2);
        });

        it("does not invalidate when the same zone reference is set again", async () => {
            const zone = makeZone();
            thisZone.set(zone);

            diffZoneMock.mockResolvedValue([]);
            await getCachedDiffZoneSummary(domain, "from", "to");

            const versionBefore = get(zoneDiffVersion);

            thisZone.set(zone);

            expect(get(zoneDiffVersion)).toBe(versionBefore);

            await getCachedDiffZoneSummary(domain, "from", "to");
            expect(diffZoneMock).toHaveBeenCalledTimes(1);
        });

        it("does invalidate when a different zone object with the same content is set", async () => {
            thisZone.set(makeZone());

            diffZoneMock.mockResolvedValue([]);
            await getCachedDiffZoneSummary(domain, "from", "to");

            const versionBefore = get(zoneDiffVersion);

            thisZone.set(makeZone());

            expect(get(zoneDiffVersion)).toBe(versionBefore + 1);
        });
    });
});

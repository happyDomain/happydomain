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
const diffZoneSummaryMock = vi.fn();

vi.mock("$lib/api/zone", () => ({
    diffZone: (...args: unknown[]) => diffZoneMock(...args),
    diffZoneSummary: (...args: unknown[]) => diffZoneSummaryMock(...args),
}));

import { thisZone } from "$lib/stores/thiszone";
import { makeDomain, makeZone } from "$lib/test-utils/fixtures";
import {
    getCachedDiffZone,
    getCachedDiffZoneSummary,
    invalidateZoneDiff,
    zoneDiffVersion,
} from "./zonediff";

describe("zonediff store", () => {
    const domain = makeDomain();

    beforeEach(() => {
        diffZoneMock.mockReset();
        diffZoneSummaryMock.mockReset();
        invalidateZoneDiff();
        thisZone.set(null);
    });

    describe("getCachedDiffZoneSummary", () => {
        it("fetches the summary from the API", async () => {
            diffZoneSummaryMock.mockResolvedValue({ nbDiffs: 3 });

            const result = await getCachedDiffZoneSummary(domain, "from", "to");

            expect(result).toEqual({ nbDiffs: 3 });
            expect(diffZoneSummaryMock).toHaveBeenCalledWith(domain, "from", "to");
        });

        it("caches identical requests instead of calling the API twice", async () => {
            diffZoneSummaryMock.mockResolvedValue({ nbDiffs: 3 });

            await getCachedDiffZoneSummary(domain, "from", "to");
            await getCachedDiffZoneSummary(domain, "from", "to");

            expect(diffZoneSummaryMock).toHaveBeenCalledTimes(1);
        });

        it("shares the in-flight promise between concurrent callers", async () => {
            let resolve!: (v: { nbDiffs: number }) => void;
            diffZoneSummaryMock.mockReturnValue(
                new Promise((r) => {
                    resolve = r;
                }),
            );

            const p1 = getCachedDiffZoneSummary(domain, "from", "to");
            const p2 = getCachedDiffZoneSummary(domain, "from", "to");

            expect(diffZoneSummaryMock).toHaveBeenCalledTimes(1);
            resolve({ nbDiffs: 7 });

            expect(await p1).toEqual({ nbDiffs: 7 });
            expect(await p2).toEqual({ nbDiffs: 7 });
        });

        it("distinguishes requests by domain id, zoneFrom, and zoneTo", async () => {
            diffZoneSummaryMock.mockResolvedValue({ nbDiffs: 1 });

            await getCachedDiffZoneSummary(domain, "from", "to");
            await getCachedDiffZoneSummary(domain, "other-from", "to");
            await getCachedDiffZoneSummary(domain, "from", "other-to");
            await getCachedDiffZoneSummary(makeDomain({ id: "other-domain" }), "from", "to");

            expect(diffZoneSummaryMock).toHaveBeenCalledTimes(4);
        });

        it("does not cache a rejected request, so a retry hits the API again", async () => {
            diffZoneSummaryMock.mockRejectedValueOnce(new Error("network down"));
            diffZoneSummaryMock.mockResolvedValueOnce({ nbDiffs: 2 });

            await expect(getCachedDiffZoneSummary(domain, "from", "to")).rejects.toThrow(
                "network down",
            );
            const result = await getCachedDiffZoneSummary(domain, "from", "to");

            expect(result).toEqual({ nbDiffs: 2 });
            expect(diffZoneSummaryMock).toHaveBeenCalledTimes(2);
        });

        it("does not evict a fresher entry when an older rejected promise settles late", async () => {
            let rejectFirst!: (e: Error) => void;
            diffZoneSummaryMock.mockReturnValueOnce(
                new Promise((_, reject) => {
                    rejectFirst = reject;
                }),
            );

            const firstCall = getCachedDiffZoneSummary(domain, "from", "to");
            firstCall.catch(() => {});

            // A cache clear in between (e.g. zone switch) replaces the pending
            // entry; when the stale request finally rejects it must not delete
            // the newer promise sitting under the same key.
            invalidateZoneDiff();

            diffZoneSummaryMock.mockResolvedValueOnce({ nbDiffs: 9 });
            const secondCall = getCachedDiffZoneSummary(domain, "from", "to");

            rejectFirst(new Error("stale failure"));
            await expect(firstCall).rejects.toThrow("stale failure");

            expect(await secondCall).toEqual({ nbDiffs: 9 });
            // The cache should still return the second promise's result afterwards.
            expect(await getCachedDiffZoneSummary(domain, "from", "to")).toEqual({ nbDiffs: 9 });
            expect(diffZoneSummaryMock).toHaveBeenCalledTimes(2);
        });
    });

    describe("getCachedDiffZone", () => {
        it("fetches the full diff from the API", async () => {
            const corrections = [{ Type: "add" }];
            diffZoneMock.mockResolvedValue(corrections);

            const result = await getCachedDiffZone(domain, "from", "to");

            expect(result).toBe(corrections);
            expect(diffZoneMock).toHaveBeenCalledWith(domain, "from", "to");
        });

        it("caches identical requests instead of calling the API twice", async () => {
            diffZoneMock.mockResolvedValue([]);

            await getCachedDiffZone(domain, "from", "to");
            await getCachedDiffZone(domain, "from", "to");

            expect(diffZoneMock).toHaveBeenCalledTimes(1);
        });

        it("keeps the summary cache and the full diff cache independent", async () => {
            diffZoneSummaryMock.mockResolvedValue({ nbDiffs: 1 });
            diffZoneMock.mockResolvedValue([{ Type: "add" }]);

            await getCachedDiffZoneSummary(domain, "from", "to");
            await getCachedDiffZone(domain, "from", "to");

            expect(diffZoneSummaryMock).toHaveBeenCalledTimes(1);
            expect(diffZoneMock).toHaveBeenCalledTimes(1);
        });

        it("does not cache a rejected request, so a retry hits the API again", async () => {
            diffZoneMock.mockRejectedValueOnce(new Error("boom"));
            diffZoneMock.mockResolvedValueOnce([{ Type: "del" }]);

            await expect(getCachedDiffZone(domain, "from", "to")).rejects.toThrow("boom");
            const result = await getCachedDiffZone(domain, "from", "to");

            expect(result).toEqual([{ Type: "del" }]);
            expect(diffZoneMock).toHaveBeenCalledTimes(2);
        });
    });

    describe("invalidateZoneDiff", () => {
        it("clears both caches so the next call refetches", async () => {
            diffZoneSummaryMock.mockResolvedValue({ nbDiffs: 1 });
            diffZoneMock.mockResolvedValue([]);

            await getCachedDiffZoneSummary(domain, "from", "to");
            await getCachedDiffZone(domain, "from", "to");

            invalidateZoneDiff();

            await getCachedDiffZoneSummary(domain, "from", "to");
            await getCachedDiffZone(domain, "from", "to");

            expect(diffZoneSummaryMock).toHaveBeenCalledTimes(2);
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
            diffZoneSummaryMock.mockResolvedValue({ nbDiffs: 1 });
            await getCachedDiffZoneSummary(domain, "from", "to");

            const versionBefore = get(zoneDiffVersion);

            thisZone.set(makeZone());

            expect(get(zoneDiffVersion)).toBe(versionBefore + 1);

            await getCachedDiffZoneSummary(domain, "from", "to");
            expect(diffZoneSummaryMock).toHaveBeenCalledTimes(2);
        });

        it("does not invalidate when the same zone reference is set again", async () => {
            const zone = makeZone();
            thisZone.set(zone);

            diffZoneSummaryMock.mockResolvedValue({ nbDiffs: 1 });
            await getCachedDiffZoneSummary(domain, "from", "to");

            const versionBefore = get(zoneDiffVersion);

            thisZone.set(zone);

            expect(get(zoneDiffVersion)).toBe(versionBefore);

            await getCachedDiffZoneSummary(domain, "from", "to");
            expect(diffZoneSummaryMock).toHaveBeenCalledTimes(1);
        });

        it("does invalidate when a different zone object with the same content is set", async () => {
            thisZone.set(makeZone());

            diffZoneSummaryMock.mockResolvedValue({ nbDiffs: 1 });
            await getCachedDiffZoneSummary(domain, "from", "to");

            const versionBefore = get(zoneDiffVersion);

            thisZone.set(makeZone());

            expect(get(zoneDiffVersion)).toBe(versionBefore + 1);
        });
    });
});

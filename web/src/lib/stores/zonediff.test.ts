import { describe, it, expect, beforeEach, vi } from "vitest";
import { get } from "svelte/store";

vi.mock("$lib/api/zone", () => ({
    diffZone: vi.fn(),
    diffZoneSummary: vi.fn(),
}));

import {
    zoneDiffVersion,
    invalidateZoneDiff,
    getCachedDiffZone,
    getCachedDiffZoneSummary,
} from "./zonediff";
import { thisZone } from "./thiszone";
import { diffZone, diffZoneSummary } from "$lib/api/zone";
import type { Domain } from "$lib/model/domain";
import type { Zone } from "$lib/model/zone";

const apiDiffZone = vi.mocked(diffZone);
const apiSummary = vi.mocked(diffZoneSummary);

const fakeDomain: Domain = {
    id: "domain-1",
    id_owner: "u",
    id_provider: "p",
    domain: "example.com",
    group: "",
    zone_history: [],
} as Domain;

describe("zonediff caching", () => {
    beforeEach(() => {
        thisZone.set(null);
        invalidateZoneDiff();
        apiDiffZone.mockReset();
        apiSummary.mockReset();
    });

    it("getCachedDiffZoneSummary calls the API once for the same key", async () => {
        apiSummary.mockResolvedValue({ nbDiffs: 3 });

        const a = await getCachedDiffZoneSummary(fakeDomain, "z1", "z2");
        const b = await getCachedDiffZoneSummary(fakeDomain, "z1", "z2");

        expect(a).toEqual({ nbDiffs: 3 });
        expect(b).toEqual({ nbDiffs: 3 });
        expect(apiSummary).toHaveBeenCalledTimes(1);
    });

    it("getCachedDiffZoneSummary keys are sensitive to zoneFrom/zoneTo", async () => {
        apiSummary.mockResolvedValue({ nbDiffs: 1 });

        await getCachedDiffZoneSummary(fakeDomain, "z1", "z2");
        await getCachedDiffZoneSummary(fakeDomain, "z1", "z3");

        expect(apiSummary).toHaveBeenCalledTimes(2);
    });

    it("getCachedDiffZone caches full diffs by (domainId, from, to)", async () => {
        apiDiffZone.mockResolvedValue([] as never);

        await getCachedDiffZone(fakeDomain, "from", "to");
        await getCachedDiffZone(fakeDomain, "from", "to");

        expect(apiDiffZone).toHaveBeenCalledTimes(1);
    });

    it("invalidateZoneDiff clears cached entries and bumps the version", async () => {
        apiSummary.mockResolvedValue({ nbDiffs: 5 });

        await getCachedDiffZoneSummary(fakeDomain, "a", "b");
        const versionBefore = get(zoneDiffVersion);

        invalidateZoneDiff();
        await getCachedDiffZoneSummary(fakeDomain, "a", "b");

        expect(apiSummary).toHaveBeenCalledTimes(2);
        expect(get(zoneDiffVersion)).toBeGreaterThan(versionBefore);
    });

    it("a thisZone change invalidates the cache automatically", async () => {
        apiSummary.mockResolvedValue({ nbDiffs: 0 });

        await getCachedDiffZoneSummary(fakeDomain, "a", "b");
        const versionBefore = get(zoneDiffVersion);

        const newZone = { id: "z-new", services: {} } as unknown as Zone;
        thisZone.set(newZone);

        // The subscriber bumps the version; the next call must hit the API again.
        await getCachedDiffZoneSummary(fakeDomain, "a", "b");

        expect(apiSummary).toHaveBeenCalledTimes(2);
        expect(get(zoneDiffVersion)).toBeGreaterThan(versionBefore);
    });

    it("a rejected summary fetch evicts itself from the cache", async () => {
        apiSummary
            .mockRejectedValueOnce(new Error("first failure"))
            .mockResolvedValueOnce({ nbDiffs: 1 });

        await expect(getCachedDiffZoneSummary(fakeDomain, "x", "y")).rejects.toThrow(
            "first failure",
        );
        // The retry call should re-issue the API request, since the rejected
        // promise was evicted by the catch handler.
        const result = await getCachedDiffZoneSummary(fakeDomain, "x", "y");
        expect(result).toEqual({ nbDiffs: 1 });
        expect(apiSummary).toHaveBeenCalledTimes(2);
    });
});

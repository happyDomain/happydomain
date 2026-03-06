// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
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

import { diffZone as APIDiffZone, diffZoneSummary as APIDiffZoneSummary } from "$lib/api/zone";
import type { Correction } from "$lib/model/correction";
import type { Domain } from "$lib/model/domain";
import type { Zone } from "$lib/model/zone";
import { thisZone } from "$lib/stores/thiszone";

const summaryCache = new Map<string, Promise<{ nbDiffs: number }>>();
const fullDiffCache = new Map<string, Promise<Array<Correction>>>();

function makeCacheKey(domainId: string, zoneFrom: string, zoneTo: string): string {
    return `${domainId}:${zoneFrom}:${zoneTo}`;
}

// Auto-invalidate on any thisZone change (zone switch, service/record modification)
let previousZone: Zone | null = null;
thisZone.subscribe((zone) => {
    if (zone !== previousZone) {
        summaryCache.clear();
        fullDiffCache.clear();
        previousZone = zone;
    }
});

export function invalidateZoneDiff(): void {
    summaryCache.clear();
    fullDiffCache.clear();
}

export function getCachedDiffZoneSummary(
    domain: Domain,
    zoneFrom: string,
    zoneTo: string,
): Promise<{ nbDiffs: number }> {
    const key = makeCacheKey(domain.id, zoneFrom, zoneTo);
    if (!summaryCache.has(key)) {
        const p = APIDiffZoneSummary(domain, zoneFrom, zoneTo);
        summaryCache.set(key, p);
        p.catch(() => {
            if (summaryCache.get(key) === p) summaryCache.delete(key);
        });
    }
    return summaryCache.get(key)!;
}

export function getCachedDiffZone(
    domain: Domain,
    zoneFrom: string,
    zoneTo: string,
): Promise<Array<Correction>> {
    const key = makeCacheKey(domain.id, zoneFrom, zoneTo);
    if (!fullDiffCache.has(key)) {
        const p = APIDiffZone(domain, zoneFrom, zoneTo);
        fullDiffCache.set(key, p);
        p.catch(() => {
            if (fullDiffCache.get(key) === p) fullDiffCache.delete(key);
        });
    }
    return fullDiffCache.get(key)!;
}

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

import { writable } from "svelte/store";
import { diffZone as APIDiffZone } from "$lib/api/zone";
import type { Correction } from "$lib/model/correction";
import type { Domain } from "$lib/model/domain";
import type { Zone } from "$lib/model/zone";
import { thisZone } from "$lib/stores/thiszone";

export const zoneDiffVersion = writable(0);

const diffCache = new Map<string, Promise<Array<Correction>>>();

function makeCacheKey(domainId: string, zoneFrom: string, zoneTo: string): string {
    return `${domainId}:${zoneFrom}:${zoneTo}`;
}

// Auto-invalidate on any thisZone change (zone switch, service/record modification)
let previousZone: Zone | null = null;
thisZone.subscribe((zone) => {
    if (zone !== previousZone) {
        diffCache.clear();
        previousZone = zone;
        zoneDiffVersion.update((v) => v + 1);
    }
});

export function invalidateZoneDiff(): void {
    diffCache.clear();
    zoneDiffVersion.update((v) => v + 1);
}

export function getCachedDiffZone(
    domain: Domain,
    zoneFrom: string,
    zoneTo: string,
): Promise<Array<Correction>> {
    const key = makeCacheKey(domain.id, zoneFrom, zoneTo);
    if (!diffCache.has(key)) {
        const p = APIDiffZone(domain, zoneFrom, zoneTo);
        diffCache.set(key, p);
        p.catch(() => {
            if (diffCache.get(key) === p) diffCache.delete(key);
        });
    }
    return diffCache.get(key)!;
}

// Comparing with the provider ("@") costs the retrieval of the live zone,
// whatever the server sends back: the summary is derived from the full diff,
// so that the publish button and the diff modal share a single request.
export function getCachedDiffZoneSummary(
    domain: Domain,
    zoneFrom: string,
    zoneTo: string,
): Promise<{ nbDiffs: number }> {
    return getCachedDiffZone(domain, zoneFrom, zoneTo).then((corrections) => ({
        // Other (99) carries informative messages, not changes.
        nbDiffs: corrections ? corrections.filter((c) => c.kind !== 99).length : 0,
    }));
}

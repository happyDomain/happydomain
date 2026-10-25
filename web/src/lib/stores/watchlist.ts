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

import { get, writable, type Writable } from "svelte/store";

import { listDomainWatches } from "$lib/api/watchlist";
import type { DomainWatch } from "$lib/model/watchlist";

export const watchlist: Writable<DomainWatch[] | undefined> = writable(undefined);

export async function refreshWatchlist(): Promise<DomainWatch[]> {
    const data = await listDomainWatches();
    watchlist.set(data);
    return data;
}

let loading: Promise<DomainWatch[]> | undefined;

/**
 * Load the watchlist unless it is already loaded, or being loaded: the
 * components needing it can all ask for it at once.
 */
export function ensureWatchlist(): Promise<DomainWatch[]> {
    const list = get(watchlist);
    if (list !== undefined) return Promise.resolve(list);
    loading ??= refreshWatchlist().finally(() => (loading = undefined));
    return loading;
}

/** Add a watch to the list, or replace the one with the same identifier. */
export function upsertWatch(w: DomainWatch): void {
    watchlist.update((list) => {
        if (!list) return list;
        const i = list.findIndex((e) => e.id === w.id);
        if (i < 0) return [w, ...list];
        const copy = [...list];
        copy[i] = w;
        return copy;
    });
}

export function removeWatch(id: string): void {
    watchlist.update((list) => list?.filter((e) => e.id !== id));
}

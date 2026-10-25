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

import {
    deleteWatchlistByWatchId,
    getWatchlist,
    postWatchlist,
    postWatchlistByWatchIdCheck,
    putWatchlistByWatchId,
} from "$lib/api-base/sdk.gen";
import type { DomainWatch, DomainWatchInput } from "$lib/model/watchlist";
import { unwrapEmptyResponse, unwrapSdkResponse } from "./errors";

export async function listDomainWatches(): Promise<DomainWatch[]> {
    return unwrapSdkResponse(await getWatchlist()) as DomainWatch[];
}

export async function addDomainWatch(input: DomainWatchInput): Promise<DomainWatch> {
    return unwrapSdkResponse(await postWatchlist({ body: input })) as DomainWatch;
}

export async function updateDomainWatch(id: string, note: string): Promise<DomainWatch> {
    return unwrapSdkResponse(
        await putWatchlistByWatchId({ path: { watchId: id }, body: { note } }),
    ) as DomainWatch;
}

export async function deleteDomainWatch(id: string): Promise<void> {
    unwrapEmptyResponse(await deleteWatchlistByWatchId({ path: { watchId: id } }));
}

export async function checkDomainWatch(id: string): Promise<DomainWatch> {
    return unwrapSdkResponse(
        await postWatchlistByWatchIdCheck({ path: { watchId: id } }),
    ) as DomainWatch;
}

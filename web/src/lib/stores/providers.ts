// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
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

import { listProviders } from "$lib/api/provider";
import type { Provider, ProviderInfos } from "$lib/model/provider";
import { filteredProvider } from "$lib/stores/home";
import { derived, get, writable, type Writable } from "svelte/store";

export const providers: Writable<Array<Provider> | undefined> = writable(undefined);
export const providersSpecs: Writable<Record<string, ProviderInfos> | undefined> =
    writable(undefined);

export async function refreshProviders() {
    const data = await listProviders();
    providers.set(data);

    const $filteredProvider = get(filteredProvider);
    if ($filteredProvider !== null) {
        let found = false;
        for (const provider of data) {
            if (provider._id == $filteredProvider._id) {
                found = true;
                break;
            }
        }
        if (!found) {
            filteredProvider.set(null);
        }
    }

    return data;
}

export const providers_idx = derived(providers, ($providers: Array<Provider> | undefined) => {
    const idx: Record<string, Provider> = {};

    if ($providers) {
        for (const p of $providers) {
            idx[p._id] = p;
        }
    }

    return idx;
});

export async function refreshProvidersSpecs() {
    const res = await fetch("/api/providers/_specs", { headers: { Accept: "application/json" } });
    if (res.status == 200) {
        const map = await res.json();
        providersSpecs.set(map);
        return map;
    } else {
        throw new Error((await res.json()).errmsg);
    }
}

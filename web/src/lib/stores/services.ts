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

import { getServiceSpecs } from "$lib/api-base/sdk.gen";
import { unwrapSdkResponse } from "$lib/api/errors";
import { derived, writable, type Writable } from "svelte/store";
import type { ServiceInfos } from "$lib/model/service_specs.svelte";

export const servicesSpecs: Writable<Record<string, ServiceInfos>> = writable({});
export const servicesSpecsLoaded: Writable<boolean> = writable(false);
export const servicesSpecsError: Writable<string | null> = writable(null);

export async function refreshServicesSpecs() {
    servicesSpecsLoaded.set(false);
    servicesSpecsError.set(null);

    try {
        const map = unwrapSdkResponse(await getServiceSpecs()) as Record<string, ServiceInfos>;
        servicesSpecs.set(map);
        servicesSpecsLoaded.set(true);
        return map;
    } catch (err) {
        const errmsg = err instanceof Error ? err.message : String(err);
        servicesSpecsError.set(errmsg);
        throw err;
    }
}

export const servicesSpecsList = derived(servicesSpecs, ($servicesSpecs: Record<string, ServiceInfos>) => {
    return Object.keys($servicesSpecs).map((idx) => $servicesSpecs[idx]);
});

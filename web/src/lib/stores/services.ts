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

import { writable, type Writable } from "svelte/store";
import type { ServiceInfos } from "$lib/model/service_specs";

export const servicesSpecs: Writable<null | Record<string, ServiceInfos>> = writable(null);

export async function refreshServicesSpecs() {
    const res = await fetch("/api/service_specs", { headers: { Accept: "application/json" } });
    if (res.status == 200) {
        const map = await res.json();
        servicesSpecs.set(map);
        return map;
    } else {
        throw new Error((await res.json()).errmsg);
    }
}

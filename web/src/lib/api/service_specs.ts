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

import { handleApiResponse } from "$lib/errors";
import type { ServiceInfos, ServiceSpec } from "$lib/model/service_specs";

export async function listServiceSpecs(): Promise<Record<string, ServiceInfos>> {
    const res = await fetch("/api/service_specs", {
        method: "GET",
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<Record<string, ServiceInfos>>(res);
}

export async function getServiceSpec(ssid: string): Promise<ServiceSpec> {
    if (ssid == "string" || ssid == "common.URL") {
        return Promise.resolve(<ServiceSpec>{ fields: [] });
    } else {
        const res = await fetch(`/api/service_specs/` + ssid, {
            method: "GET",
            headers: { Accept: "application/json" },
        });
        return await handleApiResponse<ServiceSpec>(res);
    }
}

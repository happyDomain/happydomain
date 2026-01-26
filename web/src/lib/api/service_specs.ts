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

import {
    getServiceSpecs,
    getServiceSpecsByServiceType,
    postServiceSpecsByServiceTypeInit,
} from "$lib/api-base/sdk.gen";
import type { ServiceInfos, ServiceSpec } from "$lib/model/service_specs.svelte";
import { unwrapSdkResponse } from "./errors";

export async function listServiceSpecs(): Promise<Record<string, ServiceInfos>> {
    return unwrapSdkResponse(await getServiceSpecs()) as Record<string, ServiceInfos>;
}

export async function getServiceSpec(ssid: string): Promise<ServiceSpec> {
    // Handle built-in types without making an API call
    if (ssid == "string" || ssid == "common.URL") {
        return Promise.resolve(<ServiceSpec>{ fields: null });
    } else {
        return unwrapSdkResponse(
            await getServiceSpecsByServiceType({
                path: { serviceType: ssid },
            }),
        ) as ServiceSpec;
    }
}

export async function initializeService(ssid: string): Promise<any> {
    return unwrapSdkResponse(
        await postServiceSpecsByServiceTypeInit({
            path: { serviceType: ssid },
        }),
    );
}

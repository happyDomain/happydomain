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
    getProviders,
    getProvidersByProviderId,
    getProvidersByProviderIdDomains,
    getProvidersByProviderIdDomainsByFqdn,
    postProviders,
    putProvidersByProviderId,
    deleteProvidersByProviderId,
} from "$lib/api-base/sdk.gen";
import type { Provider } from "$lib/model/provider";
import { unwrapSdkResponse, unwrapEmptyResponse } from "./errors";

export async function listProviders(): Promise<Array<Provider>> {
    return unwrapSdkResponse(await getProviders()) as Array<Provider>;
}

export async function getProvider(id: string): Promise<Provider> {
    return unwrapSdkResponse(
        await getProvidersByProviderId({
            path: { providerId: id },
        }),
    ) as Provider;
}

export async function listImportableDomains(provider: Provider): Promise<Array<string>> {
    return unwrapSdkResponse(
        await getProvidersByProviderIdDomains({
            path: { providerId: provider._id },
        }),
    ) as Array<string>;
}

/**
 * Create a domain at the provider.
 * Note: The old API used POST, but the current OpenAPI spec has GET endpoint.
 * This might need investigation if it doesn't work as expected.
 */
export async function createDomain(provider: Provider, fqdn: string): Promise<boolean> {
    return unwrapSdkResponse(
        await getProvidersByProviderIdDomainsByFqdn({
            path: { providerId: provider._id, fqdn } as any,
        }),
    ) as unknown as boolean;
}

export async function updateProvider(provider: Provider): Promise<Provider> {
    if (provider._id) {
        return unwrapSdkResponse(
            await putProvidersByProviderId({
                path: { providerId: provider._id },
                body: provider as any,
            }),
        ) as Provider;
    } else {
        return unwrapSdkResponse(
            await postProviders({
                body: provider as any,
            }),
        ) as Provider;
    }
}

export async function deleteProvider(id: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await deleteProvidersByProviderId({
            path: { providerId: id },
        }),
    );
}

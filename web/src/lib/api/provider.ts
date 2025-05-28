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

import { handleEmptyApiResponse, handleApiResponse } from "$lib/errors";
import type { Provider } from "$lib/model/provider";

export async function listProviders(): Promise<Array<Provider>> {
    const res = await fetch("/api/providers", { headers: { Accept: "application/json" } });
    return await handleApiResponse<Array<Provider>>(res);
}

export async function getProvider(id: string): Promise<Provider> {
    id = encodeURIComponent(id);
    const res = await fetch(`/api/providers/${id}`, { headers: { Accept: "application/json" } });
    return await handleApiResponse<Provider>(res);
}

export async function listImportableDomains(provider: Provider): Promise<Array<string>> {
    const res = await fetch(`/api/providers/${provider._id}/domains`, {
        method: "GET",
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<Array<string>>(res);
}

export async function createDomain(provider: Provider, fqdn: string): Promise<boolean> {
    const res = await fetch(`/api/providers/${provider._id}/domains/${fqdn}`, {
        method: "POST",
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<boolean>(res);
}

export async function updateProvider(provider: Provider): Promise<Provider> {
    const res = await fetch("/api/providers" + (provider._id ? `/${provider._id}` : ""), {
        method: provider._id ? "PUT" : "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify(provider),
    });
    return await handleApiResponse<Provider>(res);
}

export async function deleteProvider(id: string): Promise<boolean> {
    const res = await fetch(`/api/providers/${id}`, {
        method: "DELETE",
        headers: { Accept: "application/json" },
    });
    return await handleEmptyApiResponse(res);
}

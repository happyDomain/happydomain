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

import { handleEmptyApiResponse, handleApiResponse } from '$lib/errors';
import type { Domain, DomainLog } from '$lib/model/domain';
import type { Provider } from '$lib/model/provider';

export async function listDomains(): Promise<Array<Domain>> {
    const res = await fetch('/api/domains', {headers: {'Accept': 'application/json'}});
    return (await handleApiResponse<Array<Domain>>(res));
}

export async function getDomain(id: string): Promise<Domain> {
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${id}`, {headers: {'Accept': 'application/json'}});
    return await handleApiResponse<Domain>(res);
}

export async function addDomain(domain: string, provider: Provider|undefined): Promise<Domain> {
    const id_provider = provider?provider._id:undefined;

    const res = await fetch('/api/domains', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify({
            domain,
            id_provider,
        }),
    });
    return await handleApiResponse<Domain>(res);
}

export async function updateDomain(domain: Domain): Promise<Domain> {
    const res = await fetch('/api/domains' + (domain.id ? `/${domain.id}` : ''), {
        method: domain.id?'PUT':'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(domain),
    });
    return await handleApiResponse<Domain>(res);
}

export async function deleteDomain(id: string): Promise<boolean> {
    const res = await fetch(`/api/domains/${id}`, {
        method: 'DELETE',
        headers: {'Accept': 'application/json'},
    });
    return await handleEmptyApiResponse(res);
}

export async function getDomainLogs(id: string): Promise<Array<DomainLog>> {
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${id}/logs`, {headers: {'Accept': 'application/json'}});
    return await handleApiResponse<Array<DomainLog>>(res);
}

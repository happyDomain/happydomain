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

import type { dnsRR } from "$lib/dns_rr";
import { handleEmptyApiResponse, handleApiResponse } from "$lib/errors";
import type { Domain } from "$lib/model/domain";
import type { ServiceCombined, ServiceMeta } from "$lib/model/service";
import type { Zone } from "$lib/model/zone";

export async function getService(
    domain: Domain,
    zoneid: string,
    subdomain: string,
    svcid: string,
): Promise<ServiceCombined> {
    const res = await fetch(
        `/api/domains/${encodeURIComponent(domain.id)}/zone/${encodeURIComponent(zoneid)}/${encodeURIComponent(subdomain)}/services/${encodeURIComponent(svcid)}`,
        {
            headers: { Accept: "application/json" },
        },
    );
    return await handleApiResponse<ServiceCombined>(res);
}

// TODO -- behind this line, not worked on
export async function addServiceRecord(domain: Domain, id: string, service: ServiceCombined, record: dnsRR): Promise<Zone> {
    let subdomain = service._domain;
    if (subdomain === '') subdomain = '@';

    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    subdomain = encodeURIComponent(subdomain);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/${subdomain}/services`, {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(service)
    });
    return await handleApiResponse<Zone>(res);
}

export async function updateServiceRecord(domain: Domain, id: string, service: ServiceCombined, record: dnsRR): Promise<Zone> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}`, {
        method: 'PATCH',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(service),
    });
    return await handleApiResponse<Zone>(res);
}

export async function deleteServiceRecord(domain: Domain, id: string, service: ServiceMeta, record: dnsRR): Promise<Zone> {
    let subdomain = service._domain;
    if (subdomain === '') subdomain = '@';

    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    subdomain = encodeURIComponent(subdomain);
    const svcid = service._id?encodeURIComponent(service._id):undefined;

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/${subdomain}/services/${svcid}`, {
        method: 'DELETE',
        headers: {'Accept': 'application/json'}
    });
    return await handleApiResponse<Zone>(res);
}

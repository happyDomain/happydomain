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
import { printRR } from "$lib/dns";
import type { dnsRR } from "$lib/dns_rr";
import type { Correction } from "$lib/model/correction";
import type { Domain } from "$lib/model/domain";
import type { ServiceCombined, ServiceMeta } from "$lib/model/service";
import type { ServiceRecord, Zone, ZoneMeta } from "$lib/model/zone";

export async function getZone(domain: Domain, id: string): Promise<Zone> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${dnid}/zone/${id}`, {
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<Zone>(res);
}

export async function viewZone(domain: Domain, id: string): Promise<string> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${dnid}/zone/${id}/view`, {
        method: "POST",
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<string>(res);
}

export async function retrieveZone(domain: Domain): Promise<ZoneMeta> {
    const dnid = encodeURIComponent(domain.id);
    const res = await fetch(`/api/domains/${dnid}/retrieve_zone`, {
        method: "POST",
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<ZoneMeta>(res);
}

export async function applyZone(
    domain: Domain,
    id: string,
    wantedCorrections: Array<string>,
    commitMessage: string,
): Promise<ZoneMeta> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${dnid}/zone/${id}/apply_changes`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify({ wantedCorrections, commitMessage }),
    });
    return await handleApiResponse<ZoneMeta>(res);
}

export async function importZone(domain: Domain, id: string, file: any): Promise<ZoneMeta> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);

    const formData = new FormData();
    formData.append("zone", file);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/import`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: formData,
    });
    return await handleApiResponse<ZoneMeta>(res);
}

export async function diffZone(
    domain: Domain,
    id1: string,
    id2: string,
): Promise<Array<Correction>> {
    const dnid = encodeURIComponent(domain.id);
    id1 = encodeURIComponent(id1);
    id2 = encodeURIComponent(id2);
    const res = await fetch(`/api/domains/${dnid}/zone/${id2}/diff/${id1}`, {
        method: "POST",
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<Array<Correction>>(res);
}

export async function addZoneService(
    domain: Domain,
    id: string,
    service: ServiceCombined,
): Promise<Zone> {
    let subdomain = service._domain;
    if (subdomain === "") subdomain = "@";

    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    subdomain = encodeURIComponent(subdomain);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/${subdomain}/services`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify(service),
    });
    return await handleApiResponse<Zone>(res);
}

export async function updateZoneService(
    domain: Domain,
    id: string,
    service: ServiceCombined,
): Promise<Zone> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}`, {
        method: "PATCH",
        headers: { Accept: "application/json" },
        body: JSON.stringify(service),
    });
    return await handleApiResponse<Zone>(res);
}

export async function deleteZoneService(
    domain: Domain,
    id: string,
    service: ServiceMeta,
): Promise<Zone> {
    let subdomain = service._domain;
    if (subdomain === "") subdomain = "@";

    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    subdomain = encodeURIComponent(subdomain);
    const svcid = service._id ? encodeURIComponent(service._id) : undefined;

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/${subdomain}/services/${svcid}`, {
        method: "DELETE",
        headers: { Accept: "application/json" },
    });
    return await handleApiResponse<Zone>(res);
}

export async function addZoneRecord(
    domain: Domain,
    id: string,
    subdomain: string,
    record: dnsRR,
): Promise<Zone> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/records`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify([printRR(record, subdomain)]),
    });
    return await handleApiResponse<Zone>(res);
}

export async function deleteZoneRecord(
    domain: Domain,
    id: string,
    subdomain: string,
    record: dnsRR,
): Promise<Zone> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/records/delete`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify([printRR(record, subdomain)]),
    });
    return await handleApiResponse<Zone>(res);
}

export async function updateZoneRecord(
    domain: Domain,
    id: string,
    subdomain: string,
    newrr: dnsRR,
    oldrr: dnsRR,
): Promise<Zone> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/records`, {
        method: "PATCH",
        headers: { Accept: "application/json" },
        body: JSON.stringify({oldrr: printRR(oldrr, subdomain), newrr: printRR(newrr, subdomain)}),
    });
    return await handleApiResponse<Zone>(res);
}

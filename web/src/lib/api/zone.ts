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
    getDomainsByDomainIdZoneByZoneId,
    postDomainsByDomainIdZoneByZoneIdView,
    postDomainsByDomainIdRetrieveZone,
    postDomainsByDomainIdZoneByZoneIdApplyChanges,
    postDomainsByDomainIdZone,
    postDomainsByDomainIdZoneByZoneIdDiffByOldZoneId,
    postDomainsByDomainIdZoneByZoneIdBySubdomainServices,
    patchDomainsByDomainIdZoneByZoneId,
    deleteDomainsByDomainIdZoneByZoneIdBySubdomainServicesByServiceId,
    postDomainsByDomainIdZoneByZoneIdRecords,
    postDomainsByDomainIdZoneByZoneIdRecordsDelete,
    patchDomainsByDomainIdZoneByZoneIdRecords,
} from "$lib/api-base/sdk.gen";
import { printRR } from "$lib/dns";
import type { dnsRR } from "$lib/dns_rr";
import type { Correction } from "$lib/model/correction";
import type { Domain } from "$lib/model/domain";
import type { ServiceCombined, ServiceMeta } from "$lib/model/service.svelte";
import type { Zone, ZoneMeta } from "$lib/model/zone";
import { unwrapSdkResponse } from "./errors";

export async function getZone(domain: Domain, id: string): Promise<Zone> {
    return unwrapSdkResponse(
        await getDomainsByDomainIdZoneByZoneId({
            path: { domainId: domain.id, zoneId: id },
        }),
    ) as unknown as Zone;
}

export async function viewZone(domain: Domain, id: string): Promise<string> {
    return unwrapSdkResponse(
        await postDomainsByDomainIdZoneByZoneIdView({
            path: { domainId: domain.id, zoneId: id },
        }),
    ) as string;
}

export async function retrieveZone(domain: Domain): Promise<ZoneMeta> {
    return unwrapSdkResponse(
        await postDomainsByDomainIdRetrieveZone({
            path: { domainId: domain.id },
        }),
    ) as unknown as ZoneMeta;
}

export async function applyZone(
    domain: Domain,
    id: string,
    wantedCorrections: Array<string>,
    commitMessage: string,
): Promise<ZoneMeta> {
    return unwrapSdkResponse(
        await postDomainsByDomainIdZoneByZoneIdApplyChanges({
            path: { domainId: domain.id, zoneId: id },
            body: { wantedCorrections, commitMessage } as any,
        }),
    ) as unknown as ZoneMeta;
}

export async function importZone(domain: Domain, id: string, file: any): Promise<ZoneMeta> {
    const formData = new FormData();
    formData.append("zone", file);

    return unwrapSdkResponse(
        await postDomainsByDomainIdZone({
            path: { domainId: domain.id },
            body: formData as any,
        }),
    ) as unknown as ZoneMeta;
}

export async function diffZone(
    domain: Domain,
    id1: string,
    id2: string,
): Promise<Array<Correction>> {
    return unwrapSdkResponse(
        await postDomainsByDomainIdZoneByZoneIdDiffByOldZoneId({
            path: { domainId: domain.id, zoneId: id2, oldZoneId: id1 },
        }),
    ) as Array<Correction>;
}

export async function addZoneService(
    domain: Domain,
    id: string,
    service: ServiceCombined,
): Promise<Zone> {
    let subdomain = service._domain;
    if (subdomain === "") subdomain = "@";

    return unwrapSdkResponse(
        await postDomainsByDomainIdZoneByZoneIdBySubdomainServices({
            path: { domainId: domain.id, zoneId: id, subdomain },
            body: service as any,
        }),
    ) as unknown as Zone;
}

export async function updateZoneService(
    domain: Domain,
    id: string,
    service: ServiceCombined,
): Promise<Zone> {
    return unwrapSdkResponse(
        await patchDomainsByDomainIdZoneByZoneId({
            path: { domainId: domain.id, zoneId: id },
            body: service as any,
        }),
    ) as unknown as Zone;
}

export async function deleteZoneService(
    domain: Domain,
    id: string,
    service: ServiceMeta,
): Promise<Zone> {
    let subdomain = service._domain;
    if (subdomain === "") subdomain = "@";

    const svcid = service._id || "";

    return unwrapSdkResponse(
        await deleteDomainsByDomainIdZoneByZoneIdBySubdomainServicesByServiceId({
            path: { domainId: domain.id, zoneId: id, subdomain, serviceId: svcid },
        }),
    ) as unknown as Zone;
}

export async function addZoneRecord(
    domain: Domain,
    id: string,
    subdomain: string,
    record: dnsRR,
): Promise<Zone> {
    return unwrapSdkResponse(
        await postDomainsByDomainIdZoneByZoneIdRecords({
            path: { domainId: domain.id, zoneId: id },
            body: [printRR(record, subdomain)] as any,
        }),
    ) as unknown as Zone;
}

export async function deleteZoneRecord(
    domain: Domain,
    id: string,
    subdomain: string,
    record: dnsRR,
): Promise<Zone> {
    return unwrapSdkResponse(
        await postDomainsByDomainIdZoneByZoneIdRecordsDelete({
            path: { domainId: domain.id, zoneId: id },
            body: [printRR(record, subdomain)] as any,
        }),
    ) as unknown as Zone;
}

export async function updateZoneRecord(
    domain: Domain,
    id: string,
    subdomain: string,
    newrr: dnsRR,
    oldrr: dnsRR,
): Promise<Zone> {
    return unwrapSdkResponse(
        await patchDomainsByDomainIdZoneByZoneIdRecords({
            path: { domainId: domain.id, zoneId: id },
            body: { oldrr: printRR(oldrr, subdomain), newrr: printRR(newrr, subdomain) } as any,
        }),
    ) as unknown as Zone;
}

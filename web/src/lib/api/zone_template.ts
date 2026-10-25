// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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
    deleteTemplatesByTemplateId,
    deleteTemplatesByTemplateIdBySubdomainServicesByServiceId,
    getTemplates,
    getTemplatesByTemplateId,
    patchTemplatesByTemplateIdRecords,
    postDomainsByDomainIdZoneByZoneIdApplyTemplate,
    postTemplates,
    postTemplatesByTemplateIdBySubdomainServices,
    postTemplatesByTemplateIdRecords,
    postTemplatesByTemplateIdRecordsDelete,
    postTemplatesImport,
    putTemplatesByTemplateId,
    putTemplatesByTemplateIdBySubdomainServicesByServiceId,
} from "$lib/api-base/sdk.gen";
import type { HappydnsService } from "$lib/api-base/types.gen";
import { printRR } from "$lib/dns";
import type { dnsRR } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import type { Zone } from "$lib/model/zone";
import type {
    ApplyZoneTemplateForm,
    ZoneTemplate,
    ZoneTemplateCreationForm,
    ZoneTemplateImportInput,
    ZoneTemplateInput,
} from "$lib/model/zone_template";
import { unwrapEmptyResponse, unwrapSdkResponse } from "./errors";

export async function listZoneTemplates(): Promise<ZoneTemplate[]> {
    return unwrapSdkResponse(await getTemplates()) as ZoneTemplate[];
}

export async function getZoneTemplate(id: string): Promise<ZoneTemplate> {
    return unwrapSdkResponse(
        await getTemplatesByTemplateId({ path: { templateId: id } }),
    ) as ZoneTemplate;
}

/** Create an empty template, or one from a zone of a domain. */
export async function createZoneTemplate(form: ZoneTemplateCreationForm): Promise<ZoneTemplate> {
    return unwrapSdkResponse(await postTemplates({ body: form })) as ZoneTemplate;
}

export async function importZoneTemplate(
    input: ZoneTemplateImportInput,
    file: File | Blob,
): Promise<ZoneTemplate> {
    return unwrapSdkResponse(
        await postTemplatesImport({
            body: {
                name: input.name ?? "",
                description: input.description,
                origin: input.origin,
                zone: file,
            },
        }),
    ) as ZoneTemplate;
}

export async function updateZoneTemplate(
    id: string,
    input: ZoneTemplateInput,
): Promise<ZoneTemplate> {
    return unwrapSdkResponse(
        await putTemplatesByTemplateId({ path: { templateId: id }, body: input }),
    ) as ZoneTemplate;
}

export async function deleteZoneTemplate(id: string): Promise<void> {
    unwrapEmptyResponse(await deleteTemplatesByTemplateId({ path: { templateId: id } }));
}

function subdomainParam(service: HappydnsService): string {
    return service._domain === "" ? "@" : service._domain;
}

export async function addZoneTemplateService(
    id: string,
    service: HappydnsService,
): Promise<ZoneTemplate> {
    return unwrapSdkResponse(
        await postTemplatesByTemplateIdBySubdomainServices({
            path: { templateId: id, subdomain: subdomainParam(service) },
            body: service,
        }),
    ) as ZoneTemplate;
}

export async function updateZoneTemplateService(
    id: string,
    service: HappydnsService,
): Promise<ZoneTemplate> {
    return unwrapSdkResponse(
        await putTemplatesByTemplateIdBySubdomainServicesByServiceId({
            path: {
                templateId: id,
                subdomain: subdomainParam(service),
                serviceId: service._id || "",
            },
            body: service,
        }),
    ) as ZoneTemplate;
}

export async function deleteZoneTemplateService(
    id: string,
    service: HappydnsService,
): Promise<ZoneTemplate> {
    return unwrapSdkResponse(
        await deleteTemplatesByTemplateIdBySubdomainServicesByServiceId({
            path: {
                templateId: id,
                subdomain: subdomainParam(service),
                serviceId: service._id || "",
            },
        }),
    ) as ZoneTemplate;
}

export async function addZoneTemplateRecord(
    id: string,
    subdomain: string,
    record: dnsRR,
): Promise<ZoneTemplate> {
    return unwrapSdkResponse(
        await postTemplatesByTemplateIdRecords({
            path: { templateId: id },
            body: [printRR(record, subdomain)],
        }),
    ) as ZoneTemplate;
}

export async function deleteZoneTemplateRecord(
    id: string,
    subdomain: string,
    record: dnsRR,
): Promise<ZoneTemplate> {
    return unwrapSdkResponse(
        await postTemplatesByTemplateIdRecordsDelete({
            path: { templateId: id },
            body: [printRR(record, subdomain)],
        }),
    ) as ZoneTemplate;
}

export async function updateZoneTemplateRecord(
    id: string,
    subdomain: string,
    newrr: dnsRR,
    oldrr: dnsRR,
): Promise<ZoneTemplate> {
    return unwrapSdkResponse(
        await patchTemplatesByTemplateIdRecords({
            path: { templateId: id },
            body: { oldrr: printRR(oldrr, subdomain), newrr: printRR(newrr, subdomain) },
        }),
    ) as ZoneTemplate;
}

/**
 * Apply a template to a zone of a domain. The zone given back is a new one
 * when the zone was already published.
 */
export async function applyZoneTemplate(
    domain: Domain,
    zoneId: string,
    form: ApplyZoneTemplateForm,
): Promise<Zone> {
    return unwrapSdkResponse(
        await postDomainsByDomainIdZoneByZoneIdApplyTemplate({
            path: { domainId: domain.id, zoneId },
            body: form,
        }),
    ) as Zone;
}

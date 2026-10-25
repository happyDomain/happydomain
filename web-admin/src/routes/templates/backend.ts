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
    postTemplates,
    postTemplatesByTemplateIdBySubdomainServices,
    postTemplatesByTemplateIdRecords,
    postTemplatesByTemplateIdRecordsDelete,
    postTemplatesImport,
    putTemplatesByTemplateId,
    putTemplatesByTemplateIdBySubdomainServicesByServiceId,
} from "$lib/api-admin";
import type {
    HappydnsService as AdminService,
    HappydnsServiceWritable as AdminServiceWritable,
    HappydnsZoneTemplate as AdminZoneTemplate,
} from "$lib/api-admin";
import type { HappydnsService } from "$lib/api-base/types.gen";
import type { ZoneTemplateBackend } from "$lib/components/zone_templates/backend";
import { printRR } from "$lib/dns";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import type { ZoneTemplate, ZoneTemplateImportInput } from "$lib/model/zone_template";

interface Response<T> {
    data?: T;
    error?: unknown;
}

function unwrap<T>(res: Response<T>): T {
    if (res.error !== undefined) {
        const msg = (res.error as { errmsg?: string } | null)?.errmsg;
        throw new Error(msg || String(res.error));
    }
    return res.data as T;
}

// The administrative API has no transformer: it gives the dates as they are
// sent, as strings, and takes them so.

function fromAdminService(s: AdminService): ServiceWithValue {
    return {
        ...s,
        Service: (s.Service as Record<string, unknown> | undefined) ?? {},
        _propagated_at: s._propagated_at ? new Date(s._propagated_at) : undefined,
    };
}

function fromAdmin(t: AdminZoneTemplate): ZoneTemplate {
    const services: ZoneTemplate["services"] = {};
    for (const [dn, svcs] of Object.entries(t.services ?? {})) {
        services[dn] = svcs.map(fromAdminService);
    }
    return { ...t, last_modified: new Date(t.last_modified), services };
}

async function template(res: Promise<Response<AdminZoneTemplate>>): Promise<ZoneTemplate> {
    return fromAdmin(unwrap(await res));
}

function toAdmin(s: HappydnsService): AdminServiceWritable {
    // A new service is a ServiceCombined, whose fields are accessors a spread
    // does not copy: its toJSON gives them.
    const plain = (s as { toJSON?: () => HappydnsService }).toJSON?.() ?? s;
    return { ...plain, _propagated_at: plain._propagated_at?.toISOString() };
}

function subdomainParam(subdomain: string): string {
    return subdomain === "" ? "@" : subdomain;
}

/** The global templates, through the administrative API. */
export const adminTemplateBackend: ZoneTemplateBackend = {
    list: () => listGlobalTemplates(),
    get: (id) => template(getTemplatesByTemplateId({ path: { templateId: id } })),
    update: (id, input) =>
        template(putTemplatesByTemplateId({ path: { templateId: id }, body: input })),
    delete: async (id) => {
        unwrap(await deleteTemplatesByTemplateId({ path: { templateId: id } }));
    },

    addService: (id, service) =>
        template(
            postTemplatesByTemplateIdBySubdomainServices({
                path: { templateId: id, subdomain: subdomainParam(service._domain) },
                body: toAdmin(service),
            }),
        ),
    updateService: (id, service) =>
        template(
            putTemplatesByTemplateIdBySubdomainServicesByServiceId({
                path: {
                    templateId: id,
                    subdomain: subdomainParam(service._domain),
                    serviceId: service._id || "",
                },
                body: toAdmin(service),
            }),
        ),
    deleteService: (id, service) =>
        template(
            deleteTemplatesByTemplateIdBySubdomainServicesByServiceId({
                path: {
                    templateId: id,
                    subdomain: subdomainParam(service._domain),
                    serviceId: service._id || "",
                },
            }),
        ),

    addRecord: (id, subdomain, record) =>
        template(
            postTemplatesByTemplateIdRecords({
                path: { templateId: id },
                body: [printRR(record, subdomain)],
            }),
        ),
    updateRecord: (id, subdomain, newrr, oldrr) =>
        template(
            patchTemplatesByTemplateIdRecords({
                path: { templateId: id },
                body: { oldrr: printRR(oldrr, subdomain), newrr: printRR(newrr, subdomain) },
            }),
        ),
    deleteRecord: (id, subdomain, record) =>
        template(
            postTemplatesByTemplateIdRecordsDelete({
                path: { templateId: id },
                body: [printRR(record, subdomain)],
            }),
        ),
};

export async function listGlobalTemplates(): Promise<ZoneTemplate[]> {
    return unwrap(await getTemplates()).map(fromAdmin);
}

/** Create a global template: empty, or from the zone file given. */
export function createGlobalTemplate(
    input: ZoneTemplateImportInput,
    zoneFile?: Blob,
): Promise<ZoneTemplate> {
    if (zoneFile) {
        return template(
            postTemplatesImport({
                body: {
                    name: input.name ?? "",
                    description: input.description,
                    origin: input.origin,
                    zone: zoneFile,
                },
            }),
        );
    }
    return template(postTemplates({ body: input }));
}

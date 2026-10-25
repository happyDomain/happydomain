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
    addZoneTemplateRecord,
    addZoneTemplateService,
    deleteZoneTemplate,
    deleteZoneTemplateRecord,
    deleteZoneTemplateService,
    getZoneTemplate,
    listZoneTemplates,
    updateZoneTemplate,
    updateZoneTemplateRecord,
    updateZoneTemplateService,
} from "$lib/api/zone_template";
import type { HappydnsService } from "$lib/api-base/types.gen";
import type { dnsRR } from "$lib/dns_rr";
import type { ZoneTemplate, ZoneTemplateInput } from "$lib/model/zone_template";

/**
 * Where the templates edited are kept: the API of the users for their own
 * templates, the administrative API for the global ones. Each change gives
 * the template saved.
 */
export interface ZoneTemplateBackend {
    /** The templates one can switch to. */
    list(): Promise<ZoneTemplate[]>;
    get(id: string): Promise<ZoneTemplate>;
    update(id: string, input: ZoneTemplateInput): Promise<ZoneTemplate>;
    delete(id: string): Promise<void>;

    addService(id: string, service: HappydnsService): Promise<ZoneTemplate>;
    updateService(id: string, service: HappydnsService): Promise<ZoneTemplate>;
    deleteService(id: string, service: HappydnsService): Promise<ZoneTemplate>;

    addRecord(id: string, subdomain: string, record: dnsRR): Promise<ZoneTemplate>;
    updateRecord(id: string, subdomain: string, newrr: dnsRR, oldrr: dnsRR): Promise<ZoneTemplate>;
    deleteRecord(id: string, subdomain: string, record: dnsRR): Promise<ZoneTemplate>;
}

/** The templates of the user logged in. */
export const userTemplateBackend: ZoneTemplateBackend = {
    list: listZoneTemplates,
    get: getZoneTemplate,
    update: updateZoneTemplate,
    delete: deleteZoneTemplate,

    addService: addZoneTemplateService,
    updateService: updateZoneTemplateService,
    deleteService: deleteZoneTemplateService,

    addRecord: addZoneTemplateRecord,
    updateRecord: updateZoneTemplateRecord,
    deleteRecord: deleteZoneTemplateRecord,
};

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

import type {
    HappydnsApplyZoneTemplateForm,
    HappydnsZoneTemplate,
    HappydnsZoneTemplateApplyMode,
    HappydnsZoneTemplateCreationForm,
    HappydnsZoneTemplateInput,
} from "$lib/api-base/types.gen";
import type { Domain } from "$lib/model/domain";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import type { Zone } from "$lib/model/zone";

export interface ZoneTemplate extends Omit<HappydnsZoneTemplate, "services"> {
    services: Record<string, Array<ServiceWithValue>>;
}

export type ZoneTemplateInput = HappydnsZoneTemplateInput;
/** What describes a template made from a zone file, and the domain the file is written for. */
export type ZoneTemplateImportInput = ZoneTemplateInput & { origin?: string };
export type ZoneTemplateCreationForm = HappydnsZoneTemplateCreationForm;
export type ZoneTemplateApplyMode = HappydnsZoneTemplateApplyMode;
export type ApplyZoneTemplateForm = HappydnsApplyZoneTemplateForm;

/** A template without owner is offered to every user by the administrators. */
export function isGlobalTemplate(t: Pick<ZoneTemplate, "userId">): boolean {
    return !t.userId;
}

/** The number of services of a template. */
export function countServices(t: Pick<ZoneTemplate, "services">): number {
    return Object.values(t.services ?? {}).reduce((n, svcs) => n + (svcs?.length ?? 0), 0);
}

/**
 * The template as a zone, for the components showing and editing zones. It
 * keeps the identifier of the template.
 */
export function templateAsZone(t: ZoneTemplate): Zone {
    return {
        id: t.id,
        id_author: t.userId ?? "",
        default_ttl: t.default_ttl,
        last_modified: t.last_modified,
        services: t.services ?? {},
    };
}

/**
 * The domain the services of every template are shown under, and the records
 * given as text are relative to. Being reserved, no record of a real zone
 * points to it.
 */
export const TEMPLATE_ORIGIN = "example.com.";

/**
 * The origin of the templates as a domain, for the components showing and
 * editing zones. It has neither provider nor history: what can be added does
 * not depend on a provider, and there are no checks.
 */
export function templateOrigin(): Domain {
    return {
        id: "",
        id_owner: "",
        id_provider: "",
        domain: TEMPLATE_ORIGIN,
        zone_history: [],
    };
}

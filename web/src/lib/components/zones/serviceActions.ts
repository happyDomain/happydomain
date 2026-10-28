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

import { get } from "svelte/store";

import { invalidateAll } from "$app/navigation";

import type { ZoneEditor } from "./editor";
import type { HappydnsService } from "$lib/api-base/types.gen";
import { refreshDomains } from "$lib/stores/domains";
import { thisZone } from "$lib/stores/thiszone";
import { notifyServiceChange, zoneWasForked } from "$lib/stores/zonefeedback";

/** Whether the service may be deleted: the origin of a zone may not. */
export function isDeletable(service: HappydnsService): boolean {
    return (
        !!service._id &&
        service._svctype !== "abstract.Origin" &&
        service._svctype !== "abstract.NSOnlyOrigin"
    );
}

/**
 * Deletes the service from the zone in thisZone. shownZoneId is the zone the
 * page shows: when the deletion derived a new zone, the page is reloaded.
 */
export async function deleteService(
    editor: ZoneEditor,
    service: HappydnsService,
    shownZoneId: string,
): Promise<void> {
    const previousZone = get(thisZone);
    if (!previousZone) return;

    const z = await editor.deleteService(previousZone.id, service);
    notifyServiceChange("deleted", {
        svctype: service._svctype,
        dn: service._domain,
        forked: zoneWasForked(previousZone, z),
    });
    if (z.id !== shownZoneId) {
        refreshDomains().then(() => invalidateAll());
    } else {
        thisZone.set(z);
    }
}

/** Saves the service, as changed in place (its TTL), to the zone in thisZone. */
export async function saveService(editor: ZoneEditor, service: HappydnsService): Promise<void> {
    const previousZone = get(thisZone);
    if (!previousZone) return;

    const z = await editor.updateService(previousZone.id, service);
    thisZone.set(z);
    notifyServiceChange("updated", {
        svctype: service._svctype,
        dn: service._domain,
        serviceId: service._id,
        forked: zoneWasForked(previousZone, z),
    });
}

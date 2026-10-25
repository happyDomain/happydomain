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

import { writable, type Writable } from "svelte/store";

import { listZoneTemplates } from "$lib/api/zone_template";
import type { ZoneTemplate } from "$lib/model/zone_template";

/** The templates of the user, then the global ones; undefined until loaded. */
export const zoneTemplates: Writable<ZoneTemplate[] | undefined> = writable(undefined);

export async function refreshZoneTemplates(): Promise<ZoneTemplate[]> {
    const data = await listZoneTemplates();
    zoneTemplates.set(data);
    return data;
}

/** Add a template to the list, or replace the one with the same identifier. */
export function upsertZoneTemplate(t: ZoneTemplate): void {
    zoneTemplates.update((list) => {
        if (!list) return list;
        const i = list.findIndex((e) => e.id === t.id);
        if (i < 0) return [t, ...list];
        const copy = [...list];
        copy[i] = t;
        return copy;
    });
}

export function removeZoneTemplate(id: string): void {
    zoneTemplates.update((list) => list?.filter((e) => e.id !== id));
}

/** The template being shown or edited, as the server last gave it. */
export const thisTemplate: Writable<ZoneTemplate | null> = writable(null);

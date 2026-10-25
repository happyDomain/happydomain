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
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ listZoneTemplates: vi.fn() }));
vi.mock("$lib/api/zone_template", () => api);

import type { ZoneTemplate } from "$lib/model/zone_template";
import {
    refreshZoneTemplates,
    removeZoneTemplate,
    upsertZoneTemplate,
    zoneTemplates,
} from "./zone_templates";

function tpl(id: string, name: string): ZoneTemplate {
    return {
        id,
        name,
        default_ttl: 3600,
        services: {},
    } as unknown as ZoneTemplate;
}

describe("zone templates store", () => {
    beforeEach(() => {
        vi.resetAllMocks();
        zoneTemplates.set(undefined);
    });

    it("loads the templates", async () => {
        api.listZoneTemplates.mockResolvedValue([tpl("1", "Mail")]);

        await refreshZoneTemplates();
        expect(get(zoneTemplates)).toEqual([tpl("1", "Mail")]);
    });

    it("leaves templates not loaded yet alone", () => {
        upsertZoneTemplate(tpl("1", "Mail"));
        removeZoneTemplate("1");
        expect(get(zoneTemplates)).toBeUndefined();
    });

    it("adds a new template first, and replaces a known one in place", () => {
        zoneTemplates.set([tpl("1", "Mail"), tpl("2", "Web")]);

        upsertZoneTemplate(tpl("3", "Chat"));
        upsertZoneTemplate(tpl("1", "Mail servers"));

        expect(get(zoneTemplates)?.map((t) => t.name)).toEqual(["Chat", "Mail servers", "Web"]);
    });

    it("removes a template", () => {
        zoneTemplates.set([tpl("1", "Mail"), tpl("2", "Web")]);

        removeZoneTemplate("1");
        expect(get(zoneTemplates)).toEqual([tpl("2", "Web")]);
    });
});

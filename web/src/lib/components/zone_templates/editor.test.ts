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

import { describe, expect, it, vi } from "vitest";

import type { ZoneTemplate } from "$lib/model/zone_template";
import { makeService } from "$lib/test-utils/fixtures";
import type { ZoneTemplateBackend } from "./backend";
import { templateZoneEditor } from "./editor";

const saved: ZoneTemplate = {
    id: "t1",
    name: "Mail",
    default_ttl: 3600,
    services: { "": [makeService("svcs.MXs", {}, { _id: "s1" })] },
} as unknown as ZoneTemplate;

function backend(): ZoneTemplateBackend {
    const change = vi.fn().mockResolvedValue(saved);
    return {
        list: vi.fn(),
        get: vi.fn(),
        update: vi.fn(),
        delete: vi.fn(),
        addService: change,
        updateService: change,
        deleteService: change,
        addRecord: change,
        updateRecord: change,
        deleteRecord: change,
    };
}

describe("templateZoneEditor", () => {
    it("saves the changes to the template, and shows it as a zone", async () => {
        const b = backend();
        const onSaved = vi.fn();
        const editor = templateZoneEditor(
            () => b,
            () => false,
            onSaved,
        );

        const svc = makeService("svcs.MXs");
        const zone = await editor.addService("t1", svc);

        expect(b.addService).toHaveBeenCalledWith("t1", svc);
        expect(onSaved).toHaveBeenCalledWith(saved);
        expect(zone.id).toBe("t1");
        expect(zone.services[""]).toHaveLength(1);
    });

    it("tells it is not the zone of a domain, and whether it can change", () => {
        let readOnly = true;
        const editor = templateZoneEditor(backend, () => readOnly, vi.fn());

        expect(editor.isDomain).toBe(false);
        expect(editor.readOnly).toBe(true);
        readOnly = false;
        expect(editor.readOnly).toBe(false);
    });

    it("links to the pages of the template", () => {
        const editor = templateZoneEditor(backend, () => false, vi.fn());

        expect(editor.zoneHref("t1")).toBe("/templates/t1");
        expect(editor.serviceHref("t1", "", "new")).toBe("/templates/t1/%40/new");
        expect(editor.serviceHref("t1", "www", "s1")).toBe("/templates/t1/www/s1");
    });
});

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

import { describe, expect, it } from "vitest";

import { makeService } from "$lib/test-utils/fixtures";
import {
    countServices,
    isGlobalTemplate,
    templateAsZone,
    templateOrigin,
    type ZoneTemplate,
} from "./zone_template";

function tpl(overrides: Partial<ZoneTemplate> = {}): ZoneTemplate {
    return {
        id: "t1",
        userId: "u1",
        name: "Mail",
        default_ttl: 3600,
        services: {},
        ...overrides,
    } as ZoneTemplate;
}

describe("countServices", () => {
    it("counts the services of every subdomain", () => {
        const mx = makeService("svcs.MXs");
        expect(countServices(tpl({ services: { "": [mx, mx], www: [mx] } }))).toBe(3);
    });

    it("counts none in a template without services", () => {
        expect(countServices(tpl({ services: { "": [] } }))).toBe(0);
        expect(countServices({ services: undefined } as unknown as ZoneTemplate)).toBe(0);
    });
});

describe("isGlobalTemplate", () => {
    it("tells a template without owner is global", () => {
        expect(isGlobalTemplate(tpl())).toBe(false);
        expect(isGlobalTemplate(tpl({ userId: undefined }))).toBe(true);
        expect(isGlobalTemplate(tpl({ userId: "" }))).toBe(true);
    });
});

describe("templateAsZone", () => {
    it("keeps the identifier and the services of the template", () => {
        const services = { "": [makeService("svcs.MXs")] };
        const zone = templateAsZone(tpl({ services }));

        expect(zone.id).toBe("t1");
        expect(zone.id_author).toBe("u1");
        expect(zone.default_ttl).toBe(3600);
        expect(zone.services).toBe(services);
    });

    it("gives a global template no author", () => {
        const zone = templateAsZone(
            tpl({ userId: undefined, services: undefined } as unknown as ZoneTemplate),
        );

        expect(zone.id_author).toBe("");
        expect(zone.services).toEqual({});
    });
});

describe("templateOrigin", () => {
    it("is the origin of the templates, without provider nor history", () => {
        expect(templateOrigin()).toEqual({
            id: "",
            id_owner: "",
            id_provider: "",
            domain: "example.com.",
            zone_history: [],
        });
    });
});

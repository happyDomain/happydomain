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

import { describe, it, expect } from "vitest";

import { servicesSpecs } from "$lib/services_specs";
import { hasValidators } from "$lib/services/compliance";

// Same discovery as the application: a service folder is named after the
// service type it implements.
const editors = import.meta.glob("./*/editor.svelte");
const validators = import.meta.glob("./*/compliance.ts", { eager: true });
const english = import.meta.glob<{ default: Record<string, any> }>("./*/locales/en.json", {
    eager: true,
});

function folders(paths: Record<string, unknown>): string[] {
    return Object.keys(paths).map((path) => path.split("/").at(-2)!);
}

describe("service folders", () => {
    it("provides at least the editors of the migrated services", () => {
        expect(folders(editors)).toContain("svcs.ForSale");
    });

    it("names every folder after a service type known to the backend", () => {
        const unknown = folders(editors).filter((svctype) => !(svctype in servicesSpecs));
        expect(unknown).toEqual([]);
    });

    it("registers the validators under the name of their folder", () => {
        const unregistered = folders(validators).filter((svctype) => !hasValidators(svctype));
        expect(unregistered).toEqual([]);
    });

    it("names and describes every service in English", () => {
        const said: Record<string, any> = {};
        for (const module of Object.values(english)) {
            const svcinfo = module.default.svcinfo ?? {};
            for (const [family, services] of Object.entries<any>(svcinfo)) {
                for (const [name, infos] of Object.entries(services)) {
                    said[`${family}.${name}`] = infos;
                }
            }
        }

        const undescribed = Object.keys(servicesSpecs).filter(
            (svctype) => !said[svctype]?.name || !said[svctype]?.description,
        );
        expect(undescribed).toEqual([]);
    });
});

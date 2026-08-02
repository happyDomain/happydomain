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

import { config, loadTranslations, t } from "./translations";

async function load(locale: string, key = "") {
    const loader = config.loaders?.find((l) => l.locale === locale && l.key === key);
    expect(loader, `no loader for ${locale}/${key}`).toBeDefined();
    return (await loader!.loader()) as Record<string, any>;
}

describe("translations", () => {
    it("keeps the application translations", async () => {
        const en = await load("en");
        expect(en.compliance.title).toBeTruthy();
        expect(en.common).toBeTruthy();
    });

    it("merges the translations owned by the services", async () => {
        const en = await load("en");
        // Shipped by web/src/lib/services/svcs.ForSale/locales/en.json
        expect(en.resources.FORSALE).toBeTruthy();
        // A service subtree must not shadow its siblings.
        expect(en.compliance.forsale).toBeTruthy();
        expect(en.compliance.spf).toBeTruthy();
        expect(en.compliance.title).toBeTruthy();
    });

    it("merges services translations for every offered locale", async () => {
        const fr = await load("fr");
        expect(fr.compliance.forsale).toBeTruthy();
        expect(fr.resources.CAA).toBeTruthy();
    });

    it("gives the services their translations, and echoes back unknown keys", async () => {
        await loadTranslations("en", "/");
        // What $lib/services/infos.ts relies on to fall back on the name
        // registered by the backend.
        expect(t.get("svcinfo.svcs.NoSuchService.name")).toBe("svcinfo.svcs.NoSuchService.name");
        expect(t.get("svcinfo.svcs.ForSale.name")).toBe("Domain For Sale");
    });

    it("does not mutate the application translations while merging", async () => {
        const first = await load("en");
        const second = await load("en");
        expect(first).not.toBe(second);
        expect(Object.keys(second.compliance).length).toBe(Object.keys(first.compliance).length);
    });
});

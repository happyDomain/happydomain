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

import { ServiceInfos } from "$lib/model/service_specs.svelte";
import { serviceDescriptionOf, serviceNameOf } from "./infos";

// Stands for sveltekit-i18n, which echoes the key back when it is unknown.
function translator(translations: Record<string, string>) {
    return (key: string) => translations[key] ?? key;
}

const forsale = new ServiceInfos({ _svctype: "svcs.ForSale", name: "Domain For Sale" });

describe("service infos", () => {
    it("prefers the translation over the name registered by the backend", () => {
        const t = translator({ "svcinfo.svcs.ForSale.name": "Domaine à vendre" });
        expect(serviceNameOf(t, forsale)).toBe("Domaine à vendre");
    });

    it("falls back on the name registered by the backend", () => {
        expect(serviceNameOf(translator({}), forsale)).toBe("Domain For Sale");
    });

    it("falls back on the service type when nothing else is known", () => {
        const bare = new ServiceInfos({ _svctype: "svcs.Unknown" });
        expect(serviceNameOf(translator({}), bare)).toBe("svcs.Unknown");
        expect(serviceNameOf(translator({}), undefined, "svcs.Unknown")).toBe("svcs.Unknown");
    });

    it("has no description outside of the translations", () => {
        expect(serviceDescriptionOf(translator({}), forsale)).toBe("");
        const t = translator({ "svcinfo.svcs.ForSale.description": "Ce domaine est à vendre." });
        expect(serviceDescriptionOf(t, forsale)).toBe("Ce domaine est à vendre.");
    });
});

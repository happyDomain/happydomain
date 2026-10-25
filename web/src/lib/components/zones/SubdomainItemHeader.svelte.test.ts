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

import { fireEvent, render, screen } from "@testing-library/svelte";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import type { HappydnsService } from "$lib/api-base/types.gen";
import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import type { User } from "$lib/model/user";
import { ZoneViewList } from "$lib/model/usersettings";
import { providers } from "$lib/stores/providers";
import { userSession } from "$lib/stores/usersession";
import { loadTranslations } from "$lib/translations";
import { controls as ctrlServiceDetails } from "./ServiceDetailsOffcanvas.svelte";
import SubdomainItemHeader from "./SubdomainItemHeader.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    providers.set(undefined);
    userSession.set({} as User);
});

function setup(svctype: string, readonly = false) {
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    userSession.set({ settings: { zoneview: ZoneViewList } } as unknown as User);
    return {
        dn: "www",
        origin: { id: "d1", domain: "example.com.", id_provider: "p1" } as unknown as Domain,
        services: [{ _id: "s1", _svctype: svctype, _domain: "www" }] as HappydnsService[],
        zoneId: "z1",
    };
}

describe("SubdomainItemHeader", () => {
    it("offers to add an alias or a service to a regular subdomain", () => {
        render(SubdomainItemHeader, setup("abstract.Server"));
        expect(screen.getByRole("button", { name: /Add an alias/ })).toBeEnabled();
        expect(screen.getByRole("button", { name: /Add a service/ })).toBeEnabled();
        expect(screen.queryByRole("button", { name: /Service details/ })).toBeNull();
    });

    it.each(["svcs.Alias", "svcs.PTR"])(
        "opens the details of a %s instead of editing or dropping it in place",
        async (svctype) => {
            const open = vi.fn();
            ctrlServiceDetails.Open = open;
            const props = setup(svctype);
            render(SubdomainItemHeader, props);

            expect(screen.queryByRole("button", { name: /Edit target/ })).toBeNull();
            expect(screen.queryByRole("button", { name: /Drop/ })).toBeNull();
            expect(screen.queryByRole("button", { name: /Add an alias/ })).toBeNull();

            await fireEvent.click(screen.getByRole("button", { name: /Service details/ }));
            expect(open).toHaveBeenCalledWith(props.services[0]);
        },
    );

    it("hides every change of a regular subdomain through a read-only provider", () => {
        render(SubdomainItemHeader, setup("abstract.Server", true));
        expect(screen.queryByRole("button", { name: /Add an alias/ })).toBeNull();
        expect(screen.queryByRole("button", { name: /Add a service/ })).toBeNull();
    });

    it("still offers the details of an alias through a read-only provider", () => {
        render(SubdomainItemHeader, setup("svcs.Alias", true));
        expect(screen.getByRole("button", { name: /Service details/ })).toBeEnabled();
    });
});

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

import { render, screen } from "@testing-library/svelte";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import type { HappydnsService } from "$lib/api-base/types.gen";
import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import type { User } from "$lib/model/user";
import { ZoneViewList } from "$lib/model/usersettings";
import { isReadOnlyProvider, providers } from "$lib/stores/providers";
import { userSession } from "$lib/stores/usersession";
import { loadTranslations } from "$lib/translations";
import SubdomainItemHeader from "./SubdomainItemHeader.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    providers.set(undefined);
    userSession.set({} as User);
});

function setup(readonly: boolean) {
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    userSession.set({ settings: { zoneview: ZoneViewList } } as unknown as User);
    return {
        dn: "www",
        origin: { id: "d1", domain: "example.com.", id_provider: "p1" } as unknown as Domain,
        services: [{ _id: "s1", _svctype: "abstract.Server", _domain: "www" }] as HappydnsService[],
        zoneId: "z1",
    };
}

describe("isReadOnlyProvider", () => {
    const idx = {
        ro: { _id: "ro", _readonly: true } as unknown as Provider,
        rw: { _id: "rw" } as unknown as Provider,
    };

    it("tells read-only providers from the others", () => {
        expect(isReadOnlyProvider(idx, "ro")).toBe(true);
        expect(isReadOnlyProvider(idx, "rw")).toBe(false);
    });

    it("does not take an unknown provider for a read-only one", () => {
        expect(isReadOnlyProvider(idx, "missing")).toBe(false);
        expect(isReadOnlyProvider(idx, undefined)).toBe(false);
    });
});

describe("SubdomainItemHeader", () => {
    it("offers to change the subdomain through a writable provider", () => {
        render(SubdomainItemHeader, setup(false));
        expect(screen.getByRole("button", { name: /Add an alias/ })).toBeEnabled();
        expect(screen.getByRole("button", { name: /Add a service/ })).toBeEnabled();
    });

    it("disables every change through a read-only provider, and tells why", () => {
        render(SubdomainItemHeader, setup(true));
        const alias = screen.getByRole("button", { name: /Add an alias/ });
        expect(alias).toBeDisabled();
        expect(screen.getByRole("button", { name: /Add a service/ })).toBeDisabled();
        expect(alias.parentElement).toHaveAttribute(
            "title",
            "This provider is read-only: happyDomain cannot change this zone.",
        );
    });
});

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

import { cleanup, fireEvent, render, screen } from "@testing-library/svelte";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

// The network lives here: the tests decide what the backend answers.
const api = vi.hoisted(() => ({ addZoneService: vi.fn(), updateZoneService: vi.fn() }));
vi.mock("$lib/api/zone", () => api);

import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import type { Zone } from "$lib/model/zone";
import { providers } from "$lib/stores/providers";
import { thisZone } from "$lib/stores/thiszone";
import { makeZone } from "$lib/test-utils/fixtures";
import { loadTranslations } from "$lib/translations";
import Page from "./+page.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

beforeEach(() => {
    Object.values(api).forEach((f) => f.mockReset());
});

afterEach(() => {
    cleanup();
    providers.set(undefined);
    thisZone.set(null);
});

function setup(readonly: boolean) {
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    thisZone.set(
        makeZone({
            id: "z1",
            services: { www: [{ _id: "s1", _svctype: "svcs.Alias", _domain: "www", Service: {} }] },
        } as unknown as Partial<Zone>),
    );
    return {
        data: {
            domain: { id: "d1", domain: "example.com.", id_provider: "p1" } as unknown as Domain,
            history: "z1",
            zoneId: "z1",
            subdomain: "www",
            serviceid: "s1",
        },
    };
}

describe("service page", () => {
    it("offers to save the service through a writable provider", () => {
        render(Page, setup(false));
        expect(screen.getByRole("button", { name: /Update/ })).toBeEnabled();
    });

    it("shows the service without offering to save it through a read-only provider", () => {
        render(Page, setup(true));
        expect(screen.getByRole("link", { name: /check/i })).toBeTruthy();
        expect(screen.queryByRole("button", { name: /Update/ })).toBeNull();
    });

    it("does not send the service when the form is submitted through a read-only provider", async () => {
        render(Page, setup(true));

        await fireEvent.submit(document.getElementById("addSvcForm") as HTMLFormElement);

        expect(api.updateZoneService).not.toHaveBeenCalled();
        expect(api.addZoneService).not.toHaveBeenCalled();
    });
});

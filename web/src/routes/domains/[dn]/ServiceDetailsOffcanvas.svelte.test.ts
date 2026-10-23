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

import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import { providers } from "$lib/stores/providers";
import { thisZone } from "$lib/stores/thiszone";
import { makeZone } from "$lib/test-utils/fixtures";
import { loadTranslations } from "$lib/translations";
import ServiceDetailsOffcanvas, { controls } from "./ServiceDetailsOffcanvas.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    cleanup();
    providers.set(undefined);
    thisZone.set(null);
});

function open(readonly: boolean) {
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    thisZone.set(makeZone({ id: "z1" }));
    render(ServiceDetailsOffcanvas, {
        domain: { id: "d1", domain: "example.com.", id_provider: "p1" } as unknown as Domain,
        selectedHistory: "z1",
    });
    controls.Open({
        _id: "s1",
        _svctype: "svcs.Alias",
        _domain: "www",
        _ttl: 300,
    } as unknown as ServiceWithValue);
}

describe("ServiceDetailsOffcanvas", () => {
    it("offers to edit and to delete the service through a writable provider", async () => {
        open(false);
        expect(await screen.findByRole("button", { name: /Edit/ })).toBeEnabled();
        expect(screen.getByRole("button", { name: /Delete/ })).toBeEnabled();
        expect(screen.getByRole("spinbutton")).toBeEnabled();
    });

    it("keeps the details but changes nothing through a read-only provider", async () => {
        open(true);
        const edit = await screen.findByRole("button", { name: /Edit/ });
        expect(edit).toBeDisabled();
        expect(edit.parentElement).toHaveAttribute("title", expect.stringMatching(/read-only/));
        expect(screen.queryByRole("button", { name: /Delete/ })).toBeNull();
        expect(screen.getByRole("spinbutton")).toBeDisabled();
    });
});

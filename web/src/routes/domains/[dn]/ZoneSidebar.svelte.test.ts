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
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import { domains } from "$lib/stores/domains";
import { providers } from "$lib/stores/providers";
import { thisZone } from "$lib/stores/thiszone";
import { makeZone } from "$lib/test-utils/fixtures";
import { loadTranslations } from "$lib/translations";
import ZoneSidebar from "./ZoneSidebar.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
    // jsdom has none, and the list of subdomains watches which one is in view.
    vi.stubGlobal(
        "IntersectionObserver",
        class {
            observe() {}
            unobserve() {}
            disconnect() {}
        },
    );
});

afterEach(() => {
    cleanup();
    domains.set(undefined);
    providers.set(undefined);
    thisZone.set(null);
});

function setup(readonly: boolean) {
    const origin = {
        id: "d1",
        domain: "example.com.",
        id_provider: "p1",
        zone_history: ["z1"],
    } as unknown as Domain;
    domains.set([origin]);
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    thisZone.set(makeZone({ id: "z1" }));
    return { origin, selectedDomain: "d1", selectedHistory: "z1", onretrieveZoneDone: () => {} };
}

describe("ZoneSidebar", () => {
    it("offers to add a subdomain, to import a zone, to save it and to apply a template through a writable provider", async () => {
        render(ZoneSidebar, setup(false));
        expect(screen.getByRole("button", { name: /Add a subdomain/ })).toBeEnabled();

        await userEvent.setup().click(screen.getByRole("button", { name: /More actions/ }));
        expect(await screen.findByText("Import a zone file")).not.toHaveClass("disabled");
        expect(screen.getByText("Apply a template")).not.toHaveClass("disabled");
        expect(screen.getByText("Save as a template")).not.toHaveClass("disabled");
    });

    it("disables the changes to the zone, telling why, through a read-only provider", async () => {
        render(ZoneSidebar, setup(true));
        const add = screen.getByRole("button", { name: /Add a subdomain/ });
        expect(add).toBeDisabled();
        expect(add.parentElement).toHaveAttribute("title", expect.stringMatching(/read-only/));

        await userEvent.setup().click(screen.getByRole("button", { name: /More actions/ }));
        expect(await screen.findByText("Import a zone file")).toHaveClass("disabled");
        expect(screen.getByText("Apply a template")).toHaveClass("disabled");
        expect(screen.getByText("Apply a template")).toHaveAttribute(
            "title",
            expect.stringMatching(/read-only/),
        );
        // Saving the zone as a template does not change it.
        expect(screen.getByText("Save as a template")).not.toHaveClass("disabled");
    });
});

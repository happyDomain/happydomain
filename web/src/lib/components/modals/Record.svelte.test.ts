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
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

// The network lives here: the tests decide what the backend answers.
const api = vi.hoisted(() => ({
    addZoneRecord: vi.fn(),
    deleteZoneRecord: vi.fn(),
    updateZoneRecord: vi.fn(),
}));
vi.mock("$lib/api/zone", () => api);

import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import { providers } from "$lib/stores/providers";
import { makeZone } from "$lib/test-utils/fixtures";
import { loadTranslations } from "$lib/translations";
import Record, { controls } from "./Record.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

beforeEach(() => {
    Object.values(api).forEach((f) => f.mockReset());
});

afterEach(() => {
    providers.set(undefined);
});

function open(readonly: boolean) {
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    render(Record, {
        origin: { id: "d1", domain: "example.com.", id_provider: "p1" } as unknown as Domain,
        zone: makeZone(),
    });
    controls.Open(
        {
            Hdr: { Name: "www", Rrtype: 16, Class: 1, Ttl: 300, Rdlength: 0 },
            Txt: "hello",
        } as never,
        "www",
    );
}

describe("Record modal", () => {
    it("offers to save and to delete the record through a writable provider", async () => {
        open(false);
        expect(await screen.findByRole("button", { name: /Update/ })).toBeEnabled();
        expect(screen.getByRole("button", { name: /Cancel/ })).toBeEnabled();
        expect(screen.getByTitle("Delete this record")).toBeTruthy();
    });

    it("only offers to close the record through a read-only provider", async () => {
        open(true);
        expect(await screen.findByRole("button", { name: /Cancel/ })).toBeEnabled();
        expect(screen.queryByRole("button", { name: /Update/ })).toBeNull();
        expect(screen.queryByTitle("Delete this record")).toBeNull();
    });

    it("does not send the record when the form is submitted through a read-only provider", async () => {
        open(true);
        await screen.findByRole("button", { name: /Cancel/ });
        const form = document.getElementById("addRRForm") as HTMLFormElement;

        await fireEvent.submit(form);

        expect(api.updateZoneRecord).not.toHaveBeenCalled();
        expect(api.addZoneRecord).not.toHaveBeenCalled();
    });
});

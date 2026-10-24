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
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

// The network lives here: the tests decide what the backend answers.
vi.mock("$lib/api/zone", () => ({
    diffZone: vi.fn(async () => []),
    diffZoneSummary: vi.fn(async () => ({ nbDiffs: 2 })),
}));
vi.mock("$lib/api/domains", () => ({
    getDomain: vi.fn(async () => ({ id: "d1", zone_meta: {} })),
    listDomains: vi.fn(async () => []),
}));

import { diffZoneSummary } from "$lib/api/zone";
import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import type { Zone } from "$lib/model/zone";
import { domains } from "$lib/stores/domains";
import { providers } from "$lib/stores/providers";
import { thisZone } from "$lib/stores/thiszone";
import { loadTranslations } from "$lib/translations";
import ButtonZonePublish from "./ButtonZonePublish.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    vi.clearAllMocks();
    domains.set(undefined);
    providers.set(undefined);
    thisZone.set(null);
});

function setup(readonly: boolean, partial = false): Domain {
    const domain = {
        id: "d1",
        domain: "example.com.",
        id_provider: "p1",
        zone_history: ["z2", "z1"],
    } as unknown as Domain;
    domains.set([domain as never]);
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    thisZone.set({ id: "z2", partial } as Zone);
    return domain;
}

describe("ButtonZonePublish", () => {
    it("offers to publish the latest zone through a writable provider", async () => {
        render(ButtonZonePublish, { domain: setup(false), history: "z2" });
        // Once the summary of the differences is in.
        await screen.findByText("(2)");
        expect(screen.getByRole("button", { name: /Publish my changes/ })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /View the differences/ })).toBeNull();
    });

    it("only offers to view the differences through a read-only provider", async () => {
        render(ButtonZonePublish, { domain: setup(true), history: "z2" });
        await screen.findByText("(2)");
        expect(screen.getByRole("button", { name: /View the differences/ })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Publish my changes/ })).toBeNull();
    });

    it("offers no rollback of an older zone through a read-only provider", async () => {
        render(ButtonZonePublish, { domain: setup(true), history: "z1" });
        expect(
            await screen.findByRole("button", { name: /View the differences/ }),
        ).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Roll back/ })).toBeNull();
    });

    it("does not compute the differences of a partial zone beforehand", async () => {
        render(ButtonZonePublish, { domain: setup(true, true), history: "z2" });
        expect(
            await screen.findByRole("button", { name: /View the differences/ }),
        ).toBeInTheDocument();
        expect(screen.queryByText("(2)")).toBeNull();
        expect(diffZoneSummary).not.toHaveBeenCalled();
    });
});

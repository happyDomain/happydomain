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
import type { User } from "$lib/model/user";
import { ZoneViewGrid } from "$lib/model/usersettings";
import type { Zone } from "$lib/model/zone";
import { thisZone } from "$lib/stores/thiszone";
import { userSession } from "$lib/stores/usersession";
import { makeZone } from "$lib/test-utils/fixtures";
import { loadTranslations } from "$lib/translations";
import Page from "./+page.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    cleanup();
    thisZone.set(null);
    userSession.set({} as User);
});

function setup(partial: boolean) {
    userSession.set({ settings: { zoneview: ZoneViewGrid } } as unknown as User);
    thisZone.set(makeZone({ id: "z1", partial, services: {} } as unknown as Partial<Zone>));
    return {
        data: {
            domain: { id: "d1", domain: "example.com.", zone_history: ["z1"] } as unknown as Domain,
            history: "z1",
            zoneId: "z1",
        },
    };
}

describe("zone page", () => {
    it("warns the zone may miss names when it is partial", () => {
        render(Page, setup(true));
        expect(screen.getByText(/Partial zone/)).toBeTruthy();
    });

    it("does not warn about a complete zone", () => {
        render(Page, setup(false));
        expect(screen.queryByText(/Partial zone/)).toBeNull();
    });
});

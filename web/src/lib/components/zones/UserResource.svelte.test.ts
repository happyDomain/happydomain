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

import { cleanup, render } from "@testing-library/svelte";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import type { User } from "$lib/model/user";
import { providers } from "$lib/stores/providers";
import { userSession } from "$lib/stores/usersession";
import { loadTranslations } from "$lib/translations";
import UserResource from "./UserResource.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    cleanup();
    providers.set(undefined);
    userSession.set({} as User);
});

function setup(readonly: boolean) {
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    userSession.set({ settings: {} } as unknown as User);
    return {
        dn: "www",
        origin: { id: "d1", domain: "example.com.", id_provider: "p1" } as unknown as Domain,
        services: [],
        zoneId: "z1",
    };
}

describe("UserResource", () => {
    it("proposes a new service through a writable provider", () => {
        const { container } = render(UserResource, setup(false));
        expect(container.querySelector(".new-service-card")).not.toBeNull();
    });

    it("does not propose a new service through a read-only provider", () => {
        const { container } = render(UserResource, setup(true));
        expect(container.querySelector(".new-service-card")).toBeNull();
    });

    // As the session of an administrator, who has no settings.
    it("proposes a new service to a session without settings", () => {
        const props = setup(false);
        userSession.set({} as User);

        const { container } = render(UserResource, props);
        expect(container.querySelector(".new-service-card")).not.toBeNull();
    });
});

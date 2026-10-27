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
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import type { Domain } from "$lib/model/domain";
import type { AbstractNSOnlyOriginBody } from "$lib/services_bodies";
import { loadTranslations } from "$lib/translations";
import Editor from "./editor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

const origin = { domain: "example.com." } as Domain;

describe("NS-only origin editor", () => {
    it("adds a name server to a zone that has none yet", async () => {
        const user = userEvent.setup();
        const value = $state({ ns: null } as unknown as AbstractNSOnlyOriginBody);
        const { container } = render(Editor, { dn: "", origin, value });

        await user.click(screen.getByRole("button", { name: /New row/ }));
        await user.type(container.querySelector<HTMLInputElement>("#ns-0")!, "ns1");

        expect(value.ns.map((rr) => rr.Ns)).toEqual(["ns1"]);
    });
});

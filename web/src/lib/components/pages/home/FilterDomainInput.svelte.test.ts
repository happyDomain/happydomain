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
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

import { domains } from "$lib/stores/domains";
import { filteredName } from "$lib/stores/home";
import { loadTranslations } from "$lib/translations";
import FilterDomainInput from "./FilterDomainInput.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    filteredName.set("");
    domains.set(undefined);
});

async function typeName(name: string) {
    const user = userEvent.setup();
    domains.set([]);
    render(FilterDomainInput);

    await user.type(screen.getByPlaceholderText("my.domain."), name);

    return screen.getByRole("button", { name: "Add new domain" });
}

describe("FilterDomainInput", () => {
    it("offers to add a valid name", async () => {
        const add = await typeName("example.com");

        expect(add).toBeEnabled();
        expect(document.querySelector(".form-text")).toBeNull();
    });

    it("explains and blocks a name that cannot exist", async () => {
        const add = await typeName("a..example.com");

        expect(screen.getByText(/two consecutive dots/)).toHaveClass("text-danger");
        expect(add).toBeDisabled();
    });

    it("warns about an unusual name but still lets it be added", async () => {
        const add = await typeName("my_site.example.com");

        expect(screen.getByText(/character “_” is unusual/)).toHaveClass("text-warning-emphasis");
        expect(add).toBeEnabled();
    });
});

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
import { beforeAll, describe, expect, it, vi } from "vitest";

import type { Domain } from "$lib/model/domain";
import { loadTranslations } from "$lib/translations";
import NewSubdomain, { controls } from "./NewSubdomain.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

const origin = { domain: "example.com." } as Domain;

async function open() {
    const onnext = vi.fn();
    render(NewSubdomain, { props: { origin }, events: { "show-next-modal": onnext } });
    controls.Open();

    const input = await screen.findByPlaceholderText("new.subdomain");
    const proceed = screen.getByRole("button", { name: "Continue" });

    return { input, proceed, onnext };
}

describe("NewSubdomain", () => {
    it("cannot go on before a name is typed", async () => {
        const { proceed } = await open();

        expect(proceed).toBeDisabled();
    });

    it("goes on with the subdomain as typed", async () => {
        const user = userEvent.setup();
        const { input, proceed, onnext } = await open();

        await user.type(input, "blog");
        expect(proceed).toBeEnabled();
        await user.click(proceed);

        expect(onnext).toHaveBeenCalledTimes(1);
        expect(onnext.mock.calls[0][0].detail).toBe("blog");
    });

    it("goes on with a subdomain typed in full, keeping it absolute", async () => {
        const user = userEvent.setup();
        const { input, proceed, onnext } = await open();

        await user.type(input, "blog.example.com.");
        expect(proceed).toBeEnabled();
        await user.click(proceed);

        expect(onnext.mock.calls[0][0].detail).toBe("blog.example.com.");
    });

    it("refuses a name outside the zone", async () => {
        const user = userEvent.setup();
        const { input, proceed } = await open();

        await user.type(input, "blog.other.org.");

        expect(proceed).toBeDisabled();
        expect(screen.getByText("The name has to be within example.com.")).toBeVisible();
    });

    it("refuses a name that cannot exist", async () => {
        const user = userEvent.setup();
        const { input, proceed } = await open();

        await user.type(input, "a..b");

        expect(proceed).toBeDisabled();
    });
});

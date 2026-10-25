// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { cleanup, render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import type { ZoneTemplate } from "$lib/model/zone_template";
import { stubWebAnimations } from "$lib/test-utils/animations";
import { loadTranslations } from "$lib/translations";
import ModalTemplateDetails, { controls } from "./ModalTemplateDetails.svelte";

const template = {
    id: "t1",
    name: "Mail",
    description: "Our mail servers",
    default_ttl: 3600,
    services: {},
} as unknown as ZoneTemplate;

beforeAll(async () => {
    await loadTranslations("en", "/");
    stubWebAnimations();
});

afterEach(() => cleanup());

describe("ModalTemplateDetails", () => {
    it("saves the details changed", async () => {
        const user = userEvent.setup();
        const save = vi.fn().mockResolvedValue(template);
        render(ModalTemplateDetails, { template, save });
        controls.Open();

        const name = await screen.findByLabelText("Name");
        expect(name).toHaveValue("Mail");
        await user.clear(name);
        await user.type(name, "Mail servers");
        expect(screen.queryByLabelText("Origin")).toBeNull();
        await user.click(screen.getByRole("button", { name: "Save" }));

        await waitFor(() =>
            expect(save).toHaveBeenCalledWith({
                name: "Mail servers",
                description: "Our mail servers",
            }),
        );
    });

    it("tells why the details could not be saved", async () => {
        const user = userEvent.setup();
        const save = vi.fn().mockRejectedValue(new Error("A template needs a name."));
        render(ModalTemplateDetails, { template, save });
        controls.Open();

        await user.click(await screen.findByRole("button", { name: "Save" }));

        expect(await screen.findByText("A template needs a name.")).toBeVisible();
    });
});

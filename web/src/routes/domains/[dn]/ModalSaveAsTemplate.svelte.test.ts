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

const api = vi.hoisted(() => ({ createZoneTemplate: vi.fn() }));
vi.mock("$lib/api/zone_template", () => api);

const toasts = vi.hoisted(() => ({ addErrorToast: vi.fn(), addToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ toasts }));

const navigation = vi.hoisted(() => ({ goto: vi.fn() }));
vi.mock("$app/navigation", () => navigation);

import { get } from "svelte/store";

import type { Domain } from "$lib/model/domain";
import type { ZoneTemplate } from "$lib/model/zone_template";
import { zoneTemplates } from "$lib/stores/zone_templates";
import { stubWebAnimations } from "$lib/test-utils/animations";
import { loadTranslations } from "$lib/translations";
import ModalSaveAsTemplate, { controls } from "./ModalSaveAsTemplate.svelte";

const domain = {
    id: "d1",
    domain: "example.com.",
    zone_history: ["z2", "z1"],
} as unknown as Domain;

beforeAll(async () => {
    await loadTranslations("en", "/");
    stubWebAnimations();
});

afterEach(() => {
    cleanup();
    vi.resetAllMocks();
    zoneTemplates.set(undefined);
});

describe("ModalSaveAsTemplate", () => {
    it("saves the zone shown as a template", async () => {
        const user = userEvent.setup();
        const saved = { id: "t1", name: "Mail" } as ZoneTemplate;
        api.createZoneTemplate.mockResolvedValue(saved);
        zoneTemplates.set([]);
        render(ModalSaveAsTemplate, { domain, selectedHistory: "z1" });
        controls.Open();

        await user.type(await screen.findByLabelText("Name"), "Mail");
        await user.type(screen.getByLabelText("Description"), "Our mail servers");
        await user.click(screen.getByRole("button", { name: "Save the template" }));

        await waitFor(() =>
            expect(toasts.addToast).toHaveBeenCalledWith(
                expect.objectContaining({ title: "Template “Mail” saved" }),
            ),
        );
        expect(api.createZoneTemplate).toHaveBeenCalledWith({
            name: "Mail",
            description: "Our mail servers",
            domain_id: "d1",
            zone_id: "z1",
        });
        expect(get(zoneTemplates)).toEqual([saved]);
    });

    it("needs a name", async () => {
        render(ModalSaveAsTemplate, { domain, selectedHistory: "z1" });
        controls.Open();

        expect(await screen.findByRole("button", { name: "Save the template" })).toBeDisabled();
    });

    it("opens the template saved from the toast", async () => {
        const user = userEvent.setup();
        api.createZoneTemplate.mockResolvedValue({ id: "t1", name: "Mail" });
        render(ModalSaveAsTemplate, { domain, selectedHistory: "z1" });
        controls.Open();

        await user.type(await screen.findByLabelText("Name"), "Mail");
        await user.click(screen.getByRole("button", { name: "Save the template" }));

        await waitFor(() => expect(toasts.addToast).toHaveBeenCalled());
        toasts.addToast.mock.calls[0][0].onclick();
        expect(navigation.goto).toHaveBeenCalledWith("/templates/t1");
    });

    it("saves the zone being edited when none is shown", async () => {
        const user = userEvent.setup();
        api.createZoneTemplate.mockResolvedValue({ id: "t1", name: "Mail" });
        render(ModalSaveAsTemplate, { domain });
        controls.Open();

        await user.type(await screen.findByLabelText("Name"), "Mail");
        await user.click(screen.getByRole("button", { name: "Save the template" }));

        await waitFor(() =>
            expect(api.createZoneTemplate).toHaveBeenCalledWith(
                expect.objectContaining({ domain_id: "d1", zone_id: undefined }),
            ),
        );
    });

    it("tells why the template could not be saved, and stays open", async () => {
        const user = userEvent.setup();
        api.createZoneTemplate.mockRejectedValue(new Error("quota reached"));
        render(ModalSaveAsTemplate, { domain, selectedHistory: "z1" });
        controls.Open();

        await user.type(await screen.findByLabelText("Name"), "Mail");
        await user.click(screen.getByRole("button", { name: "Save the template" }));

        expect(await screen.findByText("quota reached")).toBeVisible();
        expect(screen.getByLabelText("Name")).toBeVisible();
        expect(toasts.addToast).not.toHaveBeenCalled();
    });
});

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

const navigation = vi.hoisted(() => ({ goto: vi.fn() }));
vi.mock("$app/navigation", () => navigation);
vi.mock("$app/state", () => ({ page: { params: { tid: "t1" } } }));

const toasts = vi.hoisted(() => ({ addErrorToast: vi.fn(), addToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ toasts }));

import type { ZoneTemplate } from "$lib/model/zone_template";
import { thisZone } from "$lib/stores/thiszone";
import { thisTemplate, zoneTemplates } from "$lib/stores/zone_templates";
import { stubWebAnimations } from "$lib/test-utils/animations";
import { loadTranslations } from "$lib/translations";
import type { ZoneTemplateBackend } from "./backend";
import TemplateEditorLayout from "./TemplateEditorLayout.svelte";

function tpl(id: string, name: string): ZoneTemplate {
    return {
        id,
        name,
        userId: "u1",
        default_ttl: 3600,
        services: {},
    } as unknown as ZoneTemplate;
}

const mail = tpl("t1", "Mail");
const web = tpl("t2", "Web");

beforeAll(async () => {
    await loadTranslations("en", "/");
    stubWebAnimations();
});

afterEach(() => {
    cleanup();
    vi.resetAllMocks();
    thisTemplate.set(null);
    thisZone.set(null);
    zoneTemplates.set(undefined);
});

function setup() {
    const backend = {
        list: vi.fn().mockResolvedValue([mail, web]),
        update: vi.fn(),
        delete: vi.fn().mockResolvedValue(undefined),
    } as unknown as ZoneTemplateBackend;
    render(TemplateEditorLayout, { template: mail, backend, readOnly: false });
    return backend;
}

describe("TemplateEditorLayout", () => {
    it("switches to the template chosen in the sidebar", async () => {
        const user = userEvent.setup();
        setup();

        const select = screen.getByRole("combobox", { name: "Zone templates" });
        await waitFor(() =>
            expect(screen.getByRole("option", { name: "Web" })).toBeInTheDocument(),
        );
        await user.selectOptions(select, "Web");

        expect(navigation.goto).toHaveBeenCalledWith("/templates/t2");
    });

    it("saves the details of the template", async () => {
        const user = userEvent.setup();
        const backend = setup();
        vi.mocked(backend.update).mockResolvedValue({
            ...mail,
            name: "Mail servers",
        } as ZoneTemplate);

        await user.click(screen.getByRole("button", { name: /Actions/ }));
        await user.click(await screen.findByText("Edit the details"));
        const name = await screen.findByLabelText("Name");
        await user.clear(name);
        await user.type(name, "Mail servers");
        await user.click(screen.getByRole("button", { name: "Save" }));

        await waitFor(() =>
            expect(backend.update).toHaveBeenCalledWith(
                "t1",
                expect.objectContaining({ name: "Mail servers" }),
            ),
        );
        await waitFor(() =>
            expect(screen.getByRole("combobox", { name: "Zone templates" })).toHaveDisplayValue(
                "Mail servers",
            ),
        );
    });

    it("deletes the template once confirmed, then goes back to the list", async () => {
        const user = userEvent.setup();
        const backend = setup();
        vi.stubGlobal("confirm", () => true);

        await user.click(screen.getByRole("button", { name: /Actions/ }));
        await user.click(await screen.findByText("Delete the template"));

        await waitFor(() => expect(navigation.goto).toHaveBeenCalledWith("/templates"));
        expect(backend.delete).toHaveBeenCalledWith("t1");
        expect(toasts.addToast).toHaveBeenCalledWith(
            expect.objectContaining({ title: "Template “Mail” deleted" }),
        );
    });

    it("tells why the template could not be deleted, and stays on it", async () => {
        const user = userEvent.setup();
        const backend = setup();
        vi.stubGlobal("confirm", () => true);
        vi.mocked(backend.delete).mockRejectedValue(new Error("unreachable"));

        await user.click(screen.getByRole("button", { name: /Actions/ }));
        await user.click(await screen.findByText("Delete the template"));

        await waitFor(() =>
            expect(toasts.addErrorToast).toHaveBeenCalledWith(
                expect.objectContaining({
                    title: "Unable to delete the template",
                    message: "unreachable",
                }),
            ),
        );
        expect(navigation.goto).not.toHaveBeenCalled();
    });

    it("keeps the template when the deletion is not confirmed", async () => {
        const user = userEvent.setup();
        const backend = setup();
        vi.stubGlobal("confirm", () => false);

        await user.click(screen.getByRole("button", { name: /Actions/ }));
        await user.click(await screen.findByText("Delete the template"));

        expect(backend.delete).not.toHaveBeenCalled();
    });
});

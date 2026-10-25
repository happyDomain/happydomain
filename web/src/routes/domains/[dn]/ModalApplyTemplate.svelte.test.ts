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
import { get } from "svelte/store";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ applyZoneTemplate: vi.fn(), listZoneTemplates: vi.fn() }));
vi.mock("$lib/api/zone_template", () => api);

const toasts = vi.hoisted(() => ({ addErrorToast: vi.fn(), addToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ toasts }));

import type { Domain } from "$lib/model/domain";
import type { ZoneTemplate } from "$lib/model/zone_template";
import { thisZone } from "$lib/stores/thiszone";
import { zoneTemplates } from "$lib/stores/zone_templates";
import { makeZone } from "$lib/test-utils/fixtures";
import { stubWebAnimations } from "$lib/test-utils/animations";
import { loadTranslations } from "$lib/translations";
import ModalApplyTemplate, { controls } from "./ModalApplyTemplate.svelte";
import { controls as ctrlDiffZone } from "./ModalDiffZone.svelte";

const domain = {
    id: "d1",
    domain: "example.com.",
    id_provider: "p1",
    zone_history: ["wip", "published"],
} as unknown as Domain;

function tpl(id: string, name: string, userId?: string): ZoneTemplate {
    return {
        id,
        name,
        userId,
        origin: "example.org.",
        default_ttl: 3600,
        services: {},
    } as unknown as ZoneTemplate;
}

beforeAll(async () => {
    await loadTranslations("en", "/");
    stubWebAnimations();
});

afterEach(() => {
    cleanup();
    vi.resetAllMocks();
    zoneTemplates.set(undefined);
    thisZone.set(null);
});

function open(selectedHistory = "wip") {
    const onapplied = vi.fn();
    render(ModalApplyTemplate, { domain, selectedHistory, onapplied });
    controls.Open();
    return onapplied;
}

describe("ModalApplyTemplate", () => {
    it("merges the template chosen into the zone being edited", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([tpl("t1", "Mail", "u1"), tpl("t2", "Workspace")]);
        const applied = makeZone({ id: "wip" });
        api.applyZoneTemplate.mockResolvedValue(applied);
        const onapplied = open();

        const choice = await screen.findByLabelText("Template");
        await user.selectOptions(choice, "Workspace");
        await user.click(screen.getByRole("button", { name: "Apply the template" }));

        await waitFor(() => expect(onapplied).toHaveBeenCalledWith(applied));
        expect(api.applyZoneTemplate).toHaveBeenCalledWith(domain, "wip", {
            template_id: "t2",
            mode: "merge",
        });
        expect(toasts.addToast).toHaveBeenCalledWith(
            expect.objectContaining({ title: "Template “Workspace” applied" }),
        );
    });

    it("replaces the services when asked, even from a past zone", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([tpl("t1", "Mail", "u1")]);
        api.applyZoneTemplate.mockResolvedValue(makeZone({ id: "wip" }));
        const onapplied = open("published");

        await screen.findByLabelText("Template");
        await user.click(screen.getByLabelText(/Replace/));
        await user.click(screen.getByRole("button", { name: "Apply the template" }));

        await waitFor(() => expect(onapplied).toHaveBeenCalled());
        expect(api.applyZoneTemplate).toHaveBeenCalledWith(domain, "wip", {
            template_id: "t1",
            mode: "overwrite",
        });
    });

    it("points to the templates page when there is none", async () => {
        api.listZoneTemplates.mockResolvedValue([]);
        open();

        expect(await screen.findByText(/You have no template yet/)).toBeVisible();
        expect(screen.getByRole("link", { name: "Manage my templates" })).toHaveAttribute(
            "href",
            "/templates",
        );
        expect(screen.getByRole("button", { name: "Apply the template" })).toBeDisabled();
    });

    it("tells why the template could not be applied", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([tpl("t1", "Mail", "u1")]);
        api.applyZoneTemplate.mockRejectedValue(
            new Error("The service svcs.MXs of the apex is invalid"),
        );
        const onapplied = open();

        await screen.findByLabelText("Template");
        await user.click(screen.getByRole("button", { name: "Apply the template" }));

        expect(
            await screen.findByText("The service svcs.MXs of the apex is invalid"),
        ).toBeVisible();
        expect(onapplied).not.toHaveBeenCalled();
    });

    it("shows the zone it went to, and opens the changes from the toast", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([tpl("t1", "Mail", "u1")]);
        const applied = makeZone({ id: "wip" });
        api.applyZoneTemplate.mockResolvedValue(applied);
        const openDiff = vi.spyOn(ctrlDiffZone, "Open").mockImplementation(() => {});
        const onapplied = open();

        await screen.findByLabelText("Template");
        await user.click(screen.getByRole("button", { name: "Apply the template" }));

        await waitFor(() => expect(onapplied).toHaveBeenCalledWith(applied));
        expect(get(thisZone)).toBe(applied);
        toasts.addToast.mock.calls[0][0].onclick();
        expect(openDiff).toHaveBeenCalled();
    });

    it("leaves the zone shown alone when the template went to a derived one", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([tpl("t1", "Mail", "u1")]);
        const shown = makeZone({ id: "wip" });
        thisZone.set(shown);
        const derived = makeZone({ id: "derived" });
        api.applyZoneTemplate.mockResolvedValue(derived);
        const onapplied = open();

        await screen.findByLabelText("Template");
        await user.click(screen.getByRole("button", { name: "Apply the template" }));

        await waitFor(() => expect(onapplied).toHaveBeenCalledWith(derived));
        expect(get(thisZone)).toBe(shown);
    });

    it("sends nothing for a domain without zone", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([tpl("t1", "Mail", "u1")]);
        const onapplied = vi.fn();
        render(ModalApplyTemplate, {
            domain: { ...domain, zone_history: [] },
            selectedHistory: "",
            onapplied,
        });
        controls.Open();

        await screen.findByLabelText("Template");
        await user.click(screen.getByRole("button", { name: "Apply the template" }));

        expect(api.applyZoneTemplate).not.toHaveBeenCalled();
        expect(onapplied).not.toHaveBeenCalled();
    });

    it("tells why the templates could not be loaded", async () => {
        api.listZoneTemplates.mockRejectedValue(new Error("unreachable"));
        open();

        expect(await screen.findByText("unreachable")).toBeVisible();
    });
});

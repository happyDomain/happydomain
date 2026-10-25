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

import { render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({
    listZoneTemplates: vi.fn(),
    updateZoneTemplate: vi.fn(),
    deleteZoneTemplate: vi.fn(),
}));
vi.mock("$lib/api/zone_template", () => api);

const toasts = vi.hoisted(() => ({ addErrorToast: vi.fn(), addToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ toasts }));

import type { ZoneTemplate } from "$lib/model/zone_template";
import { zoneTemplates } from "$lib/stores/zone_templates";
import { stubWebAnimations } from "$lib/test-utils/animations";
import { loadTranslations } from "$lib/translations";
import Page from "./+page.svelte";

function tpl(id: string, name: string, overrides: Partial<ZoneTemplate> = {}): ZoneTemplate {
    return {
        id,
        name,
        userId: "u1",
        default_ttl: 3600,
        services: { "": [{ _id: "s1", _svctype: "svcs.MXs", _domain: "" }] },
        ...overrides,
    } as unknown as ZoneTemplate;
}

beforeAll(async () => {
    await loadTranslations("en", "/");
    vi.stubGlobal("matchMedia", () => ({ matches: false }));
    stubWebAnimations();
});

afterEach(() => {
    vi.resetAllMocks();
    zoneTemplates.set(undefined);
});

function titles(): string[] {
    // The names of the templates and the section headings, in page order.
    return Array.from(document.querySelectorAll("h2, tbody td:first-child a")).map(
        (e) => e.textContent?.trim() ?? "",
    );
}

describe("zone templates page", () => {
    it("explains the templates while the user has none", async () => {
        api.listZoneTemplates.mockResolvedValue([]);
        render(Page);

        expect(await screen.findByText("No template yet")).toBeVisible();
        expect(screen.getByText("Save a zone")).toBeVisible();
    });

    it("lists the templates of the user, then the global ones apart", async () => {
        api.listZoneTemplates.mockResolvedValue([
            tpl("1", "Mail"),
            tpl("2", "Google Workspace", { userId: undefined }),
        ]);
        render(Page);

        await waitFor(() => expect(titles()).toContain("Mail"));
        expect(titles()).toEqual(["Mail", "Offered to everyone", "Google Workspace"]);
        expect(screen.getAllByText("1 service")).toHaveLength(2);
        expect(screen.getByRole("link", { name: "Mail" })).toHaveAttribute("href", "/templates/1");
    });

    it("shows the global templates beside the explanation when the user has none", async () => {
        api.listZoneTemplates.mockResolvedValue([
            tpl("2", "Google Workspace", { userId: undefined }),
        ]);
        render(Page);

        expect(await screen.findByText("No template yet")).toBeVisible();
        expect(screen.getByRole("link", { name: "Google Workspace" })).toBeVisible();
    });

    it("offers to try again when the templates could not be loaded", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockRejectedValueOnce(new Error("unreachable"));
        render(Page);

        expect(
            await screen.findByText(/Unable to load the templates: Error: unreachable/),
        ).toBeVisible();

        api.listZoneTemplates.mockResolvedValue([tpl("1", "Mail")]);
        await user.click(screen.getByText("Try again"));
        await waitFor(() => expect(titles()).toContain("Mail"));
    });
});

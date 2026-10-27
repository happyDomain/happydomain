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

import { fireEvent, render, screen, waitFor, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({
    createZoneTemplate: vi.fn(),
    importZoneTemplate: vi.fn(),
    listZoneTemplates: vi.fn(),
    updateZoneTemplate: vi.fn(),
    deleteZoneTemplate: vi.fn(),
}));
vi.mock("$lib/api/zone_template", () => api);

const toasts = vi.hoisted(() => ({ addErrorToast: vi.fn(), addToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ toasts }));

const navigation = vi.hoisted(() => ({ goto: vi.fn() }));
vi.mock("$app/navigation", () => navigation);

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

    it("lists the templates of the user beside the global ones, marked as such", async () => {
        api.listZoneTemplates.mockResolvedValue([
            tpl("1", "Mail"),
            tpl("2", "Google Workspace", { userId: undefined }),
        ]);
        render(Page);

        await waitFor(() => expect(titles()).toContain("Mail"));
        expect(titles()).toEqual(["Mail", "Google Workspace"]);
        const row = (name: string) => screen.getByRole("link", { name }).closest("tr")!;
        expect(within(row("Google Workspace")).getByText("Offered to everyone")).toBeVisible();
        expect(within(row("Mail")).queryByText("Offered to everyone")).toBeNull();
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

    it("creates an empty template, and opens it", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([]);
        api.createZoneTemplate.mockResolvedValue(tpl("3", "Web", { services: {} }));
        render(Page);
        await screen.findByText("No template yet");

        await user.click(screen.getByRole("button", { name: /New template/ }));
        await user.type(await screen.findByLabelText("Name"), "Web");
        expect(screen.queryByLabelText("Origin")).toBeNull();
        await user.click(screen.getByRole("button", { name: "Create the template" }));

        await waitFor(() => expect(navigation.goto).toHaveBeenCalledWith("/templates/3"));
        expect(api.createZoneTemplate).toHaveBeenCalledWith({ name: "Web", description: "" });
        expect(api.importZoneTemplate).not.toHaveBeenCalled();
    });

    it("imports a template from the zone file typed", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([]);
        api.importZoneTemplate.mockResolvedValue(tpl("4", "Imported"));
        render(Page);
        await screen.findByText("No template yet");

        await user.click(screen.getByRole("button", { name: /Import a zone file/ }));
        await user.type(await screen.findByLabelText("Name"), "Imported");
        await user.type(screen.getByLabelText("Origin"), "example.org");
        await user.type(screen.getByLabelText("Import from text"), "www 3600 IN A 192.0.2.1");
        await user.click(screen.getByRole("button", { name: "Import" }));

        await waitFor(() => expect(navigation.goto).toHaveBeenCalledWith("/templates/4"));
        const [input, file] = api.importZoneTemplate.mock.calls[0];
        expect(input).toEqual({ name: "Imported", description: "", origin: "example.org" });
        expect(await (file as Blob).text()).toBe("www 3600 IN A 192.0.2.1");
    });

    // Saved, a template is not to be made again for want of the list.
    it("opens the template created while the list could not be loaded", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockRejectedValue(new Error("unreachable"));
        api.createZoneTemplate.mockResolvedValue(tpl("3", "Web", { services: {} }));
        render(Page);
        await screen.findByText(/Unable to load the templates/);

        await user.click(screen.getByRole("button", { name: /New template/ }));
        await user.type(await screen.findByLabelText("Name"), "Web");
        await user.click(screen.getByRole("button", { name: "Create the template" }));

        await waitFor(() => expect(navigation.goto).toHaveBeenCalledWith("/templates/3"));
        expect(api.listZoneTemplates).toHaveBeenCalledTimes(1);
    });

    it("imports a template from the zone file chosen", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([]);
        api.importZoneTemplate.mockResolvedValue(tpl("5", "Uploaded"));
        const { baseElement } = render(Page);
        await screen.findByText("No template yet");

        await user.click(screen.getByRole("button", { name: /Import a zone file/ }));
        await user.type(await screen.findByLabelText("Name"), "Uploaded");
        await user.click(screen.getByText("Import from file"));
        const zone = new File(["mail 3600 IN A 192.0.2.2\n"], "example.org.zone");
        const input = baseElement.querySelector<HTMLInputElement>("input[type=file]")!;
        // jsdom cannot give files to an input, which the binding writes back.
        Object.defineProperty(input, "files", { value: [zone], writable: true });
        await fireEvent.change(input);
        await user.click(screen.getByRole("button", { name: "Import" }));

        await waitFor(() => expect(navigation.goto).toHaveBeenCalledWith("/templates/5"));
        expect(api.importZoneTemplate.mock.calls[0][1]).toBe(zone);
    });

    it("tells why a template could not be created", async () => {
        const user = userEvent.setup();
        api.listZoneTemplates.mockResolvedValue([]);
        api.importZoneTemplate.mockRejectedValue(new Error("Unable to read your zone file"));
        render(Page);
        await screen.findByText("No template yet");

        await user.click(screen.getByRole("button", { name: /Import a zone file/ }));
        await user.type(await screen.findByLabelText("Name"), "Broken");
        await user.click(screen.getByRole("button", { name: "Import" }));

        expect(await screen.findByText("Unable to read your zone file")).toBeVisible();
        expect(navigation.goto).not.toHaveBeenCalled();
    });
});

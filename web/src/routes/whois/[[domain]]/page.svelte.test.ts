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
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

const api = vi.hoisted(() => ({
    addDomainWatch: vi.fn(),
    getDomainInfo: vi.fn(),
    listDomainWatches: vi.fn(),
}));
vi.mock("$lib/api/domaininfo", () => ({ getDomainInfo: api.getDomainInfo }));
vi.mock("$lib/api/watchlist", () => ({
    addDomainWatch: api.addDomainWatch,
    listDomainWatches: api.listDomainWatches,
}));

const toasts = vi.hoisted(() => ({ addErrorToast: vi.fn(), addToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ toasts }));

import type { User } from "$lib/model/user";
import { userSession } from "$lib/stores/usersession";
import { watchlist } from "$lib/stores/watchlist";
import { stubWebAnimations } from "$lib/test-utils/animations";
import { loadTranslations } from "$lib/translations";
import Page from "./+page.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
    vi.stubGlobal("matchMedia", () => ({ matches: false }));
    stubWebAnimations();
});

beforeEach(() => {
    userSession.set({ id: "u1" } as User);
    api.listDomainWatches.mockResolvedValue([]);
});

afterEach(() => {
    vi.resetAllMocks();
    userSession.set({} as User);
    watchlist.set(undefined);
});

describe("registration data page", () => {
    it("tells a domain not registered is available, and offers to watch it", async () => {
        api.getDomainInfo.mockRejectedValue(new Error("domain not found"));
        render(Page, { data: { domain: "example.org" } });

        expect(await screen.findByText("Available now!")).toBeVisible();
        expect(screen.getByRole("button", { name: /Watch this domain/ })).toBeVisible();
    });

    it("shows the watch of a watched domain, even when the lookup fails", async () => {
        api.listDomainWatches.mockResolvedValue([
            { id: "1", domain: "example.com", phase: "redemption", note: "for my shop" },
        ]);
        api.getDomainInfo.mockRejectedValue(new Error("registry unreachable"));
        render(Page, { data: { domain: "example.com" } });

        expect(await screen.findByText("for my shop")).toBeVisible();
        expect(screen.getByText("Could not retrieve domain information.")).toBeVisible();
    });

    it("shows where the domain stands, then its registration details", async () => {
        api.getDomainInfo.mockResolvedValue({
            name: "example.com",
            expiration: "2030-01-01T00:00:00Z",
            registrar: "Acme",
            status: ["clientTransferProhibited"],
        });
        const { container } = render(Page, { data: { domain: "example.com" } });

        expect(await screen.findByRole("heading", { level: 3, name: "example.com" })).toBeVisible();
        // The card tells the name and the status: the details list the status
        // with their description instead.
        expect(container.querySelector("h2.font-monospace")).toBeNull();
        expect(screen.getByText("Transfer to another registrar is locked.")).toBeVisible();
    });

    it("shows the watch of a watched domain, then its registration details", async () => {
        api.listDomainWatches.mockResolvedValue([
            { id: "1", domain: "example.com", phase: "registered", note: "for my shop" },
        ]);
        api.getDomainInfo.mockResolvedValue({
            name: "example.com",
            expiration: "2030-01-01T00:00:00Z",
            status: ["clientTransferProhibited"],
        });
        render(Page, { data: { domain: "example.com" } });

        expect(await screen.findByText("for my shop")).toBeVisible();
        expect(screen.getByTitle("Check now")).toBeVisible();
        expect(screen.queryByTitle("Registration details")).toBeNull();
        expect(screen.getByText("Transfer to another registrar is locked.")).toBeVisible();
    });

    it("tells a watched domain available as soon as the lookup does", async () => {
        api.listDomainWatches.mockResolvedValue([
            { id: "1", domain: "example.com", phase: "pending_delete", note: "for my shop" },
        ]);
        api.getDomainInfo.mockRejectedValue(new Error("domain not found"));
        render(Page, { data: { domain: "example.com" } });

        expect(await screen.findByText("Available now!")).toBeVisible();
        expect(screen.getByText("for my shop")).toBeVisible();
    });

    it("shows the watch of the parent of a subdomain looked up", async () => {
        api.listDomainWatches.mockResolvedValue([
            { id: "1", domain: "example.com", phase: "redemption", note: "for my shop" },
        ]);
        api.getDomainInfo.mockRejectedValue(new Error("domain not found"));
        render(Page, { data: { domain: "www.example.com" } });

        expect(await screen.findByText("for my shop")).toBeVisible();
        expect(screen.queryByText("Available now!")).toBeNull();
    });

    it("turns the card into the watch once the domain is watched", async () => {
        const user = userEvent.setup();
        api.getDomainInfo.mockRejectedValue(new Error("domain not found"));
        api.addDomainWatch.mockResolvedValue({
            id: "1",
            domain: "example.org",
            phase: "available",
        });
        render(Page, { data: { domain: "example.org" } });

        await user.click(await screen.findByRole("button", { name: /Watch this domain/ }));

        expect(await screen.findByTitle("Check now")).toBeVisible();
    });

    it("loads the watchlist once", async () => {
        api.getDomainInfo.mockRejectedValue(new Error("domain not found"));
        render(Page, { data: { domain: "example.org" } });

        await screen.findByRole("button", { name: /Watch this domain/ });
        await waitFor(() => expect(api.listDomainWatches).toHaveBeenCalledTimes(1));
    });
});

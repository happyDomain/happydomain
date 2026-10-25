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

import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({
    addDomainWatch: vi.fn(),
    listDomainWatches: vi.fn(),
    checkDomainWatch: vi.fn(),
    deleteDomainWatch: vi.fn(),
    updateDomainWatch: vi.fn(),
}));
vi.mock("$lib/api/watchlist", () => api);

const toasts = vi.hoisted(() => ({ addErrorToast: vi.fn(), addToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ toasts }));

import type { User } from "$lib/model/user";
import type { DomainWatch } from "$lib/model/watchlist";
import { userSession } from "$lib/stores/usersession";
import { watchlist } from "$lib/stores/watchlist";
import { loadTranslations } from "$lib/translations";
import WatchCard from "./WatchCard.svelte";

const watch: DomainWatch = {
    id: "w1",
    domain: "example.com",
    phase: "registered",
    expiration: "2027-06-01T00:00:00Z",
    registrar: "Acme",
    note: "for my shop",
    lastCheck: "2026-10-24T11:00:00Z",
    nextCheck: "2026-10-31T11:00:00Z",
};

beforeAll(async () => {
    await loadTranslations("en", "/");
});

beforeEach(() => {
    watchlist.set([watch]);
});

afterEach(() => {
    vi.resetAllMocks();
    userSession.set({} as User);
    vi.unstubAllGlobals();
});

describe("WatchCard", () => {
    it("tells an auto-renewal waiting for the registrar", () => {
        render(WatchCard, { watch: { ...watch, status: ["autoRenewPeriod"] } });

        expect(screen.getByText("Auto-renewed")).toBeVisible();
        expect(screen.getByText(/still has to confirm the renewal/)).toBeVisible();
    });

    it("hides a registrar the lookup did not know", () => {
        render(WatchCard, { watch: { ...watch, registrar: "Unknown" } });

        expect(screen.queryByText("Unknown")).toBeNull();
    });

    it("lists the history, the latest change first", async () => {
        const user = userEvent.setup();
        render(WatchCard, {
            watch: {
                ...watch,
                history: [
                    { at: "2026-01-01T00:00:00Z", from: "unknown", to: "available" },
                    { at: "2026-02-01T00:00:00Z", from: "available", to: "registered" },
                ],
            },
        });

        await user.click(screen.getByTitle("History"));

        const items = screen
            .getAllByRole("listitem")
            .filter((li) => li.textContent?.includes("—"))
            .map((li) => li.textContent?.replace(/\s+/g, " "));
        expect(items).toHaveLength(2);
        expect(items[0]).toMatch(/Available Registered/);
        expect(items[1]).toMatch(/Not checked yet Available/);
    });

    it("checks the domain now", async () => {
        const user = userEvent.setup();
        api.checkDomainWatch.mockResolvedValue({ ...watch, phase: "expiring" });
        render(WatchCard, { watch });

        await user.click(screen.getByTitle("Check now"));

        expect(api.checkDomainWatch).toHaveBeenCalledWith("w1");
        expect(get(watchlist)?.[0].phase).toBe("expiring");
    });

    it("stops watching only once confirmed", async () => {
        const user = userEvent.setup();
        vi.stubGlobal("confirm", () => false);
        render(WatchCard, { watch });

        await user.click(screen.getByTitle("Stop watching"));

        expect(api.deleteDomainWatch).not.toHaveBeenCalled();
    });

    it("lets the user try again a removal that failed", async () => {
        const user = userEvent.setup();
        vi.stubGlobal("confirm", () => true);
        api.deleteDomainWatch.mockRejectedValue(new Error("unreachable"));
        render(WatchCard, { watch });

        await user.click(screen.getByTitle("Stop watching"));

        expect(toasts.addErrorToast).toHaveBeenCalled();
        expect(screen.getByTitle("Stop watching")).toBeEnabled();
        expect(get(watchlist)).toHaveLength(1);
    });

    it("saves the note, trimmed", async () => {
        const user = userEvent.setup();
        api.updateDomainWatch.mockImplementation(async (_id: string, note: string) => ({
            ...watch,
            note,
        }));
        render(WatchCard, { watch });

        await user.click(screen.getAllByTitle("Edit the note")[0]);
        const input = screen.getByPlaceholderText("Why do you want this domain?");
        await user.clear(input);
        await user.type(input, "  to sell flowers  {Enter}");

        expect(api.updateDomainWatch).toHaveBeenCalledWith("w1", "to sell flowers");
        expect(get(watchlist)?.[0].note).toBe("to sell flowers");
    });

    it("keeps the note being edited when it cannot be saved", async () => {
        const user = userEvent.setup();
        api.updateDomainWatch.mockRejectedValue(new Error("unreachable"));
        render(WatchCard, { watch });

        await user.click(screen.getAllByTitle("Edit the note")[0]);
        await user.type(screen.getByPlaceholderText("Why do you want this domain?"), "{Enter}");

        expect(toasts.addErrorToast).toHaveBeenCalled();
        expect(screen.getByPlaceholderText("Why do you want this domain?")).toBeVisible();
    });

    it("lets the user try again a check that failed", async () => {
        const user = userEvent.setup();
        api.checkDomainWatch.mockRejectedValue(new Error("unreachable"));
        render(WatchCard, { watch });

        await user.click(screen.getByTitle("Check now"));

        expect(toasts.addErrorToast).toHaveBeenCalled();
        expect(screen.getByTitle("Check now")).toBeEnabled();
        expect(get(watchlist)?.[0]).toEqual(watch);
    });

    it("drops the note being edited when cancelled", async () => {
        const user = userEvent.setup();
        render(WatchCard, { watch });

        await user.click(screen.getAllByTitle("Edit the note")[0]);
        await user.type(screen.getByPlaceholderText("Why do you want this domain?"), " and more");
        await user.click(screen.getByTitle("Cancel"));

        expect(api.updateDomainWatch).not.toHaveBeenCalled();
        expect(screen.queryByPlaceholderText("Why do you want this domain?")).toBeNull();
        expect(screen.getByText("for my shop")).toBeVisible();
    });

    it("names an internationalized domain in Unicode, and in ASCII below", () => {
        render(WatchCard, {
            watch: { ...watch, domain: "xn--bcher-kva.example", displayName: "bücher.example" },
        });

        expect(screen.getByRole("heading", { level: 3 })).toHaveTextContent("bücher.example");
        expect(screen.getByText("xn--bcher-kva.example")).toBeVisible();
    });

    it("tells by when a domain pending deletion should be released", () => {
        render(WatchCard, {
            watch: {
                ...watch,
                phase: "pending_delete",
                estimatedRelease: "2026-10-29T12:00:00Z",
            },
        });

        expect(screen.getByText(/at the latest/)).toHaveTextContent(
            /By .*2026 at the latest\. It can no longer be restored/,
        );
    });

    it("tells a domain in redemption can still be restored", () => {
        render(WatchCard, { watch: { ...watch, phase: "redemption" } });

        expect(screen.getByText("Its owner can still restore it, for a fee.")).toBeVisible();
        expect(screen.queryByText(/at the latest/)).toBeNull();
    });

    it("tells why the last check failed, and a domain never checked", () => {
        render(WatchCard, {
            watch: { ...watch, phase: "unknown", lastCheck: undefined, lastError: "timeout" },
        });

        expect(screen.getByText("Last check failed: timeout")).toBeVisible();
        expect(screen.getByText("Never checked")).toBeVisible();
        expect(screen.getByText("Registration data unavailable")).toBeVisible();
    });

    it("offers to watch a domain looked up, without the actions of a watch", () => {
        userSession.set({ id: "u1" } as User);
        render(WatchCard, { watch: { domain: "example.org", phase: "available" } });

        expect(screen.getByText("Available now!")).toBeVisible();
        expect(screen.getByRole("button", { name: /Watch this domain/ })).toBeVisible();
        expect(screen.queryByTitle("Check now")).toBeNull();
        expect(screen.queryByText(/Checked|Never checked/)).toBeNull();
    });

    it("leaves out the link to the registration details when asked to", () => {
        render(WatchCard, { watch, showDetails: false });

        expect(screen.queryByTitle("Registration details")).toBeNull();
        expect(screen.getByTitle("Check now")).toBeVisible();
    });
});

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
    addDomainWatch: vi.fn(),
    listDomainWatches: vi.fn(),
}));
vi.mock("$lib/api/watchlist", () => api);

const toasts = vi.hoisted(() => ({ addErrorToast: vi.fn(), addToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ toasts }));

import type { DomainWatch } from "$lib/model/watchlist";
import { watchlist } from "$lib/stores/watchlist";
import { stubWebAnimations } from "$lib/test-utils/animations";
import { loadTranslations } from "$lib/translations";
import Page from "./+page.svelte";

const free: DomainWatch = { id: "1", domain: "zfree.example", phase: "available" };
const held: DomainWatch = {
    id: "2",
    domain: "aheld.example",
    phase: "registered",
    expiration: "2027-06-01T00:00:00Z",
};

beforeAll(async () => {
    await loadTranslations("en", "/");
    vi.stubGlobal("matchMedia", () => ({ matches: false }));
    stubWebAnimations();
});

afterEach(() => {
    vi.resetAllMocks();
    watchlist.set(undefined);
});

function names(): string[] {
    return screen.getAllByRole("heading", { level: 3 }).map((h) => h.textContent ?? "");
}

describe("watchlist page", () => {
    it("explains the watchlist while it is empty", async () => {
        api.listDomainWatches.mockResolvedValue([]);
        render(Page);

        expect(await screen.findByText("Covet a domain name already taken?")).toBeVisible();
    });

    it("lists the most urgent first, and sorts on demand", async () => {
        const user = userEvent.setup();
        api.listDomainWatches.mockResolvedValue([held, free]);
        render(Page);

        await waitFor(() => expect(names()).toEqual(["zfree.example", "aheld.example"]));

        await user.click(screen.getByText(/Sort by/));
        await user.click(screen.getByText("Name"));
        expect(names()).toEqual(["aheld.example", "zfree.example"]);
    });

    it("filters the list with what is typed", async () => {
        const user = userEvent.setup();
        api.listDomainWatches.mockResolvedValue([held, free]);
        render(Page);
        await waitFor(() => expect(names()).toHaveLength(2));

        const input = screen.getByPlaceholderText("example.com");
        await user.type(input, "held");
        expect(names()).toEqual(["aheld.example"]);

        await user.clear(input);
        await user.type(input, "nothing");
        expect(screen.getByText("No watched domain matches “nothing”.")).toBeVisible();
    });

    it("offers to try again when the list could not be loaded", async () => {
        const user = userEvent.setup();
        api.listDomainWatches.mockRejectedValueOnce(new Error("unreachable"));
        render(Page);

        expect(
            await screen.findByText(/Unable to load your watchlist: Error: unreachable/),
        ).toBeVisible();

        api.listDomainWatches.mockResolvedValue([held]);
        await user.click(screen.getByText("Try again"));
        await waitFor(() => expect(names()).toEqual(["aheld.example"]));
    });

    it("refuses a bare label, which no one can register", async () => {
        const user = userEvent.setup();
        api.listDomainWatches.mockResolvedValue([]);
        render(Page);

        const input = screen.getByPlaceholderText("example.com");
        const button = screen.getByRole("button", { name: /Watch/ });

        await user.type(input, "example");
        expect(button).toBeDisabled();

        await user.type(input, ".com.");
        expect(button).toBeEnabled();
    });

    it("adds a domain, and draws attention to it", async () => {
        const user = userEvent.setup();
        api.listDomainWatches.mockResolvedValue([held]);
        api.addDomainWatch.mockResolvedValue(free);
        const { container } = render(Page);
        await waitFor(() => expect(names()).toHaveLength(1));

        const input = screen.getByPlaceholderText("example.com");
        await user.type(input, "zfree.example{Enter}");

        expect(api.addDomainWatch).toHaveBeenCalledWith({ domain: "zfree.example", note: "" });
        await waitFor(() => expect(names()).toEqual(["zfree.example", "aheld.example"]));
        expect(input).toHaveValue("");
        expect(container.querySelector(".watch-card.highlight")?.textContent).toContain(
            "zfree.example",
        );
    });

    it("tells why a domain could not be added, until the field changes", async () => {
        const user = userEvent.setup();
        api.listDomainWatches.mockResolvedValue([]);
        api.addDomainWatch.mockRejectedValue(new Error("you already watch zfree.example"));
        render(Page);

        const input = screen.getByPlaceholderText("example.com");
        await user.type(input, "zfree.example{Enter}");
        expect(await screen.findByText("you already watch zfree.example")).toBeVisible();

        await user.type(input, "x");
        expect(screen.queryByText("you already watch zfree.example")).toBeNull();
    });

    it("adds a domain with its note, trimmed", async () => {
        const user = userEvent.setup();
        api.listDomainWatches.mockResolvedValue([]);
        api.addDomainWatch.mockResolvedValue(free);
        render(Page);

        await user.click(screen.getByText("Add a note"));
        await user.type(
            screen.getByPlaceholderText("Why do you want this domain?"),
            "  for the shop ",
        );
        await user.type(screen.getByPlaceholderText("example.com"), "zfree.example{Enter}");

        expect(api.addDomainWatch).toHaveBeenCalledWith({
            domain: "zfree.example",
            note: "for the shop",
        });
        await waitFor(() => expect(screen.queryByText("Add a note")).toBeVisible());
    });

    it("locks the field while the domain is added", async () => {
        const user = userEvent.setup();
        api.listDomainWatches.mockResolvedValue([]);
        let resolveAdd: (w: DomainWatch) => void = () => {};
        api.addDomainWatch.mockReturnValue(new Promise((r) => (resolveAdd = r)));
        render(Page);

        const input = screen.getByPlaceholderText("example.com");
        await user.type(input, "zfree.example{Enter}");
        expect(input).toBeDisabled();
        expect(screen.getByRole("button", { name: /Checking…/ })).toBeDisabled();

        resolveAdd(free);
        await waitFor(() => expect(input).toBeEnabled());
    });

    it("loads the list when a domain is added before it came", async () => {
        const user = userEvent.setup();
        api.listDomainWatches.mockReturnValueOnce(new Promise(() => {}));
        api.listDomainWatches.mockResolvedValue([free, held]);
        api.addDomainWatch.mockResolvedValue(free);
        render(Page);

        await user.type(screen.getByPlaceholderText("example.com"), "zfree.example{Enter}");

        await waitFor(() => expect(names()).toEqual(["zfree.example", "aheld.example"]));
        expect(api.listDomainWatches).toHaveBeenCalledTimes(2);
    });

    it("keeps the list shown when it cannot be refreshed, and tells it", async () => {
        watchlist.set([held]);
        api.listDomainWatches.mockRejectedValue(new Error("unreachable"));
        render(Page);

        await waitFor(() => expect(toasts.addErrorToast).toHaveBeenCalled());
        expect(names()).toEqual(["aheld.example"]);
        expect(screen.queryByText("Try again")).toBeNull();
    });
});

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
import { get } from "svelte/store";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({
    addDomainWatch: vi.fn(),
    listDomainWatches: vi.fn(),
}));
vi.mock("$lib/api/watchlist", () => api);

const toasts = vi.hoisted(() => ({ addErrorToast: vi.fn(), addToast: vi.fn() }));
vi.mock("$lib/stores/toasts", () => ({ toasts }));

import type { User } from "$lib/model/user";
import { userSession } from "$lib/stores/usersession";
import { watchlist } from "$lib/stores/watchlist";
import { loadTranslations } from "$lib/translations";
import WatchDomainButton from "./WatchDomainButton.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    vi.resetAllMocks();
    userSession.set({} as User);
    watchlist.set(undefined);
});

function logIn() {
    userSession.set({ id: "u1", email: "alice@example.com" } as User);
}

describe("WatchDomainButton", () => {
    it("is not offered to a visitor", () => {
        render(WatchDomainButton, { domain: "example.com" });

        expect(screen.queryByRole("button")).toBeNull();
        expect(screen.queryByRole("link")).toBeNull();
        expect(api.listDomainWatches).not.toHaveBeenCalled();
    });

    it("leads to the watchlist when the domain, or its parent, is watched", async () => {
        logIn();
        api.listDomainWatches.mockResolvedValue([{ id: "1", domain: "example.com" }]);
        render(WatchDomainButton, { domain: "www.example.com" });

        expect(await screen.findByRole("link", { name: /In your watchlist/ })).toBeVisible();
    });

    it("watches the domain", async () => {
        const user = userEvent.setup();
        logIn();
        watchlist.set([]);
        api.addDomainWatch.mockResolvedValue({ id: "1", domain: "example.com" });
        render(WatchDomainButton, { domain: "example.com" });

        await user.click(screen.getByRole("button", { name: /Watch this domain/ }));

        expect(api.addDomainWatch).toHaveBeenCalledWith({ domain: "example.com", note: "" });
        expect(get(watchlist)).toEqual([{ id: "1", domain: "example.com" }]);
        expect(toasts.addToast).toHaveBeenCalled();
        expect(screen.getByRole("link", { name: /In your watchlist/ })).toBeVisible();
    });

    it("loads the watchlist not loaded yet once the domain is watched", async () => {
        const user = userEvent.setup();
        logIn();
        // The first load fails: the watchlist is still unknown.
        api.listDomainWatches.mockRejectedValueOnce(new Error("unreachable"));
        api.addDomainWatch.mockResolvedValue({ id: "1", domain: "example.com" });
        render(WatchDomainButton, { domain: "example.com" });
        await waitFor(() => expect(api.listDomainWatches).toHaveBeenCalledTimes(1));

        api.listDomainWatches.mockResolvedValue([{ id: "1", domain: "example.com" }]);
        await user.click(screen.getByRole("button", { name: /Watch this domain/ }));

        await waitFor(() => expect(get(watchlist)).toEqual([{ id: "1", domain: "example.com" }]));
    });

    it("tells why the domain could not be watched", async () => {
        const user = userEvent.setup();
        logIn();
        watchlist.set([]);
        api.addDomainWatch.mockRejectedValue(new Error("you cannot watch more than 100 domains"));
        render(WatchDomainButton, { domain: "example.com" });

        await user.click(screen.getByRole("button", { name: /Watch this domain/ }));

        expect(toasts.addErrorToast).toHaveBeenCalledWith(
            expect.objectContaining({ message: expect.stringContaining("more than 100") }),
        );
        expect(screen.getByRole("button", { name: /Watch this domain/ })).toBeEnabled();
    });
});

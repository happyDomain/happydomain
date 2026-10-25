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

import { get } from "svelte/store";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ listDomainWatches: vi.fn() }));
vi.mock("$lib/api/watchlist", () => api);

import { refreshWatchlist, removeWatch, upsertWatch, watchlist } from "./watchlist";

describe("watchlist store", () => {
    beforeEach(() => {
        watchlist.set(undefined);
    });

    it("loads the watchlist", async () => {
        api.listDomainWatches.mockResolvedValue([{ id: "1", domain: "example.com" }]);

        await refreshWatchlist();
        expect(get(watchlist)).toEqual([{ id: "1", domain: "example.com" }]);
    });

    it("leaves a watchlist not loaded yet alone", () => {
        upsertWatch({ id: "1", domain: "example.com" });
        removeWatch("1");
        expect(get(watchlist)).toBeUndefined();
    });

    it("adds a new watch first, and replaces a known one in place", () => {
        watchlist.set([
            { id: "1", domain: "a.example" },
            { id: "2", domain: "b.example" },
        ]);

        upsertWatch({ id: "3", domain: "c.example" });
        upsertWatch({ id: "1", domain: "a.example", note: "mine" });

        expect(get(watchlist)).toEqual([
            { id: "3", domain: "c.example" },
            { id: "1", domain: "a.example", note: "mine" },
            { id: "2", domain: "b.example" },
        ]);
    });

    it("removes a watch", () => {
        watchlist.set([
            { id: "1", domain: "a.example" },
            { id: "2", domain: "b.example" },
        ]);

        removeWatch("1");
        expect(get(watchlist)).toEqual([{ id: "2", domain: "b.example" }]);
    });

    it("keeps the watchlist it holds when it cannot be refreshed", async () => {
        watchlist.set([{ id: "1", domain: "a.example" }]);
        api.listDomainWatches.mockRejectedValue(new Error("unreachable"));

        await expect(refreshWatchlist()).rejects.toThrow("unreachable");
        expect(get(watchlist)).toEqual([{ id: "1", domain: "a.example" }]);
    });
});

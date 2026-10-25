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

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { DomainWatch } from "$lib/model/watchlist";
import {
    compareWatches,
    findWatch,
    isAutoRenewPending,
    lifecycleStep,
    phaseOfDomainInfo,
    updatedWatch,
    watchComparator,
    watchHeadline,
    watchOfDomainInfo,
} from "./watchlist";

const NOW = new Date("2026-10-24T12:00:00Z");
const inDays = (d: number) => new Date(NOW.getTime() + d * 86400000).toISOString();

describe("watchlist helpers", () => {
    beforeEach(() => {
        vi.useFakeTimers();
        vi.setSystemTime(NOW);
    });
    afterEach(() => {
        vi.useRealTimers();
    });

    it("places a phase in the life cycle", () => {
        expect(lifecycleStep("registered")).toBe(0);
        expect(lifecycleStep("expiring")).toBe(0);
        expect(lifecycleStep("grace")).toBe(1);
        expect(lifecycleStep("pending_delete")).toBe(3);
        expect(lifecycleStep("available")).toBe(4);
        expect(lifecycleStep("unknown")).toBe(-1);
        expect(lifecycleStep(undefined)).toBe(-1);
    });

    it("sorts the most urgent first, then the soonest", () => {
        const list: DomainWatch[] = [
            { domain: "far.example", phase: "registered", expiration: inDays(300) },
            { domain: "new.example", phase: "unknown" },
            { domain: "free.example", phase: "available" },
            { domain: "soon.example", phase: "registered", expiration: inDays(100) },
            { domain: "drop.example", phase: "pending_delete", estimatedRelease: inDays(2) },
            { domain: "a.example", phase: "expiring", expiration: inDays(10) },
            { domain: "b.example", phase: "expiring", expiration: inDays(10) },
        ];
        expect(list.sort(compareWatches).map((w) => w.domain)).toEqual([
            "free.example",
            "drop.example",
            "a.example",
            "b.example",
            "soon.example",
            "far.example",
            "new.example",
        ]);
    });

    it("sorts by name, expiration date or date added", () => {
        const list: DomainWatch[] = [
            {
                domain: "b.example",
                phase: "registered",
                expiration: inDays(100),
                createdAt: inDays(-1),
            },
            { domain: "xn--bcher-kva.de", displayName: "bücher.de", phase: "available" },
            {
                domain: "a.example",
                phase: "registered",
                expiration: inDays(10),
                createdAt: inDays(-5),
            },
            {
                domain: "c.example",
                phase: "expiring",
                expiration: inDays(10),
                createdAt: inDays(-1),
            },
        ];
        const sorted = (sort: Parameters<typeof watchComparator>[0]) =>
            [...list].sort(watchComparator(sort)).map((w) => w.domain);

        expect(sorted("name")).toEqual(["a.example", "b.example", "xn--bcher-kva.de", "c.example"]);
        // Same date: the most urgent first. No date: last.
        expect(sorted("expiration")).toEqual([
            "c.example",
            "a.example",
            "b.example",
            "xn--bcher-kva.de",
        ]);
        // Most recent first. Same date: the most urgent first. No date: last.
        expect(sorted("added")).toEqual([
            "c.example",
            "b.example",
            "a.example",
            "xn--bcher-kva.de",
        ]);
        expect(sorted("urgency")).toEqual(
            list
                .slice()
                .sort(compareWatches)
                .map((w) => w.domain),
        );
    });

    it("tells what matters most about a domain", () => {
        expect(watchHeadline({ phase: "available" })).toEqual(["watchlist.headline.available", {}]);
        expect(watchHeadline({ phase: "pending_delete", estimatedRelease: inDays(3) })).toEqual([
            "watchlist.headline.release-in",
            { days: 3 },
        ]);
        expect(watchHeadline({ phase: "pending_delete", estimatedRelease: inDays(-1) })).toEqual([
            "watchlist.headline.release-imminent",
            {},
        ]);
        expect(watchHeadline({ phase: "redemption" })).toEqual([
            "watchlist.headline.release-soon",
            {},
        ]);
        expect(watchHeadline({ phase: "grace", expiration: inDays(-12) })).toEqual([
            "watchlist.headline.expired-ago",
            { days: 12 },
        ]);
        expect(watchHeadline({ phase: "expiring", expiration: inDays(20) })).toEqual([
            "watchlist.headline.expires-in",
            { days: 20 },
        ]);
        expect(watchHeadline({ phase: "grace" })).toEqual(["watchlist.headline.expired", {}]);
        expect(watchHeadline({ phase: "expiring", expiration: NOW.toISOString() })).toEqual([
            "watchlist.headline.expires-today",
            {},
        ]);
        expect(watchHeadline({ phase: "registered", expiration: inDays(200) })).toEqual([
            "watchlist.headline.expires-in",
            { days: 200 },
        ]);
        expect(watchHeadline({ phase: "registered" })).toEqual([
            "watchlist.headline.no-expiration",
            {},
        ]);
        expect(watchHeadline({ phase: "unknown", lastError: "timeout" })).toEqual([
            "watchlist.headline.lookup-failed",
            {},
        ]);
        expect(watchHeadline({ phase: "unknown" })).toEqual(["watchlist.headline.not-checked", {}]);
    });

    it("tells the phase of a domain from its lookup", () => {
        expect(phaseOfDomainInfo(null)).toBe("available");
        expect(phaseOfDomainInfo({ expiration: inDays(100) })).toBe("registered");
        expect(phaseOfDomainInfo({})).toBe("registered");
        expect(phaseOfDomainInfo({ expiration: inDays(10) })).toBe("expiring");
        expect(phaseOfDomainInfo({ expiration: inDays(30) })).toBe("expiring");
        expect(phaseOfDomainInfo({ expiration: inDays(31) })).toBe("registered");
        // Not rounded to days: the server would tell it registered too.
        expect(phaseOfDomainInfo({ expiration: inDays(30.25) })).toBe("registered");
        expect(phaseOfDomainInfo({ expiration: inDays(-2) })).toBe("grace");
        expect(phaseOfDomainInfo({ expiration: inDays(-2), status: ["redemption period"] })).toBe(
            "redemption",
        );
        expect(phaseOfDomainInfo({ status: ["pendingRestore"] })).toBe("redemption");
        expect(phaseOfDomainInfo({ status: ["redemptionPeriod", "pendingDelete"] })).toBe(
            "pending_delete",
        );
    });

    it("describes a domain looked up as an unsaved watch", () => {
        const w = watchOfDomainInfo("example.com", {
            name: "EXAMPLE.com.",
            expiration: inDays(100),
            registrar: "Registrar",
            status: ["ok"],
        });
        expect(w.id).toBeUndefined();
        expect(w.domain).toBe("EXAMPLE.com");
        expect(w.phase).toBe("registered");
        expect(w.registrar).toBe("Registrar");
        expect(watchOfDomainInfo("example.org", null)).toEqual({
            domain: "example.org",
            phase: "available",
            expiration: undefined,
            registrar: undefined,
            status: undefined,
        });
    });

    it("finds the watch of a domain or of its parent", () => {
        const list: DomainWatch[] = [
            { id: "1", domain: "example.com" },
            { id: "2", domain: "xn--bcher-kva.de", displayName: "bücher.de" },
        ];
        expect(findWatch(list, "example.com")?.id).toBe("1");
        expect(findWatch(list, "Www.Example.com.")?.id).toBe("1");
        expect(findWatch(list, "bücher.de")?.id).toBe("2");
        expect(findWatch(list, "notexample.com")).toBeUndefined();
        expect(findWatch([{ id: "3", domain: "shop.example.co" }], "example.co")).toBeUndefined();
        expect(findWatch(undefined, "example.com")).toBeUndefined();
    });

    it("tells an auto-renewal waiting for its registrar", () => {
        expect(isAutoRenewPending({ status: ["autoRenewPeriod"] })).toBe(true);
        expect(isAutoRenewPending({ status: ["auto renew period"] })).toBe(true);
        expect(isAutoRenewPending({ status: ["renewPeriod"] })).toBe(false);
        expect(isAutoRenewPending({})).toBe(false);
    });

    it("updates a watch with what a lookup of its domain tells", () => {
        const w: DomainWatch = {
            id: "1",
            domain: "example.com",
            note: "mine",
            phase: "pending_delete",
            estimatedRelease: inDays(2),
            status: ["pendingDelete"],
        };

        const freed = updatedWatch(w, "Example.com.", {
            domain: "example.com",
            phase: "available",
        });
        expect(freed).toEqual({
            id: "1",
            domain: "example.com",
            note: "mine",
            phase: "available",
            estimatedRelease: undefined,
            expiration: undefined,
            registrar: undefined,
            status: undefined,
        });

        const same = updatedWatch(w, "example.com", {
            domain: "example.com",
            phase: "pending_delete",
            status: ["pendingDelete"],
        });
        expect(same.estimatedRelease).toBe(w.estimatedRelease);

        // The lookup of a subdomain tells nothing of the watched domain.
        expect(updatedWatch(w, "www.example.com", { phase: "available" })).toBe(w);
    });
});

// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
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
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

// The network lives here: the tests decide what the backend answers.
const api = vi.hoisted(() => ({
    getRetrieveZoneProgress: vi.fn(),
    retrieveZone: vi.fn(),
    refreshDomains: vi.fn(),
}));
vi.mock("$lib/api/zone", () => ({ getRetrieveZoneProgress: api.getRetrieveZoneProgress }));
vi.mock("$lib/stores/thiszone", () => ({ retrieveZone: api.retrieveZone }));
vi.mock("$lib/stores/domains", async (importOriginal) => ({
    ...(await importOriginal<typeof import("$lib/stores/domains")>()),
    refreshDomains: api.refreshDomains,
}));

import { goto } from "$app/navigation";
import type { RetrieveZoneProgress } from "$lib/api/zone";
import type { Domain } from "$lib/model/domain";
import { domains } from "$lib/stores/domains";
import { loadTranslations } from "$lib/translations";
import Page from "./+page.svelte";

const domain = { id: "d1", domain: "example.com." } as unknown as Domain;

const exploring: RetrieveZoneProgress = {
    step: "exploring",
    percent: 40,
    step_end: 60,
} as RetrieveZoneProgress;

beforeAll(async () => {
    await loadTranslations("en", "/");
    vi.stubGlobal("matchMedia", () => ({ matches: false }));
});

beforeEach(() => {
    vi.useFakeTimers();
});

afterEach(() => {
    vi.useRealTimers();
    vi.resetAllMocks();
    domains.set(undefined);
});

// rejectLater returns a promise rejected with err when reject is called.
function rejectLater(err: Error) {
    let reject = () => {};
    const promise = new Promise((_, r) => (reject = () => r(err)));
    return { promise, reject };
}

describe("import_zone page", () => {
    it("tells at once the failure of a request no progress was seen of", async () => {
        api.getRetrieveZoneProgress.mockResolvedValue(null);
        api.retrieveZone.mockRejectedValue(new Error("unreachable"));

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(0);

        expect(screen.getByText(/unreachable/)).toBeInTheDocument();
    });

    it("goes to the zone imported after its request failed", async () => {
        const request = rejectLater(new Error("gateway timeout"));
        api.retrieveZone.mockReturnValue(request.promise);
        api.getRetrieveZoneProgress.mockResolvedValueOnce(exploring).mockResolvedValue(null);
        api.refreshDomains.mockImplementation(async () => {
            domains.set([{ ...domain, zone_history: ["z1"] } as never]);
        });

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(1000);
        request.reject();
        await vi.advanceTimersByTimeAsync(1000);

        expect(goto).toHaveBeenCalled();
        expect(screen.queryByText(/gateway timeout/)).toBeNull();
    });

    it("tells the failure of the request once the import ended without a zone", async () => {
        const request = rejectLater(new Error("gateway timeout"));
        api.retrieveZone.mockReturnValue(request.promise);
        api.getRetrieveZoneProgress.mockResolvedValueOnce(exploring).mockResolvedValue(null);

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(1000);
        request.reject();
        // The import may still run: the failure waits for the next poll.
        await vi.advanceTimersByTimeAsync(0);
        expect(screen.queryByText(/gateway timeout/)).toBeNull();

        await vi.advanceTimersByTimeAsync(1000);
        expect(screen.getByText(/gateway timeout/)).toBeInTheDocument();
        expect(goto).not.toHaveBeenCalled();
    });

    it("fills the bar while the provider tells nothing of its progress", async () => {
        api.retrieveZone.mockReturnValue(new Promise(() => {}));
        api.getRetrieveZoneProgress.mockResolvedValue({
            step: "retrieving",
            percent: 0,
            step_end: 80,
        } as RetrieveZoneProgress);

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(1000);

        expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "100");
    });

    it("goes to the zone imported, by the name of its domain", async () => {
        api.retrieveZone.mockResolvedValue(undefined);
        api.getRetrieveZoneProgress.mockResolvedValue(null);
        api.refreshDomains.mockImplementation(async () => {
            domains.set([{ ...domain, zone_history: ["z1"] } as never]);
        });

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(0);

        expect(api.refreshDomains).toHaveBeenCalled();
        expect(goto).toHaveBeenCalledWith(expect.stringContaining("/domains/example.com."), {
            replaceState: true,
        });
    });

    it("goes to the zone imported by the id of a domain it does not know the name of", async () => {
        api.retrieveZone.mockResolvedValue(undefined);
        api.getRetrieveZoneProgress.mockResolvedValue(null);
        api.refreshDomains.mockResolvedValue(undefined);

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(0);

        expect(goto).toHaveBeenCalledWith(expect.stringContaining("/domains/d1"), {
            replaceState: true,
        });
    });

    it("creeps towards the end of the step, without reaching it nor going back", async () => {
        api.retrieveZone.mockReturnValue(new Promise(() => {}));
        api.getRetrieveZoneProgress.mockResolvedValue(exploring);

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(1000);
        const bar = screen.getByRole("progressbar");
        const first = Number(bar.getAttribute("aria-valuenow"));

        await vi.advanceTimersByTimeAsync(60000);
        const later = Number(bar.getAttribute("aria-valuenow"));
        expect(later).toBeGreaterThan(first);
        expect(later).toBeGreaterThan(Number(exploring.percent));
        expect(later).toBeLessThan(Number(exploring.step_end));

        // The server telling less than before does not move the bar back.
        api.getRetrieveZoneProgress.mockResolvedValue({ ...exploring, percent: 10 });
        await vi.advanceTimersByTimeAsync(1000);
        expect(Number(bar.getAttribute("aria-valuenow"))).toBeGreaterThanOrEqual(later);
    });

    it("does not creep for a user preferring reduced motion", async () => {
        vi.stubGlobal("matchMedia", () => ({ matches: true }));
        api.retrieveZone.mockReturnValue(new Promise(() => {}));
        api.getRetrieveZoneProgress.mockResolvedValue(exploring);

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(60000);

        expect(screen.getByRole("progressbar")).toHaveAttribute(
            "aria-valuenow",
            String(exploring.percent),
        );
        vi.stubGlobal("matchMedia", () => ({ matches: false }));
    });

    it("tells the step, the names, the records, the queries and the time elapsed", async () => {
        api.retrieveZone.mockReturnValue(new Promise(() => {}));
        api.getRetrieveZoneProgress.mockResolvedValue({
            step: "collecting",
            source: "wordlist",
            percent: 20,
            step_end: 30,
            done: 3,
            total: 10,
            records: 5,
            queries: 12,
            started_at: new Date(Date.now() - 6000).toISOString(),
        } as RetrieveZoneProgress);

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(1000);

        expect(screen.getByText("Collecting names to look for (wordlist)…")).toBeInTheDocument();
        expect(screen.getByText(/3 of 10 names asked/)).toBeInTheDocument();
        expect(screen.getByText(/5 records found/)).toBeInTheDocument();
        expect(screen.getByText(/\(12 queries\)/)).toBeInTheDocument();
        expect(screen.getByText("7 s elapsed")).toBeInTheDocument();
    });

    it("leaves out the names and the queries not counted yet", async () => {
        api.retrieveZone.mockReturnValue(new Promise(() => {}));
        api.getRetrieveZoneProgress.mockResolvedValue(exploring);

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(1000);

        expect(screen.getByText(/no record found yet/)).toBeInTheDocument();
        expect(screen.queryByText(/names asked/)).toBeNull();
        expect(screen.queryByText(/quer/)).toBeNull();
    });

    it("goes on polling after a poll failed", async () => {
        vi.spyOn(console, "warn").mockImplementation(() => {});
        api.retrieveZone.mockReturnValue(new Promise(() => {}));
        api.getRetrieveZoneProgress
            .mockRejectedValueOnce(new Error("network down"))
            .mockResolvedValue(exploring);

        render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(1000);
        expect(screen.queryByText(/network down/)).toBeNull();
        expect(screen.queryByRole("progressbar")).toBeNull();

        await vi.advanceTimersByTimeAsync(1000);
        await waitFor(() => expect(screen.getByRole("progressbar")).toBeInTheDocument());
    });

    it("stops polling once left", async () => {
        api.retrieveZone.mockReturnValue(new Promise(() => {}));
        api.getRetrieveZoneProgress.mockResolvedValue(exploring);

        const { unmount } = render(Page, { data: { domain } });
        await vi.advanceTimersByTimeAsync(1000);
        unmount();
        api.getRetrieveZoneProgress.mockClear();

        await vi.advanceTimersByTimeAsync(5000);
        expect(api.getRetrieveZoneProgress).not.toHaveBeenCalled();
    });
});

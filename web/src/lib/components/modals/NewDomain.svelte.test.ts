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

import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

// The network lives here: the tests decide what the backend answers.
const api = vi.hoisted(() => ({
    updateProvider: vi.fn(),
    refreshProviders: vi.fn(),
    refreshProvidersSpecs: vi.fn(),
}));
vi.mock("$lib/api/provider", () => ({ updateProvider: api.updateProvider }));
vi.mock("$lib/api/provider_specs", () => ({
    listProviders: vi.fn(async () => ({
        DDNSServer: { name: "Dynamic DNS", description: "RFC 2136" },
    })),
}));
vi.mock("$lib/api/provider_settings", () => ({
    getProviderSettings: vi.fn(() => new Promise(() => {})),
}));
vi.mock("$lib/stores/providers", async (importOriginal) => ({
    ...(await importOriginal<typeof import("$lib/stores/providers")>()),
    refreshProviders: api.refreshProviders,
    refreshProvidersSpecs: api.refreshProvidersSpecs,
}));

import type { Provider } from "$lib/model/provider";
import { providers } from "$lib/stores/providers";
import { toasts } from "$lib/stores/toasts";
import { loadTranslations } from "$lib/translations";
import NewDomain, { controls } from "./NewDomain.svelte";

const ddns = { _id: "p1", _srctype: "DDNSServer", _comment: "Mine" } as unknown as Provider;
const discovery = {
    _id: "p2",
    _srctype: "PublicDNSDiscovery",
    _comment: "Public DNS",
} as unknown as Provider;

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    vi.resetAllMocks();
    vi.restoreAllMocks();
    providers.set(undefined);
});

// monitorElsewhere opens the dialog and clicks to monitor a domain hosted
// elsewhere, returning the provider the dialog went on with.
async function monitorElsewhere(): Promise<ReturnType<typeof vi.fn>> {
    const ondone = vi.fn();
    render(NewDomain);
    controls.Open({ ondone });
    await userEvent.click(
        await screen.findByRole("button", { name: /Monitor a domain hosted elsewhere/ }),
    );
    return ondone;
}

describe("NewDomain", () => {
    it("goes on with the discovery provider the user already has", async () => {
        providers.set([ddns, discovery]);

        const ondone = await monitorElsewhere();

        await vi.waitFor(() => expect(ondone).toHaveBeenCalledWith(discovery));
        expect(api.updateProvider).not.toHaveBeenCalled();
    });

    it("waits for the providers before looking for the discovery one", async () => {
        api.refreshProviders.mockImplementation(async () => providers.set([ddns, discovery]));

        const ondone = await monitorElsewhere();

        await vi.waitFor(() => expect(ondone).toHaveBeenCalledWith(discovery));
        expect(api.updateProvider).not.toHaveBeenCalled();
    });

    it("no longer offers to monitor a domain elsewhere once a type is picked", async () => {
        providers.set([]);
        render(NewDomain);
        controls.Open({});

        await userEvent.click(await screen.findByRole("button", { name: /Dynamic DNS/ }));

        await vi.waitFor(() =>
            expect(
                screen.queryByRole("button", { name: /Monitor a domain hosted elsewhere/ }),
            ).toBeNull(),
        );
    });

    it("creates the discovery provider on first use", async () => {
        providers.set([ddns]);
        api.updateProvider.mockResolvedValue(discovery);

        const ondone = await monitorElsewhere();

        await vi.waitFor(() => expect(ondone).toHaveBeenCalledWith(discovery));
        expect(api.updateProvider).toHaveBeenCalledTimes(1);
        expect(api.updateProvider).toHaveBeenCalledWith(
            expect.objectContaining({ _srctype: "PublicDNSDiscovery", Provider: {} }),
        );
        expect(api.refreshProviders).toHaveBeenCalled();
    });

    it("tells the failure to create the discovery provider", async () => {
        providers.set([ddns]);
        api.updateProvider.mockRejectedValue(new Error("storage down"));
        const toast = vi.spyOn(toasts, "addErrorToast");

        const ondone = await monitorElsewhere();

        await vi.waitFor(() =>
            expect(toast).toHaveBeenCalledWith(
                expect.objectContaining({ message: "storage down" }),
            ),
        );
        expect(ondone).not.toHaveBeenCalled();
    });

    describe("with a suggestion", () => {
        const ovh = { _id: "p3", _srctype: "OVHAPI", _comment: "My OVH" } as unknown as Provider;
        const suggestion = { nameservers: ["ns103.ovh.net.", "dns103.ovh.net."], providers: ["OVHAPI"] };

        it("only offers the accounts of the suggested provider, and tells why", async () => {
            providers.set([ddns, ovh]);
            const ondone = vi.fn();
            render(NewDomain);
            controls.Open({ ondone, suggestion });

            expect(await screen.findByText(/ns103\.ovh\.net, dns103\.ovh\.net/)).toBeInTheDocument();
            expect(screen.queryByRole("button", { name: /Mine/ })).toBeNull();
            expect(
                screen.getByRole("button", { name: /Monitor a domain hosted elsewhere/ }),
            ).toBeInTheDocument();

            await userEvent.click(screen.getByRole("button", { name: /My OVH/ }));
            expect(ondone).toHaveBeenCalledWith(ovh);
        });

        it("offers the other accounts on demand", async () => {
            providers.set([ddns, ovh]);
            render(NewDomain);
            controls.Open({ suggestion });

            await userEvent.click(await screen.findByRole("button", { name: /Use another provider/ }));

            expect(screen.getByRole("button", { name: /Mine/ })).toBeInTheDocument();
            expect(screen.getByRole("button", { name: /My OVH/ })).toBeInTheDocument();
            expect(screen.queryByText(/ns103\.ovh\.net/)).toBeNull();
        });

        it("offers to connect the suggested provider to a user without account", async () => {
            providers.set([]);
            render(NewDomain);
            controls.Open({ suggestion });

            await userEvent.click(
                await screen.findByRole("button", { name: /Connect a new OVHAPI account/ }),
            );

            await vi.waitFor(() =>
                expect(
                    screen.queryByRole("button", { name: /Connect a new OVHAPI account/ }),
                ).toBeNull(),
            );
            // Straight to its settings: not among all the types.
            expect(screen.queryByRole("button", { name: /Dynamic DNS/ })).toBeNull();
        });

        it("lets a user without account connect any other provider", async () => {
            providers.set([]);
            render(NewDomain);
            controls.Open({ suggestion });

            await userEvent.click(await screen.findByRole("button", { name: /Use another provider/ }));

            expect(await screen.findByRole("button", { name: /Dynamic DNS/ })).toBeInTheDocument();
        });

        it("tells which account refused the domain", async () => {
            providers.set([ovh]);
            render(NewDomain);
            controls.Open({ suggestion, failure: { provider: ovh, message: "no such zone" } });

            expect(await screen.findByRole("alert")).toHaveTextContent(/“My OVH”: no such zone/);
        });

        it("forgets the suggestion the next time it opens", async () => {
            providers.set([ddns, ovh]);
            render(NewDomain);
            controls.Open({ suggestion, failure: { provider: ovh, message: "no such zone" } });
            await screen.findByRole("alert");
            controls.Open();

            expect(await screen.findByRole("button", { name: /Mine/ })).toBeInTheDocument();
            expect(screen.queryByRole("alert")).toBeNull();
        });
    });
});

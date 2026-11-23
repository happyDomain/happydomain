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

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({
    addDomain: vi.fn(),
    getProviderSuggestion: vi.fn(),
    open: vi.fn(),
    refreshProviders: vi.fn(),
}));
vi.mock("$lib/api/domains", () => ({ addDomain: api.addDomain }));
vi.mock("$lib/api/provider_suggestion", () => ({
    getProviderSuggestion: api.getProviderSuggestion,
}));
vi.mock("$lib/components/modals/NewDomain.svelte", () => ({ controls: { Open: api.open } }));
vi.mock("$lib/stores/providers", async () => {
    const { writable } = await import("svelte/store");
    return { providers: writable(undefined), refreshProviders: api.refreshProviders };
});

import type { HappydnsKeystoreLock } from "$lib/api-base/types.gen";
import { KeystoreLockedError, NotAuthorizedError } from "$lib/hey-api";
import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import { appConfig } from "$lib/stores/config";
import { providers } from "$lib/stores/providers";
import {
    addDomainSomewhere,
    addDomainToOnly,
    directProvider,
    matchingProviders,
    type DomainAddition,
} from "./provider_suggestion";

const account = (_id: string, _srctype: string) => ({ _id, _srctype }) as Provider;
const ovh = account("ovh1", "OVHAPI");
const ovh2 = account("ovh2", "OVHAPI");
const gandi = account("gandi", "GandiAPI");
const netlify = account("netlify", "NetlifyAPI");

const domain = { domain: "example.com." } as Domain;

function suggest(...types: string[]) {
    const suggestion = { nameservers: ["ns1.example.net."], providers: types };
    api.getProviderSuggestion.mockResolvedValue(suggestion);
    return suggestion;
}

let added: ReturnType<typeof vi.fn<DomainAddition["added"]>>;
let busy: ReturnType<typeof vi.fn<(busy: boolean) => void>>;
let addition: DomainAddition;

beforeEach(() => {
    added = vi.fn<DomainAddition["added"]>();
    busy = vi.fn<(busy: boolean) => void>();
    addition = { domain: "example.com", added, busy };
    api.addDomain.mockResolvedValue(domain);
});

afterEach(() => {
    // Whatever the outcome, the form does not stay stuck.
    if (busy.mock.calls.length) expect(busy).toHaveBeenLastCalledWith(false);
    vi.resetAllMocks();
    providers.set(undefined);
    appConfig.set({});
});

describe("matchingProviders", () => {
    it("keeps the accounts of the given types", () => {
        expect(matchingProviders([ovh, gandi, ovh2], ["OVHAPI"])).toEqual([ovh, ovh2]);
        expect(matchingProviders([ovh, gandi], [])).toEqual([]);
    });
});

describe("directProvider", () => {
    it("needs a single type and a single account", () => {
        expect(directProvider([gandi, ovh], ["OVHAPI"])).toBe(ovh);
        expect(directProvider([ovh, ovh2], ["OVHAPI"])).toBeUndefined();
        expect(directProvider([gandi], ["OVHAPI"])).toBeUndefined();
        expect(directProvider([ovh], [])).toBeUndefined();
    });

    it("does not choose between types, even with a single account", () => {
        expect(directProvider([netlify], ["NS1API", "NetlifyAPI"])).toBeUndefined();
    });
});

describe("addDomainSomewhere", () => {
    it("adds the domain to the only account of the only provider", async () => {
        suggest("OVHAPI");
        providers.set([gandi, ovh]);

        await addDomainSomewhere(addition);

        expect(api.addDomain).toHaveBeenCalledExactlyOnceWith("example.com", ovh);
        expect(added).toHaveBeenCalledExactlyOnceWith(domain);
        expect(api.open).not.toHaveBeenCalled();
    });

    it("loads the accounts when they are not there yet", async () => {
        suggest("OVHAPI");
        api.refreshProviders.mockResolvedValue([ovh]);

        await addDomainSomewhere(addition);

        expect(api.addDomain).toHaveBeenCalledWith("example.com", ovh);
    });

    it("asks when the accounts cannot be loaded", async () => {
        const suggestion = suggest("OVHAPI");
        api.refreshProviders.mockRejectedValue(new Error("down"));

        await addDomainSomewhere(addition);

        expect(api.open).toHaveBeenCalledWith({ ondone: expect.any(Function), suggestion });
    });

    it("asks, telling why, when the account refuses the domain", async () => {
        const suggestion = suggest("OVHAPI");
        providers.set([ovh]);
        api.addDomain.mockRejectedValue(new Error("no such zone"));

        await addDomainSomewhere(addition);

        expect(added).not.toHaveBeenCalled();
        expect(api.open).toHaveBeenCalledExactlyOnceWith({
            ondone: expect.any(Function),
            suggestion,
            failure: { provider: ovh, message: "no such zone" },
        });
    });

    it.each([
        ["no account", ["OVHAPI"], [gandi]],
        ["several accounts", ["OVHAPI"], [ovh, ovh2]],
        ["several types", ["NS1API", "NetlifyAPI"], [netlify]],
    ])("asks among the suggestions with %s", async (_, types, accounts) => {
        const suggestion = suggest(...types);
        providers.set(accounts);

        await addDomainSomewhere(addition);

        expect(api.addDomain).not.toHaveBeenCalled();
        expect(api.open).toHaveBeenCalledExactlyOnceWith({
            ondone: expect.any(Function),
            suggestion,
        });
    });

    it("asks without suggestion when no provider is recognized", async () => {
        suggest();
        providers.set([ovh]);

        await addDomainSomewhere(addition);

        expect(api.addDomain).not.toHaveBeenCalled();
        expect(api.open).toHaveBeenCalledExactlyOnceWith({ ondone: expect.any(Function) });
    });

    it("asks without suggestion when the lookup fails", async () => {
        api.getProviderSuggestion.mockRejectedValue(new Error("timeout"));

        await addDomainSomewhere(addition);

        expect(api.open).toHaveBeenCalledExactlyOnceWith({ ondone: expect.any(Function) });
    });

    it("looks nothing up when disabled", async () => {
        appConfig.set({ disable_provider_suggestion: true });
        providers.set([ovh]);

        await addDomainSomewhere(addition);

        expect(api.getProviderSuggestion).not.toHaveBeenCalled();
        expect(api.open).toHaveBeenCalledExactlyOnceWith({ ondone: expect.any(Function) });
    });

    it("never adds by itself without auto", async () => {
        const suggestion = suggest("OVHAPI");
        providers.set([ovh]);

        await addDomainSomewhere(addition, { auto: false });

        expect(api.addDomain).not.toHaveBeenCalled();
        expect(api.open).toHaveBeenCalledExactlyOnceWith({
            ondone: expect.any(Function),
            suggestion,
        });
    });

    it.each([
        new NotAuthorizedError("expired"),
        new KeystoreLockedError("locked", {} as HappydnsKeystoreLock),
    ])("lets a $name through", async (error) => {
        suggest("OVHAPI");
        providers.set([ovh]);
        api.addDomain.mockRejectedValue(error);

        await expect(addDomainSomewhere(addition)).rejects.toBe(error);
        expect(api.open).not.toHaveBeenCalled();
    });

    it("does not offer to add again a domain already added", async () => {
        suggest("OVHAPI");
        providers.set([ovh]);
        added.mockRejectedValue(new Error("unable to refresh"));

        await expect(addDomainSomewhere(addition)).rejects.toThrow("unable to refresh");
        expect(api.open).not.toHaveBeenCalled();
    });

    it("adds the domain to the account picked in the dialog", async () => {
        suggest("OVHAPI");
        providers.set([ovh, ovh2]);

        await addDomainSomewhere(addition);
        await api.open.mock.calls[0][0].ondone(ovh2);

        expect(api.addDomain).toHaveBeenCalledExactlyOnceWith("example.com", ovh2);
        expect(added).toHaveBeenCalledExactlyOnceWith(domain);
    });
});

describe("addDomainToOnly", () => {
    it("looks nothing up when the account accepts the domain", async () => {
        await addDomainToOnly(addition, gandi);

        expect(api.addDomain).toHaveBeenCalledExactlyOnceWith("example.com", gandi);
        expect(added).toHaveBeenCalledExactlyOnceWith(domain);
        expect(api.getProviderSuggestion).not.toHaveBeenCalled();
    });

    it("suggests the provider when the account refuses the domain", async () => {
        const suggestion = suggest("OVHAPI");
        api.addDomain.mockRejectedValue(new Error("no such zone"));

        await addDomainToOnly(addition, gandi);

        // Not tried a second time, even when it is the suggested one.
        expect(api.addDomain).toHaveBeenCalledTimes(1);
        expect(api.open).toHaveBeenCalledExactlyOnceWith({
            ondone: expect.any(Function),
            suggestion,
            failure: { provider: gandi, message: "no such zone" },
        });
    });

    it("reports the refusal when there is nothing to suggest", async () => {
        suggest();
        api.addDomain.mockRejectedValue(new Error("no such zone"));

        await expect(addDomainToOnly(addition, gandi)).rejects.toThrow("no such zone");
        expect(api.open).not.toHaveBeenCalled();
    });

    it("reports the refusal when disabled", async () => {
        appConfig.set({ disable_provider_suggestion: true });
        api.addDomain.mockRejectedValue(new Error("no such zone"));

        await expect(addDomainToOnly(addition, gandi)).rejects.toThrow("no such zone");
        expect(api.getProviderSuggestion).not.toHaveBeenCalled();
    });
});

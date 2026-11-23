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

import { get } from "svelte/store";

import { addDomain } from "$lib/api/domains";
import { getProviderSuggestion, type ProviderSuggestion } from "$lib/api/provider_suggestion";
import { controls as newDomainControls } from "$lib/components/modals/NewDomain.svelte";
import { KeystoreLockedError, NotAuthorizedError } from "$lib/hey-api";
import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import { appConfig } from "$lib/stores/config";
import { providers, refreshProviders } from "$lib/stores/providers";

// The backend gives up before: this only guards against it not answering.
const suggestionTimeout = 5000;

/** A domain being added. */
export interface DomainAddition {
    domain: string;
    /** Called once the domain has been added. */
    added: (domain: Domain) => void | Promise<void>;
    /** Told when happyDomain starts and stops working on the addition. */
    busy?: (busy: boolean) => void;
}

/** The accounts among accounts that belong to one of the given provider types. */
export function matchingProviders(accounts: Provider[], types: string[]): Provider[] {
    return accounts.filter((account) => types.includes(account._srctype));
}

/**
 * The account to add the domain to without asking: there has to be no doubt
 * about the provider, nor about the account.
 */
export function directProvider(accounts: Provider[], types: string[]): Provider | undefined {
    if (types.length !== 1) return undefined;

    const matching = matchingProviders(accounts, types);
    return matching.length === 1 ? matching[0] : undefined;
}

/**
 * Looks for the providers hosting the domain. Finding none, for whatever
 * reason, leaves the user choose by themself: it is never an error.
 */
async function suggestProviders(domain: string): Promise<ProviderSuggestion | undefined> {
    if (get(appConfig).disable_provider_suggestion) return undefined;

    try {
        const suggestion = await getProviderSuggestion(
            domain,
            AbortSignal.timeout(suggestionTimeout),
        );
        return suggestion.providers.length ? suggestion : undefined;
    } catch {
        return undefined;
    }
}

// Tells the errors about the account from those about the session of the user
// or the keystore they left locked: another account would not help.
function isAboutAccount(err: unknown): boolean {
    return !(err instanceof NotAuthorizedError || err instanceof KeystoreLockedError);
}

/** Adds the domain to the given account. */
export async function addDomainTo(addition: DomainAddition, provider: Provider): Promise<void> {
    addition.busy?.(true);
    try {
        await addition.added(await addDomain(addition.domain, provider));
    } finally {
        addition.busy?.(false);
    }
}

function openDialog(
    addition: DomainAddition,
    suggestion?: ProviderSuggestion,
    failure?: { provider: Provider; message: string },
): void {
    newDomainControls.Open({
        ondone: (provider) => addDomainTo(addition, provider),
        ...(suggestion ? { suggestion } : {}),
        ...(failure ? { failure } : {}),
    });
}

// tryProvider adds the domain to an account the user did not choose. When it
// is refused, it returns the error: only this is a reason to ask the user, not
// a failure coming after the domain has been added.
async function tryProvider(
    addition: DomainAddition,
    provider: Provider,
): Promise<{ error: unknown } | undefined> {
    let domain: Domain;
    try {
        domain = await addDomain(addition.domain, provider);
    } catch (error) {
        if (!isAboutAccount(error)) throw error;
        return { error };
    }
    await addition.added(domain);
}

function failureOf(provider: Provider, error: unknown) {
    return { provider, message: error instanceof Error ? error.message : String(error) };
}

/**
 * Adds a domain the user gave no account for: to the account its name servers
 * designate when there is no doubt, and to the one the user picks in the
 * dialog otherwise.
 *
 * Without auto, the user always picks the account.
 */
export async function addDomainSomewhere(
    addition: DomainAddition,
    { auto = true }: { auto?: boolean } = {},
): Promise<void> {
    addition.busy?.(true);
    try {
        const suggestion = await suggestProviders(addition.domain);

        if (suggestion && auto) {
            const accounts = get(providers) ?? (await refreshProviders().catch(() => []));
            const provider = directProvider(accounts, suggestion.providers);
            if (provider) {
                const refused = await tryProvider(addition, provider);
                if (refused) openDialog(addition, suggestion, failureOf(provider, refused.error));
                return;
            }
        }

        openDialog(addition, suggestion);
    } finally {
        addition.busy?.(false);
    }
}

/**
 * Adds a domain to the only account of the user. When it is refused, the name
 * servers may tell which account is missing.
 */
export async function addDomainToOnly(addition: DomainAddition, provider: Provider): Promise<void> {
    addition.busy?.(true);
    try {
        const refused = await tryProvider(addition, provider);
        if (!refused) return;

        const suggestion = await suggestProviders(addition.domain);
        if (!suggestion) throw refused.error;
        openDialog(addition, suggestion, failureOf(provider, refused.error));
    } finally {
        addition.busy?.(false);
    }
}

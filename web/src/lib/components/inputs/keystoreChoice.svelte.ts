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

import { getContext, setContext } from "svelte";

import { refreshKeystores } from "$lib/stores/keystores";

const key = Symbol("keystore-choice");

/**
 * The keystore chosen for the secrets of the object a form edits, shared by
 * every secret field of the form: all the secrets of an object go to the same
 * keystore. An empty value chooses none: the server keeps the secrets where
 * they are, or uses the default keystore for a new object.
 */
export class KeystoreChoice {
    value = $state("");

    // Empty for an object not stored yet.
    objectId: string | undefined = $state(undefined);

    // Bumped when the choice goes from a keystore keeping references to one
    // keeping secrets, or back: what the secret fields hold no longer means
    // the same, so each of them empties itself.
    resetSecrets = $state(0);

    // A keystore the secrets cannot go to: the one a keystore keeping a
    // secret of its own opens.
    exclude: string | undefined = $state(undefined);

    constructor(
        readonly objectType: string,
        objectId: string | undefined,
        // The object is used in the background, without its owner: only the
        // keystores opening without them fit.
        readonly unattended: boolean,
    ) {
        this.objectId = objectId;
    }
}

/**
 * Shares choice with the secret fields of the form being created, and loads
 * the keystores afresh: the form shows where the secrets are now.
 */
export function provideKeystoreChoice(choice: KeystoreChoice): KeystoreChoice {
    setContext(key, choice);
    refreshKeystores().catch(() => {});
    return choice;
}

export function useKeystoreChoice(): KeystoreChoice | undefined {
    return getContext<KeystoreChoice | undefined>(key);
}

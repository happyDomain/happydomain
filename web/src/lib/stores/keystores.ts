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

import { listKeystores, type Keystore } from "$lib/api/keystores";
import { writable, type Writable } from "svelte/store";

export const keystores: Writable<Array<Keystore> | undefined> = writable(undefined);

export async function refreshKeystores(): Promise<Array<Keystore>> {
    const data = await listKeystores();
    keystores.set(data);
    return data;
}

/**
 * Keystores the secrets of an object can go to: available ones, and, for an
 * object used in the background (notification channel, checker options), only
 * those opening without their owner.
 */
export function eligibleKeystores(list: Array<Keystore>, unattended: boolean): Array<Keystore> {
    return list.filter((ks) => ks.available && (!unattended || ks.unattended));
}

/** The keystore an object already uses, if any. */
export function keystoreOf(
    list: Array<Keystore>,
    objectType: string,
    objectId: string | undefined,
): Keystore | undefined {
    if (!objectId) return undefined;
    return list.find((ks) =>
        ks.objects?.some((o) => o.objectType === objectType && o.objectId === objectId),
    );
}

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

import {
    deleteKeystoresByRef,
    getKeystores,
    getKeystoresKinds,
    postKeystores,
    postKeystoresByRefLock,
    postKeystoresByRefMigrate,
    postKeystoresByRefUnlock,
    putKeystoresByRef,
    putKeystoresDefault,
} from "$lib/api-base/sdk.gen";
import type {
    SecretKeystore,
    SecretKindInfo,
    SecretResealReport,
    UsecaseKeystoreUsage,
} from "$lib/api-base/types.gen";
import { unwrapEmptyResponse, unwrapSdkResponse } from "./errors";

export type Keystore = UsecaseKeystoreUsage;
export type KeystoreKind = SecretKindInfo;

export async function listKeystores(): Promise<Array<Keystore>> {
    return unwrapSdkResponse(await getKeystores()) as Array<Keystore>;
}

export async function listKeystoreKinds(): Promise<Array<KeystoreKind>> {
    return unwrapSdkResponse(await getKeystoresKinds()) as Array<KeystoreKind>;
}

export async function createKeystore(
    kind: string,
    name: string,
    params: Record<string, unknown>,
    // Where the secret among params goes, for a keystore keeping one.
    keystore = "",
): Promise<SecretKeystore> {
    return unwrapSdkResponse(
        await postKeystores({ body: { kind, name, params, keystore: keystore || undefined } }),
    ) as SecretKeystore;
}

/** Updates a personal keystore with the params its update form asks for. */
export async function updateKeystore(
    ref: string,
    params: Record<string, unknown>,
    // Where the secret among params goes, for a keystore keeping one.
    keystore = "",
): Promise<SecretKeystore> {
    return unwrapSdkResponse(
        await putKeystoresByRef({
            path: { ref },
            body: { params, keystore: keystore || undefined },
        }),
    ) as SecretKeystore;
}

export async function deleteKeystore(ref: string): Promise<boolean> {
    return unwrapEmptyResponse(await deleteKeystoresByRef({ path: { ref } }));
}

export async function migrateKeystore(ref: string, to: string): Promise<Array<SecretResealReport>> {
    return unwrapSdkResponse(
        await postKeystoresByRefMigrate({ path: { ref }, body: { to } }),
    ) as Array<SecretResealReport>;
}

/** Choose the keystore new objects go to; an empty ref goes back to the policy. */
export async function setDefaultKeystore(ref: string): Promise<boolean> {
    return unwrapEmptyResponse(await putKeystoresDefault({ body: { ref } }));
}

/** Opens a keystore for the rest of the session, with what its unlock form asks. */
export async function unlockKeystore(
    ref: string,
    material: Record<string, unknown>,
): Promise<boolean> {
    return unwrapEmptyResponse(
        await postKeystoresByRefUnlock({ path: { ref }, body: { material } }),
    );
}

export async function lockKeystore(ref: string): Promise<boolean> {
    return unwrapEmptyResponse(await postKeystoresByRefLock({ path: { ref } }));
}

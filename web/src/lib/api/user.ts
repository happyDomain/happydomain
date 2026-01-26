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
    postUsers,
    postAuthLogin,
    postAuthLogout,
    patchUsers,
    postUsersByUserIdRecovery,
    postUsersByUserIdEmail,
    postUsersByUserIdNewPassword,
    deleteUsersByUserId,
    postUsersByUserIdDelete,
    postUsersByUserIdSettings,
    getUsersByUserId,
} from "$lib/api-base/sdk.gen";
import type { UserSettings } from "$lib/model/usersettings";
import type { User, SignUpForm, LoginForm } from "$lib/model/user";
import { unwrapSdkResponse, unwrapEmptyResponse } from "./errors";

export async function registerUser(form: SignUpForm): Promise<User> {
    return unwrapSdkResponse(await postUsers({ body: form })) as unknown as User;
}

export async function authUser(form: LoginForm): Promise<User> {
    return unwrapSdkResponse(await postAuthLogin({ body: form })) as unknown as User;
}

export async function logout(): Promise<boolean> {
    return unwrapEmptyResponse(await postAuthLogout());
}

export async function specialUserOperations(
    email: string,
    kind: "recovery" | "validation",
): Promise<{ errmsg: string }> {
    return unwrapSdkResponse(
        await patchUsers({
            body: {
                email,
                kind,
            },
        }),
    ) as { errmsg: string };
}

export function forgotAccountPassword(email: string): Promise<{ errmsg: string }> {
    return specialUserOperations(email, "recovery");
}

export async function resendValidationEmail(email: string): Promise<{ errmsg: string }> {
    return specialUserOperations(email, "validation");
}

export async function recoverAccount(
    userid: string,
    key: string,
    password: string,
): Promise<boolean> {
    return unwrapEmptyResponse(
        await postUsersByUserIdRecovery({
            path: { userId: userid },
            body: {
                key,
                password,
            },
        }),
    );
}

export async function validateEmail(userid: string, key: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await postUsersByUserIdEmail({
            path: { userId: userid },
            body: {
                key,
            },
        }),
    );
}

export async function changeUserPassword(
    user: User,
    form: { current: string; password: string; passwordconfirm: string },
): Promise<boolean> {
    return unwrapEmptyResponse(
        await postUsersByUserIdNewPassword({
            path: { userId: user.id },
            body: form,
        }),
    );
}

export async function deleteMyUser(user: User): Promise<boolean> {
    return unwrapEmptyResponse(
        await deleteUsersByUserId({
            path: { userId: user.id },
        }),
    );
}

export async function deleteUserAccount(user: User, password: string): Promise<boolean> {
    return unwrapEmptyResponse(
        await postUsersByUserIdDelete({
            path: { userId: user.id },
            body: {
                current: password,
            },
        }),
    );
}

export async function saveAccountSettings(
    user: User,
    settings: UserSettings,
): Promise<UserSettings> {
    return unwrapSdkResponse(
        await postUsersByUserIdSettings({
            path: { userId: user.id },
            body: settings,
        }),
    ) as UserSettings;
}

export function cleanUserSession(): void {
    for (const k of Object.keys(sessionStorage)) {
        if (k.indexOf("newprovider-") == 0) {
            sessionStorage.removeItem(k);
        }
    }
}

export async function getUser(id: string): Promise<User> {
    return unwrapSdkResponse(
        await getUsersByUserId({
            path: { userId: id },
        } as any),
    ) as unknown as User;
}

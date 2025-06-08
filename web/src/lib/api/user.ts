// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
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

import { handleEmptyApiResponse, handleApiResponse, handleAuthApiResponse } from "$lib/errors";
import type { UserSettings } from "$lib/model/usersettings";
import type { User, SignUpForm, LoginForm } from "$lib/model/user";

export async function registerUser(form: SignUpForm): Promise<User> {
    const res = await fetch("/api/users", {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify(form),
    });
    return await handleApiResponse<User>(res);
}

export async function authUser(form: LoginForm): Promise<User> {
    const res = await fetch("/api/auth", {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify(form),
    });
    return await handleAuthApiResponse<User>(res);
}

export async function logout(): Promise<boolean> {
    const res = await fetch("/api/auth/logout", {
        method: "POST",
        headers: { Accept: "application/json" },
    });
    return await handleEmptyApiResponse(res);
}

export async function specialUserOperations(
    email: string,
    kind: "recovery" | "validation",
): Promise<{ errmsg: string }> {
    const res = await fetch("/api/users", {
        method: "PATCH",
        headers: { Accept: "application/json" },
        body: JSON.stringify({
            email,
            kind,
        }),
    });
    return await handleApiResponse<{ errmsg: string }>(res);
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
    userid = encodeURIComponent(userid);
    const res = await fetch(`/api/users/${userid}/recovery`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify({
            key,
            password,
        }),
    });
    return await handleEmptyApiResponse(res);
}

export async function validateEmail(userid: string, key: string): Promise<boolean> {
    userid = encodeURIComponent(userid);
    const res = await fetch(`/api/users/${userid}/email`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify({
            key,
        }),
    });
    return await handleEmptyApiResponse(res);
}

export async function changeUserPassword(
    user: User,
    form: { current: string; password: string; passwordconfirm: string },
): Promise<boolean> {
    const userid = encodeURIComponent(user.id);
    const res = await fetch(`/api/users/${userid}/new_password`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify(form),
    });
    return await handleEmptyApiResponse(res);
}

export async function deleteMyUser(user: User): Promise<boolean> {
    const userid = encodeURIComponent(user.id);
    const res = await fetch(`/api/users/${userid}`, {
        method: "DELETE",
        headers: { Accept: "application/json" },
    });
    return await handleEmptyApiResponse(res);
}

export async function deleteUserAccount(user: User, password: string): Promise<boolean> {
    const userid = encodeURIComponent(user.id);
    const res = await fetch(`/api/users/${userid}/delete`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify({
            current: password,
        }),
    });
    return await handleEmptyApiResponse(res);
}

export async function saveAccountSettings(
    user: User,
    settings: UserSettings,
): Promise<UserSettings> {
    const userid = encodeURIComponent(user.id);
    const res = await fetch(`/api/users/${userid}/settings`, {
        method: "POST",
        headers: { Accept: "application/json" },
        body: JSON.stringify(settings),
    });
    return await handleApiResponse<UserSettings>(res);
}

export function cleanUserSession(): void {
    for (const k of Object.keys(sessionStorage)) {
        if (k.indexOf("newprovider-") == 0) {
            sessionStorage.removeItem(k);
        }
    }
}

export async function getUser(id: string): Promise<User> {
    id = encodeURIComponent(id);
    const res = await fetch(`/api/users/${id}`, { headers: { Accept: "application/json" } });
    return await handleApiResponse<User>(res);
}

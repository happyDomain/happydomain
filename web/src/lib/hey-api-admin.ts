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

import { goto } from '$app/navigation';
import { authLinks } from '$links';

import type { CreateClientConfig } from './api-admin/client.gen';
import { clearAdminToken, getAdminToken } from '$lib/stores/adminsession';

export class NotAuthorizedError extends Error {
    constructor(message: string) {
        super(message);
        this.name = "NotAuthorizedError";
    }
}

async function customFetch(
    input: RequestInfo | URL,
    init?: RequestInit
): Promise<Response> {
    // Attach the admin bearer token (when authentication is enabled) to every
    // admin API call.
    const token = getAdminToken();
    if (token) {
        const headers = new Headers(init?.headers);
        headers.set("Authorization", "Bearer " + token);
        init = { ...init, headers };
    }

    const response = await fetch(input, init);

    if (response.status === 401) {
        clearAdminToken();
        const login = authLinks().login();
        if (typeof window !== "undefined" && window.location.pathname !== login) {
            // The login path is already resolved; the one handed over to it is
            // read from the address bar, base path included, so the login page
            // can send the user back to it as is.
            // eslint-disable-next-line svelte/no-navigation-without-resolve
            goto(login + "?next=" + encodeURIComponent(window.location.pathname));
        }
        throw new NotAuthorizedError("Admin authentication required.");
    }

    if (response.status === 400) {
        const json = await response.json();
        if (json.error === "error in openapi3filter.SecurityRequirementsError: security requirements failed: invalid session") {
            throw new NotAuthorizedError(json.error.substring(80));
        }
    }

    return response;
}


export const createClientConfig: CreateClientConfig = (config) => ({
    ...config,
    baseUrl: '/api/',
    fetch: customFetch,
});

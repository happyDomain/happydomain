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

import { refreshUserSession } from "$lib/stores/usersession";
import type { CreateClientConfig } from "./api-base/client.gen";

export class NotAuthorizedError extends Error {
    constructor(message: string) {
        super(message);
        this.name = "NotAuthorizedError";
    }
}

export class ProviderNoDomainListingSupport extends Error {
    constructor(message: string) {
        super(message);
        this.name = "ProviderNoDomainListingSupport";
    }
}

async function customFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
    const response = await fetch(input, init);

    // Handle 401 Unauthorized - attempt session refresh and retry
    if (response.status === 401) {
        try {
            await refreshUserSession();
            // Retry the original request after successful session refresh
            const retryResponse = await fetch(input, init);
            return retryResponse;
        } catch (err) {
            if (err instanceof Error) {
                throw new NotAuthorizedError(err.message);
            } else {
                throw new NotAuthorizedError("Session refresh failed");
            }
        }
    }

    // For error responses with JSON content, check for specific error cases
    // Clone BEFORE consuming to avoid "Body has already been consumed" error
    if (!response.ok && response.headers.get("content-type")?.includes("application/json")) {
        const clone = response.clone();
        try {
            const json = await clone.json();

            // Check for session validation errors (400 status)
            if (
                response.status === 400 &&
                json.error ===
                "error in openapi3filter.SecurityRequirementsError: security requirements failed: invalid session"
            ) {
                throw new NotAuthorizedError(json.error.substring(80));
            }

            // Check for specific provider errors
            if (json.errmsg) {
                if (json.errmsg === "the provider doesn't support domain listing") {
                    throw new ProviderNoDomainListingSupport(json.errmsg);
                }
            }
        } catch (err) {
            // If it's one of our custom errors, re-throw it
            if (err instanceof NotAuthorizedError || err instanceof ProviderNoDomainListingSupport) {
                throw err;
            }
            // Otherwise, ignore JSON parsing errors and return the original response
        }
    }

    return response;
}

export const createClientConfig: CreateClientConfig = (config) => {
    // In test environments (Node.js), we need a full URL with protocol and host
    // In browser environments, relative URLs work fine
    const baseUrl = typeof window !== 'undefined' ? "/api/" : "http://localhost/api/";

    return {
        ...config,
        baseUrl,
        fetch: customFetch,
    };
};

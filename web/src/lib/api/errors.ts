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

// Re-export error classes from hey-api.ts
export { NotAuthorizedError, ProviderNoDomainListingSupport } from "$lib/hey-api";

/**
 * SDK Response type (simplified)
 */
interface SdkResponse<T> {
    data?: T;
    error?: unknown;
    response?: Response;
    request?: Request;
}

/**
 * Unwraps an SDK response, extracting the data or throwing an error.
 * Handles 204 No Content responses by returning true for boolean types.
 *
 * @param result The SDK response object
 * @returns The unwrapped data
 * @throws Error if the response contains an error or no data
 */
export function unwrapSdkResponse<T>(result: SdkResponse<T>): T {
    // Check for errors first
    if (result.error !== undefined) {
        // If the error is an Error object, throw it
        if (result.error instanceof Error) {
            throw result.error;
        }
        // If the error has an errmsg field, throw an Error with that message
        if (typeof result.error === 'object' && result.error !== null && 'errmsg' in result.error) {
            throw new Error((result.error as { errmsg: string }).errmsg);
        }
        // Otherwise, throw a generic error
        throw new Error(String(result.error));
    }

    // Handle 204 No Content - return the data (which might be an empty object)
    if (result.response?.status === 204) {
        return result.data as T;
    }

    // Return data if it exists
    if (result.data !== undefined) {
        return result.data;
    }

    // No data and no error - this shouldn't happen
    throw new Error("SDK response contains neither data nor error");
}

/**
 * Unwraps an SDK response for operations that return empty/boolean responses.
 * Specifically handles 204 No Content responses by returning true.
 *
 * @param result The SDK response object
 * @returns true if successful, otherwise throws
 * @throws Error if the response contains an error
 */
export function unwrapEmptyResponse(result: SdkResponse<unknown>): boolean {
    // Check for errors first
    if (result.error !== undefined) {
        // If the error is an Error object, throw it
        if (result.error instanceof Error) {
            throw result.error;
        }
        // If the error has an errmsg field, throw an Error with that message
        if (typeof result.error === 'object' && result.error !== null && 'errmsg' in result.error) {
            throw new Error((result.error as { errmsg: string }).errmsg);
        }
        // Otherwise, throw a generic error
        throw new Error(String(result.error));
    }

    // Handle 204 No Content or any successful response
    if (result.response?.ok) {
        return true;
    }

    // Data exists and response was ok
    if (result.data !== undefined) {
        return true;
    }

    // No data and no error - this shouldn't happen
    throw new Error("SDK response contains neither data nor error");
}

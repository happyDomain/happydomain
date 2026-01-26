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

/**
 * Creates a mock Response object for testing API calls.
 * This is needed because the OpenAPI SDK expects proper Response objects with headers.
 *
 * @param data The data to return
 * @param status HTTP status code (default: 200)
 * @param statusText HTTP status text (default: "OK")
 * @returns A mock Response object
 */
export function createMockResponse(
    data: any,
    status: number = 200,
    statusText: string = "OK"
): Response {
    const headers = new Headers({
        'Content-Type': 'application/json',
    });

    return {
        ok: status >= 200 && status < 300,
        status,
        statusText,
        headers,
        json: () => Promise.resolve(data),
        text: () => Promise.resolve(JSON.stringify(data)),
        blob: () => Promise.resolve(new Blob([JSON.stringify(data)])),
        arrayBuffer: () => Promise.resolve(new ArrayBuffer(0)),
        clone: function() { return this; },
    } as Response;
}

/**
 * Creates a mock fetch function that returns the given response.
 *
 * @param data The data to return
 * @param status HTTP status code (default: 200)
 * @returns A mock fetch function
 */
export function createMockFetch(data: any, status: number = 200) {
    return () => Promise.resolve(createMockResponse(data, status));
}

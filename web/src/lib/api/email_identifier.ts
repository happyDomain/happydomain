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

import { customFetch } from "$lib/hey-api";
import { base } from "$app/paths";

/**
 * Ask the backend for the OPENPGPKEY/SMIMEA owner name prefix of an email
 * local-part.
 *
 * Only needed outside a secure context, where the browser hides the Web Crypto
 * API and the editors can't compute the hash themselves.
 */
export async function getEmailIdentifier(domainId: string, username: string): Promise<string> {
    // Goes through customFetch so an expired session is refreshed and retried,
    // like every other API call.
    const res = await customFetch(`${base}/api/domains/${domainId}/email-identifier`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username }),
    });

    if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.errmsg || `HTTP ${res.status}`);
    }

    return (await res.json()).identifier;
}

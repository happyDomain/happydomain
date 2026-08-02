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

import { resolve } from "$app/paths";

/**
 * The counterpart of web/src/links.ts.
 *
 * src/lib is shared between the two apps by symlink, and some of what it holds
 * links to pages that only the user-facing app has: a domain of its own, a
 * provider of its own. The admin reaches those through a user, so it answers
 * with the listing it does have. Nothing here is reached today, since the
 * components asking are not mounted in the admin, but the answers keep the
 * shared code honest in both.
 */
export function domainLinks() {
    return {
        zone: (_dn: string) => resolve("/domains"),
        history: (_dn: string) => resolve("/domains"),
        checks: (_dn: string) => resolve("/domains"),
        service: (_dn: string, _historyid: string, _subdomain: string, _serviceid: string) =>
            resolve("/domains"),
    };
}

export function providerLinks() {
    return {
        provider: (_prvid: string) => resolve("/providers"),
    };
}

export function authLinks() {
    return {
        // The admin has no login page of its own: it is reached authenticated.
        login: () => resolve("/"),
    };
}

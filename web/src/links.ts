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

import type { CheckerLinks } from "$lib/checker_links";

export function globalCheckerLinks(): CheckerLinks {
    return {
        list: resolve("/checkers"),
        checker: (checkerId) => resolve("/checkers/[checkerId]", { checkerId }),
        back: resolve("/"),
    };
}

export function domainCheckerLinks(dn: string): CheckerLinks {
    return {
        list: resolve("/domains/[dn]/checkers", { dn }),
        checker: (checkerId) => resolve("/domains/[dn]/checkers/[checkerId]", { dn, checkerId }),
        executions: (checkerId) =>
            resolve("/domains/[dn]/checkers/[checkerId]/executions", { dn, checkerId }),
        execution: (checkerId, execId) =>
            resolve("/domains/[dn]/checkers/[checkerId]/executions/[execId]", {
                dn,
                checkerId,
                execId,
            }),
        back: resolve("/domains/[dn]/[[historyid]]", { dn }),
    };
}

export function serviceCheckerLinks(
    dn: string,
    historyid: string,
    subdomain: string,
    serviceid: string,
): CheckerLinks {
    const params = { dn, historyid, subdomain, serviceid };
    return {
        list: resolve(
            "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checkers",
            params,
        ),
        checker: (checkerId) =>
            resolve("/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checkers/[checkerId]", {
                ...params,
                checkerId,
            }),
        executions: (checkerId) =>
            resolve(
                "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checkers/[checkerId]/executions",
                { ...params, checkerId },
            ),
        execution: (checkerId, execId) =>
            resolve(
                "/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]/checkers/[checkerId]/executions/[execId]",
                { ...params, checkerId, execId },
            ),
        back: resolve("/domains/[dn]/[[historyid]]/[subdomain]/[serviceid]", params),
    };
}

/**
 * Where the shared components link to. web-admin shares src/lib and answers
 * the same names with its own routes, so a component rendered in either app
 * links somewhere that exists there.
 */
export function domainLinks() {
    return {
        zone: (dn: string) => resolve("/domains/[dn]/[[historyid]]", { dn }),
        history: (dn: string) => resolve("/domains/[dn]/history", { dn }),
        checks: (dn: string) => resolve("/domains/[dn]/checks", { dn }),
    };
}

export function providerLinks() {
    return {
        provider: (prvid: string) => resolve("/providers/[prvid]", { prvid }),
    };
}

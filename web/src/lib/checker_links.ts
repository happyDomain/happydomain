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
import type { ResolvedPathname } from "$app/types";

/**
 * The same checker screens are reachable under three route families: global,
 * for one domain, and for one service. The components rendering them used to
 * take the common prefix as a string and append segments to it, which put the
 * shape of a route in the hands of whoever had no way of knowing it.
 *
 * They now take one of these instead: the route stays where it is known, and
 * every link it hands out is resolved against the base path and checked
 * against the route table.
 */
export interface CheckerLinks {
    /** The list of checkers. */
    list: ResolvedPathname;
    /** One checker, with its configuration and its last results. */
    checker(checkerId: string): ResolvedPathname;
    /**
     * The past executions of one checker. Absent in the global scope, which
     * has no route for them: the caller is expected not to offer the link.
     */
    executions?(checkerId: string): ResolvedPathname;
    /** One execution of one checker. Absent wherever executions is. */
    execution?(checkerId: string, execId: string): ResolvedPathname;
    /** What the sidebar goes back to when leaving the checkers. */
    back: ResolvedPathname;
}

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

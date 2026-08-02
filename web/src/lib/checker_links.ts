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

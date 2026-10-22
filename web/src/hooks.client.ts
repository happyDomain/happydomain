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

import type { HandleClientError } from "@sveltejs/kit";

export const handleError: HandleClientError = ({ error, status, message }) => {
    // Once a hook is set, SvelteKit no longer logs the error itself.
    console.error(error);

    // SvelteKit hides the cause of an unexpected error behind a generic
    // "Internal Error", which protects a server's secrets. Here the error
    // never left the browser: hiding it only keeps the user, and their bug
    // report, in the dark.
    //
    // A 404 from the router is the exception: its message repeats the whole
    // path, which may hold a domain name.
    if (status !== 404 && error instanceof Error && error.message) {
        return { message: error.message };
    }

    return { message };
};

// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
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

import { writable, type Writable } from "svelte/store";
import { getAuth } from "$lib/api-base/sdk.gen";
import { unwrapSdkResponse } from "$lib/api/errors";
import { setRefreshingSession } from "$lib/hey-api";
import type { User } from "$lib/model/user";

export const userSession: Writable<User> = writable({} as User);

export async function refreshUserSession() {
    setRefreshingSession(true);
    try {
        const user = unwrapSdkResponse(await getAuth()) as unknown as User;
        userSession.set(user);
        return user;
    } catch (err) {
        userSession.set({} as User);
        throw err;
    } finally {
        setRefreshingSession(false);
    }
}

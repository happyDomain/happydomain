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

import { get_store_value } from "svelte/internal";
import { writable, type Writable } from "svelte/store";
import { getUser as APIGetUser } from "$lib/api/user";
import type { User } from "$lib/model/user";
import { Mutex } from "$lib/model/mutex";

export const users: Writable<Record<string, User>> = writable({});

const mutex = new Mutex();

const requests: Record<string, Promise<User>> = {};

export async function getUser(id: string, force?: boolean) {
    let unlock = await mutex.lock();
    if (id in requests) {
        unlock();
        await requests[id];
        return get_store_value(users)[id];
    }

    const user = get_store_value(users)[id];
    if (user && !force) {
        unlock();
        return user;
    }

    requests[id] = APIGetUser(id);
    unlock();

    const data = await requests[id];
    const obj: Record<string, User> = {};
    obj[id] = data;
    users.update((u) => Object.assign(u, obj));
    delete requests[id];
    return data;
}

import { get_store_value } from 'svelte/internal';
import { writable, type Writable } from 'svelte/store';
import { getUser as APIGetUser } from '$lib/api/user';
import type { User } from '$lib/model/user';

export const users: Writable<Record<string, User>> = writable({ });

function Mutex() {
    let current = Promise.resolve();
    this.lock = () => {
        let _resolve;
        const p = new Promise(resolve => {
            _resolve = () => resolve();
        });
        const rv = current.then(() => _resolve);
        current = p;
        return rv;
    };
}

const mutex = new Mutex();

const requests: Record<string, Promise<User>> = { };

export async function getUser(id: string, force: bool) {
    let unlock = await mutex.lock();
    if (requests[id]) {
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
    const obj = { };
    obj[id] = data;
    users.update((u) => Object.assign(u, obj));
    delete requests[id];
    return data;
}

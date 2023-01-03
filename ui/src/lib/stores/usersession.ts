import { writable, type Writable } from 'svelte/store';
import type { User } from '$lib/model/user';

export const userSession: Writable<null | User> = writable(null);

export async function refreshUserSession(fetch: (input: RequestInfo | URL, init?: RequestInit | undefined) => Promise<Response> = window.fetch) {
    const res = await fetch('/api/auth', {headers: {'Accept': 'application/json'}})
    if (res.status == 200) {
        const user = await res.json() as User;
        userSession.set(user);
        return user
    } else {
        userSession.set(null);
        throw new Error((await res.json()).errmsg);
    }
}

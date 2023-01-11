import { redirect, type Load } from '@sveltejs/kit';
import { get_store_value } from 'svelte/internal';

import { userSession } from '$lib/stores/usersession';

export const load: Load = async({ parent }) => {
    const data = await parent();

    if (get_store_value(userSession) != null) {
        throw redirect(302, '/');
    }

    return data;
}

import { get_store_value } from 'svelte/internal';
import type { Load } from '@sveltejs/kit';

import { domains, refreshDomains } from '$lib/stores/domains';

export const load: Load = async({ parent, params }) => {
    const data = await parent();

    if (!get_store_value(domains)) await refreshDomains();

    return {
        domain: params.dn,
        ...data,
    }
}

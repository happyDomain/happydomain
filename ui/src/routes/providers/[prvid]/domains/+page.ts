import { redirect, type Load } from '@sveltejs/kit';

import { getProvider } from '$lib/api/provider';

export const load: Load = async({ params }) => {
    if (params.prvid == undefined) {
        throw redirect(302, '/providers/');
    }

    return {
        provider: await getProvider(params.prvid),
        provider_id: params.prvid,
    }
}

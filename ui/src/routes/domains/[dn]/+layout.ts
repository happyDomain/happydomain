import type { Load } from '@sveltejs/kit';

export const load: Load = async({ parent, params }) => {
    const data = await parent();

    return {
        domain: params.dn,
        ...data,
    }
}

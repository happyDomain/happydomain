import type { Load } from '@sveltejs/kit';

export const load: Load = async({ params, url }) => {
    const nsPrvId = url.searchParams.get("nsprvid");
    return {
        ptype: params.ptype,
        state: params.state?parseInt(params.state):0,
        providerId: nsPrvId?nsPrvId:undefined,
    }
}

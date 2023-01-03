import type { Load } from '@sveltejs/kit';

export const load: Load = async({ params }) => {
    return {
        domain: params.dn,
        history: params.historyid,
    }
}

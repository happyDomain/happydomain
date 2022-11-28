import type { Load } from '@sveltejs/kit';

export const load: Load = async({ params }) => {
    return {
        ...history.state,
        domain: params.domain,
    };
}

import type { Load } from '@sveltejs/kit';

export const load: Load = async({ parent }) => {
    const data = await parent();

    return {
        ...data,
        isHistoryPage: true,
    }
}

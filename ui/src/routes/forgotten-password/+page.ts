import type { Load } from '@sveltejs/kit';

export const load: Load = async({ url }: {url: URL}) => {
    const user = url.searchParams.get("u");
    const key = url.searchParams.get("k");

    return {
        user,
        key,
    };
}

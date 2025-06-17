import type { Load } from "@sveltejs/kit";
import { get } from "svelte/store";

import { providers, refreshProviders } from "$lib/stores/providers";

export const load: Load = async ({ parent }) => {
    const data = await parent();

    if (!get(providers)) await refreshProviders();

    return data;
};

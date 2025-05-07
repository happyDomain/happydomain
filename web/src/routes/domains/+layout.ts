import { get_store_value } from "svelte/internal";
import type { Load } from "@sveltejs/kit";

import { providers, refreshProviders } from "$lib/stores/providers";

export const load: Load = async ({ parent }) => {
    const data = await parent();

    if (!get_store_value(providers)) await refreshProviders();

    return data;
};

import type { Load } from "@sveltejs/kit";
import { get } from "svelte/store";

import { domains, refreshDomains } from "$lib/stores/domains";
import { providers, refreshProviders } from "$lib/stores/providers";

export const load: Load = async ({ parent }) => {
    const data = await parent();

    if (!get(providers)) await refreshProviders();
    if (!get(domains)) await refreshDomains();

    return data;
};

import type { Load } from "@sveltejs/kit";
import { get } from "svelte/store";

import { checkers, refreshCheckers } from "$lib/stores/checkers";

export const load: Load = async ({ parent, params }) => {
    const data = await parent();

    if (!get(checkers)) await refreshCheckers();

    const subdomain = params.subdomain === "@" ? "" : params.subdomain;
    const serviceid = params.serviceid;

    return {
        ...data,
        subdomain,
        serviceid,
    };
};

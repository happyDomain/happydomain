import { type Load } from "@sveltejs/kit";

import { checkers, refreshCheckers } from "$lib/stores/checkers";
import { get } from "svelte/store";

export const load: Load = async ({ parent, params }) => {
    const data = await parent();

    if (get(checkers) === undefined) {
        refreshCheckers();
    }

    const subdomain = params.subdomain === "@" ? "" : params.subdomain;
    const serviceid = params.serviceid;

    return {
        ...data,
        subdomain,
        serviceid,
        isTestsPage: true,
    };
};

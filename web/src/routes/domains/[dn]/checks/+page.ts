import { get } from "svelte/store";
import { error, type Load } from "@sveltejs/kit";

import { getZone, thisZone } from "$lib/stores/thiszone";

export const load: Load = async ({ parent }) => {
    const data = await parent();

    const domain = data.domain;
    if (!domain) {
        error(404, { message: "Domain not found" });
    }

    if (domain.zone_history && domain.zone_history.length > 0) {
        const zoneId = domain.zone_history[0];
        const current = get(thisZone);
        if (current?.id !== zoneId) {
            getZone(domain, zoneId);
        }
    }

    return { ...data };
};

import { redirect, type Load } from "@sveltejs/kit";

import { showZone } from "$lib/stores/thiszone";

// The zone being edited: the latest of the history.
export const load: Load = async ({ parent, params, fetch }) => {
    const data = await parent();

    const domain = data.domain;
    if (!domain.zone_history || domain.zone_history.length === 0) {
        redirect(307, `/domains/${encodeURIComponent(params.dn!)}/import_zone`);
    }

    const zoneId: string = domain.zone_history[0];
    showZone(domain, zoneId, fetch);

    return {
        ...data,
        history: zoneId,
        zoneId,
    };
};

import { redirect, type Load } from "@sveltejs/kit";

import { showZone } from "$lib/stores/thiszone";

export const load: Load = async ({ parent, params, fetch }) => {
    const data = await parent();

    const domain = data.domain;
    if (!domain.zone_history || domain.zone_history.length === 0) {
        redirect(307, `/domains/${encodeURIComponent(params.dn!)}/import_zone`);
    }

    // The sidebar lists the subdomains of the current zone.
    showZone(domain, domain.zone_history[0], fetch);

    return { ...data };
};

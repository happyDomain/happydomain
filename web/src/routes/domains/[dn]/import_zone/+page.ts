import { redirect, type Load } from "@sveltejs/kit";
import { get } from "svelte/store";

import { domains_idx, refreshDomains } from "$lib/stores/domains";

export const load: Load = async ({ parent, params }) => {
    const data = await parent();

    // The zone may have been imported since the domains were loaded, by an
    // import left running in the background: coming back to this page must
    // not import it again.
    await refreshDomains();
    if (get(domains_idx)[params.dn!]?.zone_history?.length) {
        redirect(307, `/domains/${encodeURIComponent(params.dn!)}`);
    }

    return {
        ...data,
        isZonePage: true,
    };
};

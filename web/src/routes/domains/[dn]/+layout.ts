import { error, type Load } from "@sveltejs/kit";
import { get } from "svelte/store";

import type { Domain } from "$lib/model/domain";
import { domains, domains_idx, refreshDomains } from "$lib/stores/domains";

export const load: Load = async ({ parent, params }) => {
    const data = await parent();

    if (!get(domains)) await refreshDomains();

    if (!params.dn) {
        error(404, {
            message: "Domain not found",
        });
    }

    const domain: Domain | null = get(domains_idx)[params.dn];

    if (!domain) {
        error(404, {
            message: "Domain not found",
        });
    }

    let historyid = undefined;
    if (domain.zone_history && domain.zone_history.length > 0) {
        historyid = domain.zone_history[0];
    }

    return {
        domain,
        history: historyid,
        ...data,
    };
};

import { get } from "svelte/store";
import { error, redirect, type Load } from "@sveltejs/kit";

import { getZone, thisZone } from "$lib/stores/thiszone";

export const load: Load = async ({ parent, params }) => {
    const data = await parent();

    const domain = data.domain;

    if (!params.dn) {
        redirect(307, `/domains/`);
    }

    if (!domain.zone_history || domain.zone_history.length === 0) {
        redirect(307, `/domains/${encodeURIComponent(params.dn)}/import_zone`);
    }

    let definedhistory = true;
    if (!params.historyid) {
        params.historyid = domain.zone_history[0];
        definedhistory = false;
        //throw redirect(307, `/domains/${data.domain.domain}/${domain.zone_history[0]}`);
    }

    const zhidx = domain.zone_history.indexOf(params.historyid);
    if (zhidx < 0) {
        error(404, {
            message: "Zone not found in history",
        });
    }

    const zoneId: string = domain.zone_history[zhidx];

    const currentZone = get(thisZone);
    if (currentZone?.id !== zoneId) {
        getZone(domain, zoneId);
    }

    return {
        ...data,
        history: params.historyid,
        definedhistory,
        zoneId,
    };
};

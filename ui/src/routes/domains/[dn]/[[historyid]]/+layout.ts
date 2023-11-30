import { get_store_value } from 'svelte/internal';
import { error, redirect } from '@sveltejs/kit';
import type { Load } from '@sveltejs/kit';

import { domains_idx } from '$lib/stores/domains';
import { getZone } from '$lib/stores/thiszone';

export const load: Load = async({ parent, params }) => {
    const data = await parent();

    const domain = data.domain;

    if (!domain.zone_history || domain.zone_history.length === 0) {
        throw redirect(307, `/domains/${data.domain.domain}/import_zone`);
    }

    let definedhistory = true;
    if (!params.historyid) {
        params.historyid = domain.zone_history[0];
        definedhistory = false;
        //throw redirect(307, `/domains/${data.domain.domain}/${domain.zone_history[0]}`);
    }

    const zhidx = domain.zone_history.indexOf(params.historyid);
    if (zhidx < 0) {
        throw error(404, {
	    message: 'Zone not found in history'
	});
    }

    const zoneId: string = domain.zone_history[zhidx];

    const zone = getZone(domain, zoneId);

    return {
        ...data,
        history: params.historyid,
        definedhistory,
        zoneId,
        streamed: {
            zone,
        },
    }
}

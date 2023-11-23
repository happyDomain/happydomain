import { get_store_value } from 'svelte/internal';
import { error, redirect } from '@sveltejs/kit';
import type { Load } from '@sveltejs/kit';

import { getZone } from '$lib/api/zone';
import { domainCompare } from '$lib/dns';
import { domains_idx } from '$lib/stores/domains';

export const load: Load = async({ parent, params }) => {
    const data = await parent();

    const domain: DomainInList | null = get_store_value(domains_idx)[data.domain];

    if (domain === null) {
        throw error(404, {
	    message: 'Domain not found'
	});
    }
    if (!domain.zone_history || domain.zone_history.length === 0) {
        throw error(500, {
	    message: 'Domain not initialized'
	});
    }

    if (!params.historyid) {
        params.historyid = domain.zone_history[0];
        //throw redirect(307, `/domains/${data.domain}/${domain.zone_history[0]}`);
    }

    const zhidx = domain.zone_history.indexOf(params.historyid);
    if (zhidx < 0) {
        throw error(404, {
	    message: 'Zone not found in history'
	});
    }

    const zoneId: string = domain.zone_history[zhidx];

    const zone = getZone(domain, zoneId);

    const sortedDomains = zone.then((z) => {
        if (!z.services) {
            return [];
        }
        const domains = Object.keys(z.services);
        domains.sort(domainCompare);
        return domains;
    })

    return {
        history: params.historyid,
        selectedDomain: domain,
        zoneId,
        streamed: {
            zone,
            sortedDomains,
        },
        ...data,
    }
}

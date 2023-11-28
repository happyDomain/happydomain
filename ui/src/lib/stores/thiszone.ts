import { derived, writable, type Writable } from 'svelte/store';
import { domainCompare } from '$lib/dns';
import {
    retrieveZone as APIRetrieveZone,
    getZone as APIGetZone,
} from '$lib/api/zone';
import type { Domain, DomainInList } from '$lib/model/domain';
import type { Zone } from '$lib/model/zone';
import { refreshDomains } from '$lib/stores/domains';

export const thisZone: Writable<null | Zone> = writable(null);

export const sortedDomains = derived(
    thisZone,
    ($thisZone: null|Zone) => {
        if (!$thisZone) {
            return null;
        }
        if (!$thisZone.services) {
            return [];
        }
        const domains = Object.keys($thisZone.services);
        domains.sort(domainCompare);
        return domains;
    },
);

export async function getZone(domain: DomainInList | Domain, zoneId: string) {
    thisZone.set(null);

    const zone = await APIGetZone(domain, zoneId);

    thisZone.set(zone);

    return zone;
}

export async function retrieveZone(domain: string) {
    const meta = await APIRetrieveZone(domain);
    await refreshDomains();
    return meta;
}

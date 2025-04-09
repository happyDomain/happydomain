// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { derived, writable, type Writable } from 'svelte/store';
import { domainCompare } from '$lib/dns';
import {
    retrieveZone as APIRetrieveZone,
    getZone as APIGetZone,
} from '$lib/api/zone';
import type { Domain } from '$lib/model/domain';
import type { Zone } from '$lib/model/zone';
import { refreshDomains } from '$lib/stores/domains';

export const thisZone: Writable<null | Zone> = writable(null);

// sortedDomains returns all subdomains, sorted
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

// sortedDomainsWithIntermediate returns all subdomains, sorted, with intermediate subdomains
export const sortedDomainsWithIntermediate = derived(
    sortedDomains,
    ($sortedDomains: null|Array<string>) => {
        if (!$sortedDomains || $sortedDomains.length <= 1) {
            return $sortedDomains;
        }
        const domains: Array<string> = [$sortedDomains[0]];

        let previous = domains[0].split('.');
        for (let i = 1; i < $sortedDomains.length; i++) {
            const cur = $sortedDomains[i].split('.');

            if (previous.length < cur.length && previous[0] !== cur[cur.length - previous.length]) {
                domains.push(cur.slice(cur.length - previous.length).join('.'));
            }

            while (previous.length + 1 < cur.length) {
                previous = cur.slice(cur.length - previous.length - 1);
                domains.push(previous.join('.'));
            }

            domains.push(cur.join('.'));
            previous = cur;
        }

        return domains;
    },
);

export async function getZone(domain: Domain, zoneId: string) {
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

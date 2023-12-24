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

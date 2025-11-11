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

import { get } from "svelte/store";
import { derived, writable, type Writable } from "svelte/store";
import { domainCompare } from "$lib/dns";
import { retrieveZone as APIRetrieveZone, getZone as APIGetZone } from "$lib/api/zone";
import type { Domain } from "$lib/model/domain";
import type { Zone } from "$lib/model/zone";
import { refreshDomains } from "$lib/stores/domains";

// Main store for the current zone
export const thisZone: Writable<Zone | null> = writable(null);

// Derived store to retrieve all domain aliases
export const thisAliases = derived(thisZone, ($thisZone) => {
    const aliases: Record<string, string[]> = {};
    if (!$thisZone?.services) return aliases;

    Object.entries($thisZone.services).forEach(([dn, services]) => {
        services?.forEach((svc) => {
            if (svc._svctype === "svcs.CNAME") {
                const target = svc.service.Target;
                aliases[target] = [...(aliases[target] || []), dn];
            }
        });
    });

    if (aliases["@"]) {
        aliases[""] = aliases["@"];
    }

    return aliases;
});

// Derived store to retrieve all subdomains, sorted
export const sortedDomains = derived(thisZone, ($thisZone) => {
    if (!$thisZone?.services) return null;

    return Object.keys($thisZone.services).sort(domainCompare);
});

// Derived store to retrieve all subdomains, sorted, and with all intermediates, empty, subdomains
export const sortedDomainsWithIntermediate = derived(sortedDomains, ($sortedDomains) => {
    if (!$sortedDomains || $sortedDomains.length <= 1) return $sortedDomains;

    const domains = [$sortedDomains[0]];
    let previous = domains[0].split(".");

    for (let i = 1; i < $sortedDomains.length; i++) {
        const current = $sortedDomains[i].split(".");
        if (previous.length < current.length && previous[0] !== current[current.length - previous.length]) {
            domains.push(current.slice(current.length - previous.length).join("."));
        }
        while (previous.length + 1 < current.length) {
            previous = current.slice(current.length - previous.length - 1);
            domains.push(previous.join("."));
        }
        domains.push(current.join("."));
        previous = current;
    }

    return domains;
});

// getZone retrieve a given zone
export async function getZone(domain: Domain, zoneId: string) {
    const currentZone = get(thisZone);
    if (currentZone?.id === zoneId) {
        return currentZone;
    }

    thisZone.set(null);

    const zone = await APIGetZone(domain, zoneId);

    thisZone.set(zone);

    return zone;
}

export async function retrieveZone(domain: Domain) {
    const meta = await APIRetrieveZone(domain);
    await refreshDomains();
    return meta;
}

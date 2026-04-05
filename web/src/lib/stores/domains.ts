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

import { get, derived, writable, type Writable } from "svelte/store";
import { listDomains } from "$lib/api/domains";
import type { HappydnsDomainWithCheckStatus } from "$lib/api-base/types.gen";

export const domains: Writable<Array<HappydnsDomainWithCheckStatus> | undefined> = writable(undefined);
export const newlyGroups: Writable<Array<string>> = writable([]);

export async function refreshDomains() {
    const data = await listDomains();
    data.forEach((e) => {
        if (!e.group) e.group = "";
    });
    domains.set(data);
    return data;
}

export const groups = derived(domains, ($domains: Array<HappydnsDomainWithCheckStatus> | undefined) => {
    if (!$domains) return [];

    const groups = new Set<string>();

    for (const domain of $domains) {
        groups.add(domain.group || "");
    }

    return Array.from(groups).sort((a, b) => {
        if (!a) return 1;
        if (!b) return -1;
        return a.toLowerCase().localeCompare(b.toLowerCase());
    });
});

export const domains_idx = derived(domains, ($domains: Array<HappydnsDomainWithCheckStatus> | undefined) => {
    const idx: Record<string, HappydnsDomainWithCheckStatus> = {};

    if (!$domains) return idx;

    const multiview = new Set<string>();

    for (const d of $domains) {
        idx[d.id] = d;

        if (idx[d.domain]) {
            multiview.add(d.domain);
        } else {
            idx[d.domain] = d;
        }
    }

    for (const dn of multiview) {
        delete idx[dn];
    }

    return idx;
});

export const domains_by_name = derived(domains, ($domains: Array<HappydnsDomainWithCheckStatus> | undefined) => {
    const idx: Record<string, Array<HappydnsDomainWithCheckStatus>> = {};

    if (!$domains) return idx;

    for (const d of $domains) {
        if (idx[d.domain]) {
            idx[d.domain].push(d);
        } else {
            idx[d.domain] = [d];
        }
    }

    return idx;
});

export const domains_by_groups = derived(domains, ($domains: Array<HappydnsDomainWithCheckStatus> | undefined) => {
    const groups: Record<string, Array<HappydnsDomainWithCheckStatus>> = {};

    if ($domains === undefined) {
        return groups;
    }

    for (const domain of $domains) {
        if (groups[domain.group || ""] === undefined) {
            groups[domain.group || ""] = [];
        }

        groups[domain.group || ""].push(domain);
    }

    return groups;
});

export function domainLink(dnid: string): string {
    const idx = get(domains_idx);
    const entry = idx[dnid];
    if (entry && idx[entry.domain]) {
        return entry.domain;
    }
    return dnid;
}

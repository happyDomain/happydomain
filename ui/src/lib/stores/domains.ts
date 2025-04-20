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

import { derived, writable, type Writable } from "svelte/store";
import { listDomains } from "$lib/api/domains";
import type { Domain } from "$lib/model/domain";

export const domains: Writable<null | Array<Domain>> = writable(null);

export async function refreshDomains() {
    const data = await listDomains();
    domains.set(data);
    return data;
}

export const groups = derived(domains, ($domains: null | Array<Domain>) => {
    const groups: Record<string, null> = {};

    if ($domains) {
        for (const domain of $domains) {
            if (groups[domain.group] === undefined) {
                groups[domain.group] = null;
            }
        }
    }

    return Object.keys(groups).sort();
});

export const domains_idx = derived(domains, ($domains: null | Array<Domain>) => {
    const idx: Record<string, Domain> = {};

    if ($domains) {
        for (const d of $domains) {
            idx[d.domain] = d;
        }
    }

    return idx;
});

export const domains_by_groups = derived(domains, ($domains: null | Array<Domain>) => {
    const groups: Record<string, Array<Domain>> = {};

    if ($domains === null) {
        return groups;
    }

    for (const domain of $domains) {
        if (groups[domain.group] === undefined) {
            groups[domain.group] = [];
        }

        groups[domain.group].push(domain);
    }

    return groups;
});

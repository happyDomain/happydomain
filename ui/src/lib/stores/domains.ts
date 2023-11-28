import { derived, writable, type Writable } from 'svelte/store';
import { listDomains } from '$lib/api/domains';
import type { DomainInList } from '$lib/model/domain';

export const domains: Writable<null | Array<DomainInList>> = writable(null);

export async function refreshDomains() {
    const data = await listDomains();
    domains.set(data);
    return data;
}

export const groups = derived(
    domains,
    ($domains: null|Array<DomainInList>) => {
        const groups: Record<string, null> = { };

        if ($domains) {
            for (const domain of $domains) {
                if (groups[domain.group] === undefined) {
                    groups[domain.group] = null;
                }
            }
        }

        return Object.keys(groups).sort();
    },
);

export const domains_idx = derived(
    domains,
    ($domains: null|Array<DomainInList>) => {
        const idx: Record<string, DomainInList> = { };

        if ($domains) {
            for (const d of $domains) {
                idx[d.domain] = d;
            }
        }

        return idx;
    },
);

export const domains_by_groups = derived(
    domains,
    ($domains: null|Array<DomainInList>) => {
        const groups: Record<string, Array<DomainInList>> = { };

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
    },
);

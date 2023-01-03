import { derived, writable, type Writable } from 'svelte/store';
import { listDomains } from '$lib/api/domains';
import type { Domain } from '$lib/model/domain';

export const domains: Writable<null | Array<Domain>> = writable(null);

export async function refreshDomains() {
    const data = await listDomains();
    domains.set(data);
    return data;
}

export const groups = derived(
    domains,
    ($domains: null|Array<Domain>) => {
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
    ($domains: null|Array<Domain>) => {
        const idx: Record<string, Domain> = { };

        if ($domains) {
            for (const d of $domains) {
                idx[d.domain] = d;
            }
        }

        return idx;
    },
);

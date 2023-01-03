import { derived, writable, type Writable } from 'svelte/store';
import { listProviders } from '$lib/api/provider';
import { Provider, ProviderInfos } from '$lib/model/provider';

export const providers: Writable<null | Array<Provider>> = writable(null);
export const providersSpecs: Writable<null | Record<string, ProviderInfos>> = writable(null);

export async function refreshProviders() {
    const data = await listProviders();
    providers.set(data);
    return data;
}

export const providers_idx = derived(
    providers,
    ($providers: null|Array<Provider>) => {
        const idx: Record<string, Provider> = { };

        if ($providers) {
            for (const p of $providers) {
                idx[p._id] = p;
            }
        }

        return idx;
    },
);

export async function refreshProvidersSpecs() {
    const res = await fetch('/api/providers/_specs', {headers: {'Accept': 'application/json'}})
    if (res.status == 200) {
        const data = await res.json();

        const map: Record<string, ProviderInfos> = { };
        for (const pi in data) {
            map[pi] = new ProviderInfos(data[pi])
        }
        providersSpecs.set(map);
        return map;
    } else {
        throw new Error((await res.json()).errmsg);
    }
}

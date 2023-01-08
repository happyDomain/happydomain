import { writable, type Writable } from 'svelte/store';
import type { ServiceInfos } from '$lib/model/service_specs';

export const servicesSpecs: Writable<null | Record<string, ServiceInfos>> = writable(null);

export async function refreshServicesSpecs() {
    const res = await fetch('/api/service_specs', {headers: {'Accept': 'application/json'}})
    if (res.status == 200) {
        const map = await res.json();
        servicesSpecs.set(map);
        return map;
    } else {
        throw new Error((await res.json()).errmsg);
    }
}

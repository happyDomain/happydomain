import { writable, type Writable } from 'svelte/store';
import { ServiceInfos } from '$lib/model/service_specs';

export const servicesSpecs: Writable<null | Record<string, ServiceInfos>> = writable(null);

export async function refreshServicesSpecs() {
    const res = await fetch('/api/service_specs', {headers: {'Accept': 'application/json'}})
    if (res.status == 200) {
        const data = await res.json();

        const map: Record<string, ServiceInfos> = { };
        for (const pi in data) {
            map[pi] = new ServiceInfos(data[pi])
        }
        servicesSpecs.set(map);
        return map;
    } else {
        throw new Error((await res.json()).errmsg);
    }
}

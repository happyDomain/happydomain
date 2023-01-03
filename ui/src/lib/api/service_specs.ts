import { handleApiResponse } from '$lib/errors';
import type { ServiceInfos, ServiceSpec } from '$lib/model/service_specs';

export async function listServiceSpecs(): Promise<Record<string, ServiceInfos>> {
    const res = await fetch('/api/service_specs', {
        method: 'GET',
        headers: {'Accept': 'application/json'},
    });
    return await handleApiResponse<Record<string, ServiceInfos>>(res);
}

export async function getServiceSpec(ssid: string): Promise<ServiceSpec> {
    const res = await fetch(`/api/service_specs/` + ssid, {
        method: 'GET',
        headers: {'Accept': 'application/json'},
    });
    return await handleApiResponse<ServiceSpec>(res);
}

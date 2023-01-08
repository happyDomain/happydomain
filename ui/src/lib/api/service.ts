import { handleEmptyApiResponse, handleApiResponse } from '$lib/errors';
import type { ServiceCombined } from '$lib/model/service';

export async function updateService(zoneid: string, svc: ServiceCombined): Promise<ServiceCombined> {
    const res = await fetch('/api/zone/' + zoneid + '/services/' + (svc._id ? `/${svc._id}` : ''), {
        method: svc._id?'PUT':'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(svc),
    });
    return await handleApiResponse<ServiceCombined>(res);
}

export async function deleteService(zoneid: string, id: string): Promise<boolean> {
    const res = await fetch(`/api/zone/${zoneid}/services/${id}`, {
        method: 'DELETE',
        headers: {'Accept': 'application/json'},
    });
    return await handleEmptyApiResponse(res);
}

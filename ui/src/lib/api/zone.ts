import { handleApiResponse } from '$lib/errors';
import type { Domain } from '$lib/model/domain';
import type { ServiceCombined, ServiceMeta } from '$lib/model/service';
import type { ServiceRecord, Zone, ZoneMeta } from '$lib/model/zone';

export async function getZone(domain: Domain, id: string): Promise<Zone> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${dnid}/zone/${id}`, {headers: {'Accept': 'application/json'}});
    return await handleApiResponse<Zone>(res);
}

export async function viewZone(domain: Domain, id: string): Promise<string> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${dnid}/zone/${id}/view`, {
        method: 'POST',
        headers: {'Accept': 'application/json'}
    });
    return await handleApiResponse<string>(res);
}

export async function importZone(domain: Domain): Promise<ZoneMeta> {
    const dnid = encodeURIComponent(domain.id);
    const res = await fetch(`/api/domains/${dnid}/import_zone`, {
        method: 'POST',
        headers: {'Accept': 'application/json'}
    });
    return await handleApiResponse<ZoneMeta>(res);
}

export async function applyZone(domain: Domain, id: string, selectedDiffs: Array<string>): Promise<ZoneMeta> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${dnid}/zone/${id}/apply_changes`, {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(selectedDiffs),
    });
    return await handleApiResponse<ZoneMeta>(res);
}

export async function diffZone(domain: Domain, id1: string, id2: string): Promise<Array<string>> {
    const dnid = encodeURIComponent(domain.id);
    id1 = encodeURIComponent(id1);
    id2 = encodeURIComponent(id2);
    const res = await fetch(`/api/domains/${dnid}/diff_zones/${id1}/${id2}`, {
        method: 'POST',
        headers: {'Accept': 'application/json'}
    });
    return await handleApiResponse<Array<string>>(res);
}

export async function addZoneService(domain: Domain, id: string, service: ServiceCombined): Promise<Zone> {
    let subdomain = service._domain;
    if (subdomain === '') subdomain = '@';

    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    subdomain = encodeURIComponent(subdomain);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/${subdomain}/services`, {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(service)
    });
    return await handleApiResponse<Zone>(res);
}

export async function updateZoneService(domain: Domain, id: string, service: ServiceCombined): Promise<Zone> {
    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);

    const res = await fetch(`/api/domains/${dnid}/zone/${id}`, {
        method: 'PATCH',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(service),
    });
    return await handleApiResponse<Zone>(res);
}

export async function deleteZoneService(domain: Domain, id: string, service: ServiceMeta): Promise<Zone> {
    let subdomain = service._domain;
    if (subdomain === '') subdomain = '@';

    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    subdomain = encodeURIComponent(subdomain);
    const svcid = service._id?encodeURIComponent(service._id):undefined;

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/${subdomain}/services/${svcid}`, {
        method: 'DELETE',
        headers: {'Accept': 'application/json'}
    });
    return await handleApiResponse<Zone>(res);
}

export async function getServiceRecords(domain: Domain, id: string, service: ServiceMeta): Promise<Array<ServiceRecord>> {
    let subdomain = service._domain;
    if (subdomain === '') subdomain = '@';

    const dnid = encodeURIComponent(domain.id);
    id = encodeURIComponent(id);
    const svcid = service._id?encodeURIComponent(service._id):undefined;

    const res = await fetch(`/api/domains/${dnid}/zone/${id}/${subdomain}/services/${svcid}/records`, {
        headers: {'Accept': 'application/json'}
    });
    return await handleApiResponse<Array<ServiceRecord>>(res);
}

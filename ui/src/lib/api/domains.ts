import { handleEmptyApiResponse, handleApiResponse } from '$lib/errors';
import type { Domain, DomainInList, DomainLog } from '$lib/model/domain';
import type { Provider } from '$lib/model/provider';

export async function listDomains(): Promise<Array<DomainInList>> {
    const res = await fetch('/api/domains', {headers: {'Accept': 'application/json'}});
    return (await handleApiResponse<Array<DomainInList>>(res));
}

export async function getDomain(id: string): Promise<Domain> {
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${id}`, {headers: {'Accept': 'application/json'}});
    return await handleApiResponse<Domain>(res);
}

export async function addDomain(domain: string, provider: Provider|undefined): Promise<Domain> {
    const id_provider = provider?provider._id:undefined;

    const res = await fetch('/api/domains', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify({
            domain,
            id_provider,
        }),
    });
    return await handleApiResponse<Domain>(res);
}

export async function updateDomain(domain: Domain): Promise<Domain> {
    const res = await fetch('/api/domains' + (domain.id ? `/${domain.id}` : ''), {
        method: domain.id?'PUT':'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(domain),
    });
    return await handleApiResponse<Domain>(res);
}

export async function deleteDomain(id: string): Promise<boolean> {
    const res = await fetch(`/api/domains/${id}`, {
        method: 'DELETE',
        headers: {'Accept': 'application/json'},
    });
    return await handleEmptyApiResponse(res);
}

export async function getDomainLogs(id: string): Promise<Array<DomainLog>> {
    id = encodeURIComponent(id);
    const res = await fetch(`/api/domains/${id}/logs`, {headers: {'Accept': 'application/json'}});
    return await handleApiResponse<Array<DomainLog>>(res);
}

import { handleEmptyApiResponse, handleApiResponse } from '$lib/errors';
import type { Provider } from '$lib/model/provider';

export async function listProviders(): Promise<Array<Provider>> {
    const res = await fetch('/api/providers', {headers: {'Accept': 'application/json'}});
    return (await handleApiResponse<Array<Provider>>(res));
}

export async function getProvider(id: string): Promise<Provider> {
    id = encodeURIComponent(id);
    const res = await fetch(`/api/providers/${id}`, {headers: {'Accept': 'application/json'}});
    return await handleApiResponse<Provider>(res);
}

export async function listImportableDomains(provider: Provider): Promise<Array<string>> {
    const res = await fetch(`/api/providers/${provider._id}/domains`, {
        method: 'GET',
        headers: {'Accept': 'application/json'},
    });
    return await handleApiResponse<Array<string>>(res);
}

export async function updateProvider(provider: Provider): Promise<Provider> {
    const res = await fetch('/api/providers' + (provider._id ? `/${provider._id}` : ''), {
        method: provider._id?'PUT':'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(provider),
    });
    return await handleApiResponse<Provider>(res);
}

export async function deleteProvider(id: string): Promise<boolean> {
    const res = await fetch(`/api/providers/${id}`, {
        method: 'DELETE',
        headers: {'Accept': 'application/json'},
    });
    return await handleEmptyApiResponse(res);
}

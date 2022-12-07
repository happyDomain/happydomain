import { handleApiResponse } from '$lib/errors';
import type { Provider, ProviderList } from '$lib/model/provider';

export async function listProviders(): Promise<ProviderList> {
    const res = await fetch('/api/providers/_specs', {
        method: 'GET',
        headers: {'Accept': 'application/json'},
    });
    return await handleApiResponse(res);
}

export async function getProviderSpec(psid: string): Promise<Provider> {
    const res = await fetch(`/api/providers/_specs/` + psid, {
        method: 'GET',
        headers: {'Accept': 'application/json'},
    });
    return new Provider(await handleApiResponse(res));
}

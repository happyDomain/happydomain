import { handleApiResponse } from '$lib/errors';
import type { ProviderInfos, ProviderList } from '$lib/model/provider';

export async function listProviders(): Promise<ProviderList> {
    const res = await fetch('/api/providers/_specs', {
        method: 'GET',
        headers: {'Accept': 'application/json'},
    });
    return await handleApiResponse<ProviderList>(res);
}

export async function getProviderSpec(psid: string): Promise<ProviderInfos> {
    const res = await fetch(`/api/providers/_specs/` + psid, {
        method: 'GET',
        headers: {'Accept': 'application/json'},
    });
    return await handleApiResponse<ProviderInfos>(res);
}

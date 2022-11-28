import { handleApiResponse } from '$lib/errors';
import type { ResolverForm } from '$lib/model/resolver';

export async function resolve(form: ResolverForm): Promise<any> {
    const res = await fetch(`/api/resolver`, {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(form),
    });
    return await handleApiResponse(res);
}

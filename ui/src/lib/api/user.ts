import { handleApiResponse } from '$lib/errors';
import type { SignUpForm } from '$lib/model/user';

export async function registerUser(form: SignUpForm): Promise<any> {
    const res = await fetch('api/users', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(form),
    });
    return await handleApiResponse(res);
}

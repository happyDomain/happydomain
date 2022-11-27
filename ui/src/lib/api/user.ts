import { handleApiResponse } from '$lib/errors';
import type { SignUpForm, LoginForm } from '$lib/model/user';

export async function registerUser(form: SignUpForm): Promise<any> {
    const res = await fetch('api/users', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(form),
    });
    return await handleApiResponse(res);
}

export async function authUser(form: LoginForm): Promise<any> {
    const res = await fetch('api/auth', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(form),
    });
    return await handleApiResponse(res);
}

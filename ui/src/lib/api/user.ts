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

export async function forgotAccountPassword(email: string): any {
    const res = await fetch('api/users', {
        method: 'PATCH',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify({
            email,
            kind: 'recovery',
        }),
    });
    return await handleApiResponse(res);
}

export async function recoverAccount(userid: string, key: string, password: string): any {
    userid = encodeURIComponent(userid);
    const res = await fetch(`api/users/${userid}/recovery`, {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify({
            key,
            password,
        }),
    });
    return await handleApiResponse(res);
}

export async function validateEmail(userid: string, key: string): any {
    userid = encodeURIComponent(userid);
    const res = await fetch(`api/users/${userid}/email`, {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify({
            key,
        }),
    });
    return await handleApiResponse(res);
}

import { handleApiResponse } from '$lib/errors';
import type { SignUpForm, LoginForm } from '$lib/model/user';

export async function registerUser(form: SignUpForm): Promise<any> {
    const res = await fetch('/api/users', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(form),
    });
    return await handleApiResponse(res);
}

export async function authUser(form: LoginForm): Promise<any> {
    const res = await fetch('/api/auth', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify(form),
    });
    return await handleApiResponse(res);
}

export async function logout(): Promise<any> {
    const res = await fetch('/api/auth/logout', {
        method: 'POST',
        headers: {'Accept': 'application/json'},
    });
    return await handleApiResponse(res);
}

export async function specialUserOperations(email: string, kind: "recovery"|"validation"): Promise<any> {
    const res = await fetch('/api/users', {
        method: 'PATCH',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify({
            email,
            kind,
        }),
    });
    return await handleApiResponse(res);
}

export function forgotAccountPassword(email: string): Promise<any> {
    return specialUserOperations(email, "recovery")
}

export async function resendValidationEmail(email: string): Promise<any> {
    return specialUserOperations(email, "validation")
}

export async function recoverAccount(userid: string, key: string, password: string): Promise<any> {
    userid = encodeURIComponent(userid);
    const res = await fetch(`/api/users/${userid}/recovery`, {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify({
            key,
            password,
        }),
    });
    return await handleApiResponse(res);
}

export async function validateEmail(userid: string, key: string): Promise<any> {
    userid = encodeURIComponent(userid);
    const res = await fetch(`/api/users/${userid}/email`, {
        method: 'POST',
        headers: {'Accept': 'application/json'},
        body: JSON.stringify({
            key,
        }),
    });
    return await handleApiResponse(res);
}

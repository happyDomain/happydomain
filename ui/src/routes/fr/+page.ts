import { redirect } from '@sveltejs/kit';
import type { Load } from '@sveltejs/kit';

export const load: Load = async() => {
    throw redirect(302, '/join');
}

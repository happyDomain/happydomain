import { redirect } from '@sveltejs/kit';
import type { Load } from '@sveltejs/kit';
import { get_store_value } from 'svelte/internal';

import { refreshDomains } from '$lib/stores/domains';
import { userSession } from '$lib/stores/usersession';
import { config as tsConfig, locale } from '$lib/translations';

export const load: Load = async({ parent }) => {
    await parent();

    // If not connected, redirect to main website in the right language
    if (!get_store_value(userSession)) {
        const initLocale = locale.get() || window.navigator.language || window.navigator.languages[0] || tsConfig.fallbackLocale;
        throw redirect(302, '/' + initLocale);
    }

    await refreshDomains();

    return {};
}

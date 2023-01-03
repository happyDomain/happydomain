import type { Load } from '@sveltejs/kit';
import { get_store_value } from 'svelte/internal';

import { refreshUserSession } from '$lib/stores/usersession';
import { config as tsConfig, locale, loadTranslations } from '$lib/translations';

export const ssr = false;

export const load: Load = async({ fetch, route, url }) => {
    const initLocale = locale.get() || window.navigator.language || window.navigator.languages[0] || tsConfig.fallbackLocale || "en";

    await loadTranslations(initLocale, url.pathname);

    // Load user session if any
    try {
        const user = await refreshUserSession(fetch);
        if (get_store_value(locale) != user.settings.language) {
            locale.set(user.settings.language);
        }
    } catch {}

    return {
        route,
    };
}

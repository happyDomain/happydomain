import type { Load } from '@sveltejs/kit';

import { refreshUserSession } from '$lib/stores/usersession';
import { config as tsConfig, locale, loadTranslations } from '$lib/translations';

export const ssr = false;

export const load: Load = async({ url }) => {
    const initLocale = locale.get() || window.navigator.language || window.navigator.languages[0] || tsConfig.fallbackLocale;

    await loadTranslations(initLocale, url.pathname);

    // Load user session if any
    try {
        await refreshUserSession();
    } catch {}

    return {};
}

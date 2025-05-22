// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { redirect, type Load } from "@sveltejs/kit";
import { get } from "svelte/store";

import { toasts } from "$lib/stores/toasts";
import { refreshUserSession } from "$lib/stores/usersession";
import { config as tsConfig, locale, loadTranslations, t } from "$lib/translations";

export const ssr = false;

const sw_state = { triedUpdate: false, hasUpdate: false };

function onSWupdate(sw_state: { hasUpdate: boolean }, installingWorker: ServiceWorker) {
    if (!sw_state.hasUpdate) {
        toasts.addToast({
            title: get(t)("upgrade.title"),
            message: get(t)("upgrade.content"),
            onclick: () => installingWorker.postMessage("SKIP_WAITING"),
        });
    }
    sw_state.hasUpdate = true;
}

export const load: Load = async ({ fetch, route, url }) => {
    const { MODE } = import.meta.env;

    const initLocale =
        url.searchParams.get("lang") ||
        locale.get() ||
        (window.navigator.language ? window.navigator.language.substring(0,2) : null) ||
        window.navigator.languages[0] ||
        tsConfig.fallbackLocale ||
        "en";

    await loadTranslations(initLocale, url.pathname);

    if (MODE == "production" && "serviceWorker" in navigator) {
        navigator.serviceWorker.ready.then((registration) => {
            registration.onupdatefound = () => {
                const installingWorker = registration.installing;

                if (installingWorker === null) return;

                installingWorker.onstatechange = () => {
                    if (installingWorker.state === "installed") {
                        if (navigator.serviceWorker.controller) {
                            onSWupdate(sw_state, installingWorker);
                        }
                    }
                };
            };

            if (!sw_state.triedUpdate) {
                sw_state.triedUpdate = true;
                registration.update();
                setInterval(
                    function (reg: ServiceWorkerRegistration) {
                        reg.update();
                    },
                    36000000,
                    registration,
                );
            }
        });

        let refreshing = false;
        navigator.serviceWorker.addEventListener("controllerchange", () => {
            if (!refreshing) {
                window.location.reload();
                refreshing = true;
            }
        });
    }

    // Load user session if any
    try {
        const user = await refreshUserSession(fetch);
        if (!url.searchParams.has("lang") && get(locale) != user.settings.language) {
            locale.set(user.settings.language);
        }
    } catch (err) {
        if (route.id != null && route.id != "/login" && route.id != "/forgotten-password" && route.id != "/join" && !route.id.startsWith("/resolver") && route.id != "/providers/features" && !route.id.startsWith("/email-validation")) {
            toasts.addToast({
                type: 'error',
                title: get(t)("errors.session.title"),
                message: get(t)("errors.session.content"),
            });
            throw redirect(302, '/login?next=' + encodeURIComponent(url.pathname));
        }
    }

    return {
        sw_state,
    };
};

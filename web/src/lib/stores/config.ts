// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2025 happyDomain
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

import { goto } from '$app/navigation';
import { writable } from 'svelte/store';

import type { Color } from "@sveltestrap/sveltestrap";

interface AppConfig {
    captcha_provider?: string;
    captcha_site_key?: string;
    disable_embedded_login?: boolean;
    disable_providers?: boolean;
    disable_registration?: boolean;
    hide_feedback?: boolean;
    msg_header?: {
        text: string;
        color: Color;
    };
    no_mail?: boolean;
    oidc_configured?: boolean;
}

const defaultConfig: AppConfig = {
    disable_embedded_login: false,
    disable_providers: false,
    disable_registration: false,
    hide_feedback: false,
    msg_header: undefined,
    no_mail: false,
    oidc_configured: false,
};

function getConfigFromScriptTag(): AppConfig | null {
    if (typeof document !== 'undefined') {
        const configScript = document.getElementById('app-config');
        if (configScript) {
            try {
                return JSON.parse(configScript.textContent || '');
            } catch (e) {
                console.error('Failed to parse app config:', e);
            }
        }
    }
    return null;
}

const initialConfig = getConfigFromScriptTag() || defaultConfig;

export const appConfig = writable<AppConfig>(initialConfig);

export const base: string = typeof document !== 'undefined'
    ? (document.querySelector('base')?.getAttribute('href') ?? '')
    : '';

export function navigate(url: string, opts?: Parameters<typeof goto>[1]) {
    return goto(base + url, opts);
}

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

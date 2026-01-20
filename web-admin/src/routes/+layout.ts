// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { config as tsConfig, locale, loadTranslations, t } from "$lib/translations";

export const ssr = false;

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

    return {};
};

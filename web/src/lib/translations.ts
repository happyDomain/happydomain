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

import i18n from "sveltekit-i18n";
import type { Config } from "sveltekit-i18n";
const { MODE } = import.meta.env;

interface Params {
    action?: string;
    id?: string;
    domain?: string;
    type?: string;
    happyDomain?: string;
    thing?: string;
    identify?: string;
    provider?: string;
    "security-operations"?: string;
    "first-step"?: string;
    n?: number;
    count?: number;
    min?: number;
    max?: number;
    suggestion?: string;
    // add more parameters that are used here
}

export const config: Config<Params> = {
    fallbackLocale: "en",
    loaders: [
        {
            locale: "de",
            key: "",
            loader: async () => (await import("./locales/de.json")).default,
        },
        {
            locale: "en",
            key: "",
            loader: async () => {
                if (MODE == "development") {
                    return await (await fetch("/src/lib/locales/en.json")).json();
                } else {
                    return (await import("./locales/en.json")).default;
                }
            },
        },
        {
            locale: "es",
            key: "",
            loader: async () => (await import("./locales/es.json")).default,
        },
        {
            locale: "fr",
            key: "",
            loader: async () => (await import("./locales/fr.json")).default,
        },
        {
            locale: "hi",
            key: "",
            loader: async () => (await import("./locales/hi.json")).default,
        },
        {
            locale: "zh",
            key: "",
            loader: async () => (await import("./locales/zh.json")).default,
        },
    ],
};

export const { t, locales, locale, loadTranslations } = new i18n(config);

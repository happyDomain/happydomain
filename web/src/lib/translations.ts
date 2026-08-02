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
    // Services carry their own translations, hence their own interpolation
    // parameters: they must never have to come and edit this file.
    [key: string]: string | number | undefined;

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
    svctype?: string;
    name?: string;
    nbDiffs?: number;
    nbSelected?: number;
    countdown?: string;
    error?: string;
    options?: string;
    key?: string;
    days?: number;
    intervalMin?: string;
    intervalMax?: string;
    intervalDefault?: string;
    service?: string;
    value?: string;
    // add more parameters that are used here
}

// Locales the application offers. The language switcher is built from the
// loaders below, so a service shipping a translation for a language absent
// from this list stays dormant until the language is enabled here.
const APP_LOCALES = ["de", "en", "es", "fr", "hi", "zh"];

// Each service folder owns its translations in `locales/<lang>.json`; they are
// collected here so adding a service never means editing a shared file.
const serviceLocaleFiles = import.meta.glob<{ default: Record<string, unknown> }>(
    "./services/*/locales/*.json",
);

function isObject(v: unknown): v is Record<string, unknown> {
    return typeof v === "object" && v !== null && !Array.isArray(v);
}

function deepMerge(target: Record<string, unknown>, source: Record<string, unknown>) {
    for (const [key, value] of Object.entries(source)) {
        const previous = target[key];
        if (isObject(previous) && isObject(value)) {
            deepMerge(previous, value);
        } else {
            target[key] = value;
        }
    }
    return target;
}

const perLocaleServiceFiles: Record<
    string,
    Array<() => Promise<{ default: Record<string, unknown> }>>
> = {};

for (const [path, load] of Object.entries(serviceLocaleFiles)) {
    const lang = path.split("/").at(-1)?.replace(/\.json$/, "");
    if (!lang || !APP_LOCALES.includes(lang)) continue;
    (perLocaleServiceFiles[lang] ??= []).push(load);
}

/**
 * Merge the translations owned by the services into the application ones.
 * sveltekit-i18n keeps a single payload per locale and key, so they have to be
 * merged here rather than registered as extra loaders.
 */
async function withServices(lang: string, appTranslations: Record<string, unknown>) {
    const modules = await Promise.all((perLocaleServiceFiles[lang] ?? []).map((load) => load()));
    // Clone, so the merge never mutates the cached JSON modules.
    return modules.reduce(
        (acc, module) => deepMerge(acc, module.default),
        structuredClone(appTranslations),
    );
}

export const config: Config<Params> = {
    fallbackLocale: "en",
    loaders: [
        {
            locale: "de",
            key: "",
            loader: async () => withServices("de", (await import("./locales/de.json")).default),
        },
        {
            locale: "en",
            key: "",
            loader: async () => {
                if (MODE == "development") {
                    return withServices("en", await (await fetch("/src/lib/locales/en.json")).json());
                } else {
                    return withServices("en", (await import("./locales/en.json")).default);
                }
            },
        },
        {
            locale: "es",
            key: "",
            loader: async () => withServices("es", (await import("./locales/es.json")).default),
        },
        {
            locale: "fr",
            key: "",
            loader: async () => withServices("fr", (await import("./locales/fr.json")).default),
        },
        {
            locale: "hi",
            key: "",
            loader: async () => withServices("hi", (await import("./locales/hi.json")).default),
        },
        {
            locale: "zh",
            key: "",
            loader: async () => withServices("zh", (await import("./locales/zh.json")).default),
        },
        {
            locale: "en",
            key: "locales",
            loader: async () => {
                if (MODE == "development") {
                    return await (await fetch("/src/lib/locales/lang.json")).json();
                } else {
                    return (await import("./locales/lang.json")).default;
                }
            },
        },
    ],
};

export const { t, locales, locale, loadTranslations } = new i18n(config);

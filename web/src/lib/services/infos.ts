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

import { derived, type Readable } from "svelte/store";

import type { ServiceInfos } from "$lib/model/service_specs.svelte";
import { t } from "$lib/translations";

/**
 * How a service is introduced to the user is a matter of wording, not of DNS:
 * both live with the service, in its `locales/<lang>.json`, under
 * `svcinfo.<svctype>`. The backend only knows the untranslated name it
 * registers, which serves as the last resort when a translation is missing.
 */
function key(svctype: string, field: "name" | "description"): string {
    return `svcinfo.${svctype}.${field}`;
}

type Translate = (key: string) => string;

function translated(translate: Translate, svctype: string, field: "name" | "description"): string {
    const k = key(svctype, field);
    const value = translate(k);
    // sveltekit-i18n echoes the key back when it has no translation for it.
    return value === k ? "" : value;
}

/**
 * @param svc Specifications of the service, when they are known.
 * @param svctype Service type, to fall back on when they are not.
 */
export function serviceNameOf(
    translate: Translate,
    svc?: ServiceInfos,
    svctype?: string,
): string {
    const type = svc?._svctype || svctype || "";
    return translated(translate, type, "name") || svc?.name || type;
}

export function serviceDescriptionOf(
    translate: Translate,
    svc?: ServiceInfos,
    svctype?: string,
): string {
    return translated(translate, svc?._svctype || svctype || "", "description");
}

type InfosGetter = (svc?: ServiceInfos, svctype?: string) => string;

/** `$serviceName(svc, svctype)`, reactive to the selected language. */
export const serviceName: Readable<InfosGetter> = derived(
    t,
    ($t) => (svc?: ServiceInfos, svctype?: string) =>
        serviceNameOf($t as Translate, svc, svctype),
);

/** `$serviceDescription(svc, svctype)`, reactive to the selected language. */
export const serviceDescription: Readable<InfosGetter> = derived(
    t,
    ($t) => (svc?: ServiceInfos, svctype?: string) =>
        serviceDescriptionOf($t as Translate, svc, svctype),
);

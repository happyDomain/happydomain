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

import { get } from "svelte/store";

import type { Field } from "$lib/model/custom_form";
import { getAvailableResourceTypes, type ProviderInfos } from "$lib/model/provider";
import type { ServiceCombined } from "$lib/model/service";
import { servicesSpecs } from "$lib/stores/services";

export interface ServiceRestrictions {
    alone: boolean;
    exclusive: Array<string>;
    glue: boolean;
    leaf: boolean;
    nearAlone: boolean;
    needTypes: Array<number>;
    rootOnly: boolean;
    single: boolean;
}

export interface ServiceInfos {
    name: string;
    _svctype: string;
    _svcicon: string;
    description: string;
    family: string;
    categories: Array<string>;
    record_types: Array<number>;
    tabs: boolean;
    restrictions: ServiceRestrictions;
}

export interface ServiceSpec {
    fields: null | Array<Field>;
}

export function passRestrictions(
    svcinfo: ServiceInfos,
    provider_specs: ProviderInfos,
    zservices: Record<string, Array<ServiceCombined>>,
    dn: string,
): null | string {
    if (svcinfo.restrictions) {
        // Handle NeedTypes restriction: hosting provider need to support given types.
        if (svcinfo.restrictions.needTypes) {
            const availableResourceTypes = getAvailableResourceTypes(provider_specs);

            for (const needType of svcinfo.restrictions.needTypes) {
                if (availableResourceTypes.indexOf(needType) < 0) {
                    return "is not available on this domain name hosting provider.";
                }
            }
        }

        // Handle rootOnly restriction.
        if (svcinfo.restrictions.rootOnly && dn !== "") {
            return "can only be present at the root of your domain.";
        }

        if (zservices[dn] == null) return null;

        const sspecs = get(servicesSpecs);

        if (sspecs == null) return null;

        // Handle Alone restriction: only nearAlone are allowed.
        if (svcinfo.restrictions.alone) {
            for (const s of zservices[dn]) {
                if (
                    s._svctype !== svcinfo._svctype &&
                    sspecs[s._svctype].restrictions &&
                    !sspecs[s._svctype].restrictions.nearAlone
                ) {
                    return "only one per subdomain.";
                }
            }
        }

        // Handle Exclusive restriction: service can't be present along with another listed one.
        if (svcinfo.restrictions.exclusive) {
            for (const s of zservices[dn]) {
                for (const exclu of svcinfo.restrictions.exclusive) {
                    if (s._svctype === exclu) {
                        return "cannot coexist with " + sspecs[s._svctype].name + ".";
                    }
                }
            }
        }

        // Check reverse Exclusivity
        for (const k in zservices[dn]) {
            const s = sspecs[zservices[dn][k]._svctype];
            if (!s.restrictions || !s.restrictions.exclusive) {
                continue;
            }
            for (const i in s.restrictions.exclusive) {
                if (svcinfo._svctype === s.restrictions.exclusive[i]) {
                    return "cannot coexist with " + sspecs[s._svctype].name + ".";
                }
            }
        }

        // Handle Single restriction: only one instance of the service per subdomain.
        if (svcinfo.restrictions.single) {
            for (const s of zservices[dn]) {
                if (s._svctype === svcinfo._svctype) {
                    return "can only be present once per subdomain.";
                }
            }
        }

        // Handle presence of Alone and Leaf service in subdomain already.
        let oneAlone: string | null = null;
        let oneLeaf: string | null = null;
        for (const s of zservices[dn]) {
            if (
                sspecs[s._svctype] &&
                sspecs[s._svctype].restrictions &&
                sspecs[s._svctype].restrictions.alone
            ) {
                oneAlone = s._svctype;
            }
            if (
                sspecs[s._svctype] &&
                sspecs[s._svctype].restrictions &&
                sspecs[s._svctype].restrictions.leaf
            ) {
                oneLeaf = s._svctype;
            }
        }
        if (oneAlone && oneAlone !== svcinfo._svctype && !svcinfo.restrictions.nearAlone) {
            return (
                "cannot coexist with " +
                sspecs[oneAlone].name +
                ", that requires to be the only one in the subdomain."
            );
        }
        if (oneLeaf && oneLeaf !== svcinfo._svctype && !svcinfo.restrictions.glue) {
            return "cannot coexist with " + sspecs[oneLeaf].name + ", that cannot have subdomains.";
        }
    }

    return null;
}

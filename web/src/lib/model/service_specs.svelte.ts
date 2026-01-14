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

import { getRrtype, newRR } from "$lib/dns_rr";
import type { Field } from "$lib/model/custom_form.svelte";
import { getAvailableResourceTypes, type ProviderInfos } from "$lib/model/provider";
import type { ServiceCombined } from "$lib/model/service.svelte";
import { servicesSpecs, servicesSpecsLoaded } from "$lib/stores/services";

export class ServiceRestrictions {
    alone = $state(false);
    exclusive = $state<Array<string>>([]);
    glue = $state(false);
    leaf = $state(false);
    nearAlone = $state(false);
    needTypes = $state<Array<number>>([]);
    rootOnly = $state(false);
    single = $state(false);

    constructor(data?: Partial<ServiceRestrictions>) {
        if (data) {
            this.alone = data.alone ?? false;
            this.exclusive = data.exclusive ?? [];
            this.glue = data.glue ?? false;
            this.leaf = data.leaf ?? false;
            this.nearAlone = data.nearAlone ?? false;
            this.needTypes = data.needTypes ?? [];
            this.rootOnly = data.rootOnly ?? false;
            this.single = data.single ?? false;
        }
    }
}

export class ServiceInfos {
    name = $state("");
    _svctype = $state("");
    _svcicon = $state("");
    description = $state("");
    family = $state("");
    categories = $state<Array<string>>([]);
    record_types = $state<Array<number>>([]);
    tabs = $state(false);
    restrictions = $state(new ServiceRestrictions());

    constructor(data?: Partial<ServiceInfos>) {
        if (data) {
            this.name = data.name ?? "";
            this._svctype = data._svctype ?? "";
            this._svcicon = data._svcicon ?? "";
            this.description = data.description ?? "";
            this.family = data.family ?? "";
            this.categories = data.categories ?? [];
            this.record_types = data.record_types ?? [];
            this.tabs = data.tabs ?? false;
            this.restrictions = data.restrictions ? new ServiceRestrictions(data.restrictions) : new ServiceRestrictions();
        }
    }
}

export class ServiceSpec {
    fields = $state<null | Array<Field>>(null);

    constructor(data?: Partial<ServiceSpec>) {
        if (data) {
            this.fields = data.fields ?? null;
        }
    }
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

        if (!get(servicesSpecsLoaded)) return null;

        const sspecs = get(servicesSpecs);

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

export function newRecord(field: Field) {
    if (field.type.replace(/^(\[\])?\*?/, "").startsWith("dns.") || field.type.replace(/^(\[\])?\*?/, "").startsWith("happydns.")) {
        return newRR("", getRrtype(field.type.split(".")[1]));
    } else {
        return newRR("", 0);
    }
}

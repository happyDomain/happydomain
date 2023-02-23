import { get_store_value } from 'svelte/internal';

import type { Field } from '$lib/model/custom_form';
import { getAvailableResourceTypes, type ProviderInfos } from '$lib/model/provider';
import type { ServiceCombined } from '$lib/model/service';
import { servicesSpecs } from '$lib/stores/services';

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
    tabs: boolean;
    restrictions: ServiceRestrictions;
}

export interface ServiceSpec {
    fields: Array<Field>;
}

export function passRestrictions(svcinfo: ServiceInfos, provider_specs: ProviderInfos, zservices: Record<string, Array<ServiceCombined>>, dn: string): null | string {
    if (svcinfo.restrictions) {
        // Handle NeedTypes restriction: hosting provider need to support given types.
        if (svcinfo.restrictions.needTypes) {
            const availableResourceTypes = getAvailableResourceTypes(provider_specs);

            for (const needType of svcinfo.restrictions.needTypes) {
                if (availableResourceTypes.indexOf(needType) < 0) {
                    return 'is not available on svcinfo domain name hosting provider.';
                }
            }
        }

        // Handle rootOnly restriction.
        if (svcinfo.restrictions.rootOnly && dn !== '') {
            return 'can only be present at the root of your domain.';
        }

        if (zservices[dn] == null) return null;

        const sspecs = get_store_value(servicesSpecs);

        if (sspecs == null) return null;

        // Handle Alone restriction: only nearAlone are allowed.
        if (svcinfo.restrictions.alone) {
            for (const s of zservices[dn]) {
                if (s._svctype !== svcinfo._svctype && sspecs[s._svctype].restrictions && !sspecs[s._svctype].restrictions.nearAlone) {
                    return 'only one per subdomain.';
                }
            }
        }

        // Handle Exclusive restriction: service can't be present along with another listed one.
        if (svcinfo.restrictions.exclusive) {
            for (const s of zservices[dn]) {
                for (const exclu of svcinfo.restrictions.exclusive) {
                    if (s._svctype === exclu) {
                        return 'cannot coexist with ' + sspecs[s._svctype].name + '.';
                    }
                }
            }
        }

        // Check reverse Exclusivity
        for (const k in zservices[dn]) {
            const s = sspecs[zservices[dn][k]._svctype]
            if (!s.restrictions || !s.restrictions.exclusive) {
                continue
            }
            for (const i in s.restrictions.exclusive) {
                if (svcinfo._svctype === s.restrictions.exclusive[i]) {
                    return 'cannot coexist with ' + sspecs[s._svctype].name + '.';
                }
            }
        }

        // Handle Single restriction: only one instance of the service per subdomain.
        if (svcinfo.restrictions.single) {
            for (const s of zservices[dn]) {
                if (s._svctype === svcinfo._svctype) {
                    return 'can only be present once per subdomain.';
                }
            }
        }

        // Handle presence of Alone and Leaf service in subdomain already.
        let oneAlone: string | null = null
        let oneLeaf: string | null = null
        for (const s of zservices[dn]) {
            if (sspecs[s._svctype] && sspecs[s._svctype].restrictions && sspecs[s._svctype].restrictions.alone) {
                oneAlone = s._svctype
            }
            if (sspecs[s._svctype] && sspecs[s._svctype].restrictions && sspecs[s._svctype].restrictions.leaf) {
                oneLeaf = s._svctype
            }
        }
        if (oneAlone && oneAlone !== svcinfo._svctype && !svcinfo.restrictions.nearAlone) {
            return 'cannot coexist with ' + sspecs[oneAlone].name + ', that requires to be the only one in the subdomain.';
        }
        if (oneLeaf && oneLeaf !== svcinfo._svctype && !svcinfo.restrictions.glue) {
            return 'cannot coexist with ' + sspecs[oneLeaf].name + ', that cannot have subdomains.';
        }
    }

    return null;
}

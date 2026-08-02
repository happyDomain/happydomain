import { nsrrtype } from "$lib/dns";
import { serviceDescriptionOf, serviceNameOf } from "$lib/services/infos";
import type { HappydnsService } from "$lib/api-base/types.gen";
import { passRestrictions, type ServiceInfos } from "$lib/model/service_specs.svelte";
import type { ProviderInfos } from "$lib/model/provider";

export interface FilteredServices {
    available: ServiceInfos[];
    disabled: Array<{ svc: ServiceInfos; reason: string }>;
}

export function filterServices(
    servicesList: ServiceInfos[],
    providerSpecs: ProviderInfos,
    zservices: Record<string, Array<HappydnsService>>,
    dn: string,
    filteredName: string,
    filteredFamily: string | null = null,
    // Names and descriptions belong to the services and are translated, so the
    // caller hands over the current translation function.
    translate: (key: string) => string = (key) => key,
): FilteredServices {
    // Apply restrictions to all services
    const allServicesWithRestrictions = servicesList
        .filter(svc => svc.family !== "hidden")
        .map(svc => {
            const reason = passRestrictions(svc, providerSpecs, zservices, dn);
            return { svc, reason };
        });

    // Helper function to check if a service matches the filters
    function svc_match(svc: ServiceInfos): boolean {
        // Check family filter
        const familyMatch = filteredFamily == null || svc.family == filteredFamily;

        // Check name/description/record types/categories filter
        const nameMatch = !filteredName ||
            serviceNameOf(translate, svc).toLowerCase().indexOf(filteredName.toLowerCase()) >= 0 ||
            serviceDescriptionOf(translate, svc).toLowerCase().indexOf(filteredName.toLowerCase()) >= 0 ||
            (svc.record_types && svc.record_types.some((rtype) => nsrrtype(rtype).toLowerCase().indexOf(filteredName.toLowerCase()) >= 0)) ||
            (svc.categories && svc.categories.some((category) => category.toLowerCase().indexOf(filteredName.toLowerCase()) >= 0));

        return familyMatch && !!nameMatch;
    }

    // Separate available and disabled services, applying filters
    const available = allServicesWithRestrictions
        .filter(({ svc, reason }) => reason == null && svc_match(svc))
        .map(({ svc }) => svc);

    const disabled = allServicesWithRestrictions
        .filter(({ svc, reason }) => reason != null && svc_match(svc))
        .map(({ svc, reason }) => ({ svc, reason: reason! }));

    return { available, disabled };
}

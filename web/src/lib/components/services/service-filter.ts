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
    // null for an origin without provider, as a zone template.
    providerSpecs: ProviderInfos | null,
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

    // Searching "email" finds "e-mail": dashes are dropped on both sides.
    function normalize(text: string): string {
        return text.toLowerCase().replaceAll("-", "");
    }

    const term = normalize(filteredName);

    // Rank of a service against the search term, lower is more relevant, or
    // null when it does not match. Without a term every service ranks the same.
    function svc_rank(svc: ServiceInfos): number | null {
        if (filteredFamily != null && svc.family != filteredFamily) return null;
        if (!term) return 0;

        const name = normalize(serviceNameOf(translate, svc));
        const rtypes = (svc.record_types ?? []).map((rtype) => normalize(nsrrtype(rtype)));

        // Typing a record type, as "A", is asking for the service that holds it.
        if (name === term || rtypes.includes(term)) return 0;
        if (name.startsWith(term)) return 1;
        if (name.includes(term)) return 2;
        if (
            rtypes.some((rtype) => rtype.includes(term)) ||
            (svc.categories ?? []).some((category) => normalize(category).includes(term)) ||
            (svc.keywords ?? []).some((keyword) => normalize(keyword).includes(term))
        )
            return 3;
        if (normalize(serviceDescriptionOf(translate, svc)).includes(term)) return 4;
        return null;
    }

    // Without a term, services are listed alphabetically. Among equally
    // relevant services, the generic ones come first: they are the heaviest,
    // as the analyzer runs them last. Typing "SRV" thus shows the SRV record
    // before the services built on SRV records. The orphan record, the
    // heaviest of all, is the last resort: it always comes last.
    const isOrphan = (svc: ServiceInfos) => (svc._svctype === "svcs.Orphan" ? 1 : 0);
    const ranked = allServicesWithRestrictions
        .map(({ svc, reason }) => ({ svc, reason, rank: svc_rank(svc) }))
        .filter(({ rank }) => rank != null)
        .sort(
            (a, b) =>
                isOrphan(a.svc) - isOrphan(b.svc) ||
                (term
                    ? a.rank! - b.rank! || (b.svc.weight ?? 0) - (a.svc.weight ?? 0)
                    : serviceNameOf(translate, a.svc).localeCompare(serviceNameOf(translate, b.svc))),
        );

    // Separate available and disabled services, each in relevance order
    const available = ranked.filter(({ reason }) => reason == null).map(({ svc }) => svc);

    const disabled = ranked
        .filter(({ reason }) => reason != null)
        .map(({ svc, reason }) => ({ svc, reason: reason! }));

    return { available, disabled };
}

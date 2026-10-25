// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { getContext, setContext } from "svelte";

import type { ResolvedPathname } from "$app/types";

import { domainLinks } from "$links";
import {
    addZoneRecord,
    addZoneService,
    deleteZoneRecord,
    deleteZoneService,
    updateZoneRecord,
    updateZoneService,
} from "$lib/api/zone";
import type { HappydnsService } from "$lib/api-base/types.gen";
import type { dnsRR } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import type { Zone } from "$lib/model/zone";
import { domainLink } from "$lib/stores/domains";

const key = Symbol("zone-editor");

/**
 * How the zone in `thisZone` is saved and linked to: a domain zone or a zone
 * template. A change may give a zone with a new id, when a published domain
 * zone had to be derived first.
 */
export interface ZoneEditor {
    /** The zone of a domain, with checks and a history, rather than a template. */
    readonly isDomain: boolean;
    /** Nothing can be changed, as in a template offered to every user. */
    readonly readOnly: boolean;

    addService(zoneId: string, service: HappydnsService): Promise<Zone>;
    updateService(zoneId: string, service: HappydnsService): Promise<Zone>;
    deleteService(zoneId: string, service: HappydnsService): Promise<Zone>;

    addRecord(zoneId: string, subdomain: string, record: dnsRR): Promise<Zone>;
    updateRecord(zoneId: string, subdomain: string, newrr: dnsRR, oldrr: dnsRR): Promise<Zone>;
    deleteRecord(zoneId: string, subdomain: string, record: dnsRR): Promise<Zone>;

    /** The page showing the zone. */
    zoneHref(zoneId: string): ResolvedPathname;
    /** The page editing a service of the zone; "new" as serviceId adds one. */
    serviceHref(zoneId: string, subdomain: string, serviceId: string): ResolvedPathname;
}

/** The editor of the zones of a domain; domain is read at each call. */
export function domainZoneEditor(domain: () => Domain): ZoneEditor {
    const dn = () => encodeURIComponent(domainLink(domain().id));

    return {
        isDomain: true,
        // Read-only providers are checked by each component (isReadOnlyProvider).
        readOnly: false,

        addService: (zoneId, service) => addZoneService(domain(), zoneId, service),
        updateService: (zoneId, service) => updateZoneService(domain(), zoneId, service),
        deleteService: (zoneId, service) => deleteZoneService(domain(), zoneId, service),

        addRecord: (zoneId, subdomain, record) =>
            addZoneRecord(domain(), zoneId, subdomain, record),
        updateRecord: (zoneId, subdomain, newrr, oldrr) =>
            updateZoneRecord(domain(), zoneId, subdomain, newrr, oldrr),
        deleteRecord: (zoneId, subdomain, record) =>
            deleteZoneRecord(domain(), zoneId, subdomain, record),

        zoneHref: (zoneId) => domainLinks().zone(dn(), encodeURIComponent(zoneId)),
        serviceHref: (zoneId, subdomain, serviceId) =>
            domainLinks().service(
                dn(),
                encodeURIComponent(zoneId),
                encodeURIComponent(subdomain === "" ? "@" : subdomain),
                encodeURIComponent(serviceId),
            ),
    };
}

/** Shares the editor of the zone with the components of the page. */
export function provideZoneEditor(editor: ZoneEditor): ZoneEditor {
    setContext(key, editor);
    return editor;
}

/** The context giving editor to a component mounted on its own, as in tests. */
export function zoneEditorContext(editor: ZoneEditor): Map<symbol, ZoneEditor> {
    return new Map([[key, editor]]);
}

/**
 * The editor of the zone the component shows: the one provided by the page,
 * or else the one of the zones of origin, a domain.
 */
export function useZoneEditor(origin: () => Domain): ZoneEditor {
    return getContext<ZoneEditor | undefined>(key) ?? domainZoneEditor(origin);
}

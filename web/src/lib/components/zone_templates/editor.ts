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

import { templateLinks } from "$links";
import type { ZoneEditor } from "$lib/components/zones/editor";
import type { Zone } from "$lib/model/zone";
import { templateAsZone, type ZoneTemplate } from "$lib/model/zone_template";
import type { ZoneTemplateBackend } from "./backend";

/**
 * The editor of a zone template: the identifier of the zone is the one of
 * the template. backend and readOnly are read at each call. Each change is
 * given to saved, then shown as a zone.
 */
export function templateZoneEditor(
    backend: () => ZoneTemplateBackend,
    readOnly: () => boolean,
    saved: (t: ZoneTemplate) => void,
): ZoneEditor {
    const show = async (change: Promise<ZoneTemplate>): Promise<Zone> => {
        const t = await change;
        saved(t);
        return templateAsZone(t);
    };

    return {
        isDomain: false,
        get readOnly() {
            return readOnly();
        },

        addService: (id, service) => show(backend().addService(id, service)),
        updateService: (id, service) => show(backend().updateService(id, service)),
        deleteService: (id, service) => show(backend().deleteService(id, service)),

        addRecord: (id, subdomain, record) => show(backend().addRecord(id, subdomain, record)),
        updateRecord: (id, subdomain, newrr, oldrr) =>
            show(backend().updateRecord(id, subdomain, newrr, oldrr)),
        deleteRecord: (id, subdomain, record) =>
            show(backend().deleteRecord(id, subdomain, record)),

        zoneHref: (id) => templateLinks().template(encodeURIComponent(id)),
        serviceHref: (id, subdomain, serviceId) =>
            templateLinks().service(
                encodeURIComponent(id),
                encodeURIComponent(subdomain === "" ? "@" : subdomain),
                encodeURIComponent(serviceId),
            ),
    };
}

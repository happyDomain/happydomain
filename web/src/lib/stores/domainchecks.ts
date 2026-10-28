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

import { derived, writable } from "svelte/store";

import { loadCheckSections, type CheckSection } from "$lib/domain_checks";
import type { Zone } from "$lib/model/zone";

export interface DomainChecks {
    domainId: string;
    // The zone whose services were checked.
    zone: Zone;
    sections: Promise<CheckSection[]>;
}

// The checks of the domain being browsed, shared by its pages and sidebars.
export const domainChecks = writable<DomainChecks | null>(null);

// The same checks once loaded, by subdomain: "" is the apex and the domain.
export const checksBySubdomain = derived(
    domainChecks,
    (
        $domainChecks,
        set: (v: { domainId: string; sections: Map<string, CheckSection> } | null) => void,
    ) => {
        // The previous results stay until these come, sparing a flicker.
        if (!$domainChecks) return;
        let current = true;
        $domainChecks.sections.then(
            (sections) => {
                if (!current) return;
                set({
                    domainId: $domainChecks.domainId,
                    sections: new Map(
                        sections.map((s) => [s.subdomain === "@" ? "" : s.subdomain, s]),
                    ),
                });
            },
            // The dashboard tells about the failure.
            () => {},
        );
        return () => {
            current = false;
        };
    },
    null,
);

export function refreshDomainChecks(domainId: string, zone: Zone): void {
    domainChecks.set({ domainId, zone, sections: loadCheckSections(domainId, zone) });
}

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

import { listScopedCheckers, type CheckerScope } from "$lib/api/checkers";
import type { HappydnsCheckerStatus, HappydnsStatus } from "$lib/api-base/types.gen";
import { domainCompare } from "$lib/dns";
import type { Zone } from "$lib/model/zone";

/** The service of the zone a check is about; none for the domain itself. */
export interface CheckTarget {
    // "@" for the apex.
    subdomain: string;
    serviceId: string;
}

export interface CheckItem {
    status: HappydnsCheckerStatus;
    scope: CheckerScope;
    target?: CheckTarget;
    // The service checked, to tell apart the checks of a subdomain.
    chip?: string;
}

/** The checks about a subdomain, "@" being the apex and the domain itself. */
export interface CheckSection {
    subdomain: string;
    items: CheckItem[];
}

function serviceLabel(svc: { _id?: string; _svctype?: string }): string {
    return svc._svctype || svc._id || "service";
}

// Severity weight: bigger means more important. not-run lands at the bottom.
function severityWeight(status: HappydnsCheckerStatus): number {
    const code = status.latestExecution?.result?.status;
    if (code === undefined || code === null) return -1;
    return code as number;
}

function sortItemsBySeverityThenName(items: CheckItem[]): CheckItem[] {
    return [...items].sort((a, b) => {
        const diff = severityWeight(b.status) - severityWeight(a.status);
        if (diff !== 0) return diff;
        const an = a.status.name || a.status.id || "";
        const bn = b.status.name || b.status.id || "";
        return an.localeCompare(bn);
    });
}

/** The worst status of the checks that ran; undefined when none did. */
export function worstStatus(items: CheckItem[]): HappydnsStatus | undefined {
    let worst: HappydnsStatus | undefined;
    for (const item of items) {
        const code = item.status.latestExecution?.result?.status;
        if (code !== undefined && code !== null && (worst === undefined || code > worst)) {
            worst = code;
        }
    }
    return worst;
}

/**
 * The checks of a domain and of the services of its zone, grouped by
 * subdomain. A subdomain without any check has no section.
 */
export async function loadCheckSections(
    domainId: string,
    zone: Zone | null,
): Promise<CheckSection[]> {
    const grouped = new Map<string, CheckSection>();

    const ensureSection = (subdomain: string): CheckSection => {
        let s = grouped.get(subdomain);
        if (!s) {
            s = { subdomain, items: [] };
            grouped.set(subdomain, s);
        }
        return s;
    };

    const domainStatuses = await listScopedCheckers({ domainId });
    if (domainStatuses.length > 0) {
        const sec = ensureSection("@");
        for (const s of domainStatuses) {
            sec.items.push({ status: s, scope: { domainId } });
        }
    }

    if (zone) {
        const targets: Array<CheckTarget & { serviceName: string }> = [];
        for (const [subdomain, services] of Object.entries(zone.services ?? {})) {
            for (const svc of services ?? []) {
                if (!svc._id) continue;
                targets.push({
                    subdomain: subdomain === "" ? "@" : subdomain,
                    serviceId: svc._id,
                    serviceName: serviceLabel(svc),
                });
            }
        }

        const serviceLists = await Promise.all(
            targets.map((tg) =>
                listScopedCheckers({
                    domainId,
                    zoneId: zone.id!,
                    subdomain: tg.subdomain,
                    serviceId: tg.serviceId,
                }).catch(() => [] as HappydnsCheckerStatus[]),
            ),
        );

        targets.forEach((tg, i) => {
            const statuses = serviceLists[i];
            if (statuses.length === 0) return;
            const sec = ensureSection(tg.subdomain);
            const scope: CheckerScope = {
                domainId,
                zoneId: zone.id!,
                subdomain: tg.subdomain,
                serviceId: tg.serviceId,
            };
            const target = { subdomain: tg.subdomain, serviceId: tg.serviceId };
            for (const s of statuses) {
                sec.items.push({ status: s, scope, target, chip: tg.serviceName });
            }
        });
    }

    const sections = Array.from(grouped.values());
    for (const s of sections) s.items = sortItemsBySeverityThenName(s.items);
    // In the order of the zone: the apex first, each subdomain under its parent.
    sections.sort((a, b) =>
        domainCompare(
            a.subdomain === "@" ? "" : a.subdomain,
            b.subdomain === "@" ? "" : b.subdomain,
        ),
    );
    return sections;
}

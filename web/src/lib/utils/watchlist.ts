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

// Helpers of the domain watchlist: how each phase looks, how urgent it is,
// and what to say about a watched domain.

import type { DomainWatch, DomainWatchPhase } from "$lib/model/watchlist";
import { daysUntil } from "./domaininfo";

/** Phases from the most to the least urgent for someone wanting the domain. */
export const PHASE_ORDER: DomainWatchPhase[] = [
    "available",
    "pending_delete",
    "redemption",
    "grace",
    "expiring",
    "registered",
    "unknown",
];

/** Bootstrap color and icon of each phase. */
export const PHASE_STYLE: Record<DomainWatchPhase, { color: string; icon: string }> = {
    available: { color: "success", icon: "check-circle-fill" },
    pending_delete: { color: "danger", icon: "hourglass-bottom" },
    redemption: { color: "warning", icon: "hourglass-split" },
    grace: { color: "warning", icon: "calendar-x" },
    expiring: { color: "info", icon: "calendar-event" },
    registered: { color: "secondary", icon: "lock-fill" },
    unknown: { color: "light", icon: "question-circle" },
};

/** Bootstrap color and icon of a domain the registry auto-renewed. */
export const AUTO_RENEW_STYLE = { color: "info", icon: "arrow-repeat" };

/** Steps of the life cycle a domain goes through when its owner lets it go. */
export const LIFECYCLE_STEPS: DomainWatchPhase[] = [
    "registered",
    "grace",
    "redemption",
    "pending_delete",
    "available",
];

/**
 * Index of the step of the phase in LIFECYCLE_STEPS (an expiring domain is
 * still registered), -1 when unknown.
 */
export function lifecycleStep(phase: DomainWatchPhase | undefined): number {
    if (phase === "expiring") return 0;
    return LIFECYCLE_STEPS.indexOf(phase ?? "unknown");
}

/**
 * Whether the registry renewed the domain on its own, at its expiration date,
 * and waits for the registrar to confirm the renewal or to delete the domain.
 */
export function isAutoRenewPending(w: DomainWatch): boolean {
    return hasStatus(w.status, "autoRenewPeriod");
}

// Compared however it is spelled, as the server's domaininfo.HasStatus does.
function hasStatus(status: string[] | undefined, code: string): boolean {
    const want = code.toLowerCase();
    return (status ?? []).some((s) => s.toLowerCase().replace(/[^a-z]/g, "") === want);
}

/**
 * The watch of the domain among the list. The watch is on the registrable
 * domain: a subdomain matches it.
 */
export function findWatch(
    list: DomainWatch[] | undefined,
    domain: string,
): DomainWatch | undefined {
    const name = domain.toLowerCase().replace(/\.$/, "");
    return list?.find(
        (w) =>
            name === w.domain ||
            name === w.displayName ||
            name.endsWith("." + w.domain) ||
            name.endsWith("." + w.displayName),
    );
}

function phaseRank(w: DomainWatch): number {
    const i = PHASE_ORDER.indexOf(w.phase ?? "unknown");
    return i < 0 ? PHASE_ORDER.length : i;
}

function dateRank(w: DomainWatch): number {
    const d = w.estimatedRelease ?? w.expiration;
    return d ? new Date(d).getTime() : Number.POSITIVE_INFINITY;
}

/** Name of a watched domain to show, in Unicode for an internationalized one. */
export function nameOf(w: DomainWatch): string {
    return w.displayName || w.domain || "";
}

/** Sort order of the watchlist: most urgent first, then soonest. */
export function compareWatches(a: DomainWatch, b: DomainWatch): number {
    return (
        phaseRank(a) - phaseRank(b) ||
        dateRank(a) - dateRank(b) ||
        nameOf(a).localeCompare(nameOf(b))
    );
}

export type WatchSort = "urgency" | "name" | "expiration" | "added";

export const WATCH_SORTS: { key: WatchSort; icon: string }[] = [
    { key: "urgency", icon: "exclamation-circle" },
    { key: "name", icon: "sort-alpha-down" },
    { key: "expiration", icon: "calendar-event" },
    { key: "added", icon: "clock-history" },
];

/** Comparison function of the watchlist for the given sort order. */
export function watchComparator(sort: WatchSort): (a: DomainWatch, b: DomainWatch) => number {
    switch (sort) {
        case "name":
            return (a, b) => nameOf(a).localeCompare(nameOf(b));
        case "expiration":
            return (a, b) =>
                (a.expiration ? new Date(a.expiration).getTime() : Infinity) -
                    (b.expiration ? new Date(b.expiration).getTime() : Infinity) ||
                compareWatches(a, b);
        case "added":
            return (a, b) =>
                (b.createdAt ? new Date(b.createdAt).getTime() : 0) -
                    (a.createdAt ? new Date(a.createdAt).getTime() : 0) || compareWatches(a, b);
        default:
            return compareWatches;
    }
}

/**
 * Translation key and parameters of the headline of a watched domain: what
 * matters most about it right now.
 */
export function watchHeadline(w: DomainWatch): [string, { days?: number }] {
    switch (w.phase) {
        case "available":
            return ["watchlist.headline.available", {}];
        case "pending_delete":
        case "redemption": {
            if (!w.estimatedRelease) return ["watchlist.headline.release-soon", {}];
            const days = daysUntil(w.estimatedRelease);
            if (days <= 0) return ["watchlist.headline.release-imminent", {}];
            return ["watchlist.headline.release-in", { days }];
        }
        case "grace": {
            if (!w.expiration) return ["watchlist.headline.expired", {}];
            return [
                "watchlist.headline.expired-ago",
                { days: Math.max(0, -daysUntil(w.expiration)) },
            ];
        }
        case "expiring":
        case "registered": {
            if (!w.expiration) return ["watchlist.headline.no-expiration", {}];
            const days = daysUntil(w.expiration);
            if (days <= 0) return ["watchlist.headline.expires-today", {}];
            return ["watchlist.headline.expires-in", { days }];
        }
        default:
            if (w.lastError) return ["watchlist.headline.lookup-failed", {}];
            return ["watchlist.headline.not-checked", {}];
    }
}

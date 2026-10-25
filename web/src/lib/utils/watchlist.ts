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

import type { DomainInfo } from "$lib/model/domaininfo";
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
 * How close to its expiration date a registered domain is said to be expiring,
 * as expiringWithin of internal/usecase/domainwatch.
 */
const EXPIRING_WITHIN_MS = 30 * 86400000;

/**
 * Phase of a domain from its registration data, info being null when it is not
 * registered. Mirrors domaininfo.PhaseOf (pkg/domaininfo/lifecycle.go) and
 * phaseOf of internal/usecase/domainwatch, which tells the expiring domains
 * apart: keep them in sync.
 */
export function phaseOfDomainInfo(info: DomainInfo | null): DomainWatchPhase {
    if (!info) return "available";
    if (hasStatus(info.status, "pendingDelete")) return "pending_delete";
    if (hasStatus(info.status, "redemptionPeriod") || hasStatus(info.status, "pendingRestore"))
        return "redemption";
    if (info.expiration) {
        const left = new Date(info.expiration).getTime() - Date.now();
        if (left < 0) return "grace";
        if (left <= EXPIRING_WITHIN_MS) return "expiring";
    }
    return "registered";
}

/**
 * A watch, not saved, describing a domain from a single lookup: info is its
 * registration data, null when it is not registered.
 */
export function watchOfDomainInfo(domain: string, info: DomainInfo | null): DomainWatch {
    return {
        domain: info?.name?.replace(/\.$/, "") || domain,
        phase: phaseOfDomainInfo(info),
        expiration: info?.expiration,
        registrar: info?.registrar,
        status: info?.status,
    };
}

/**
 * The watch w of domain, updated with what a lookup of domain just told: the
 * watch is only checked so often. The lookup of a subdomain tells nothing of
 * the watched domain.
 */
export function updatedWatch(w: DomainWatch, domain: string, lookedUp: DomainWatch): DomainWatch {
    const name = domain.toLowerCase().replace(/\.$/, "");
    if (name !== w.domain && name !== w.displayName) return w;
    return {
        ...w,
        phase: lookedUp.phase,
        expiration: lookedUp.expiration,
        registrar: lookedUp.registrar,
        status: lookedUp.status,
        // Estimated from when the phase was first seen.
        estimatedRelease: lookedUp.phase === w.phase ? w.estimatedRelease : undefined,
    };
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

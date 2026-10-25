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

// Helpers shared by the views of the registration information of a domain
// (RDAP/WHOIS): its registry statuses and its expiration date.

const DAY_MS = 86400000;

/**
 * Bootstrap color of a registry status badge.
 * @param code EPP status code, such as "clientTransferProhibited"
 */
export function statusColor(code: string): string {
    const lc = code.toLowerCase();
    if (lc === "ok" || lc === "active") return "success";
    if (lc.includes("hold")) return "danger";
    if (lc.includes("prohibited") || lc.includes("pending")) return "info";
    return "secondary";
}

/**
 * Number of whole days from now until the date, negative once it is past.
 * @param date ISO 8601 date string or Date
 */
export function daysUntil(date: string | Date): number {
    return Math.round((new Date(date).getTime() - Date.now()) / DAY_MS);
}

/**
 * Bootstrap color telling how close an expiration date is.
 * @param expiration ISO 8601 date string or Date, if known
 */
export function expirationColor(expiration?: string | Date): string {
    if (!expiration) return "secondary";
    const days = daysUntil(expiration);
    if (days < 30) return "danger";
    if (days < 90) return "warning";
    return "success";
}

/**
 * Share of a year left before an expiration date, as a percentage.
 * @param expiration ISO 8601 date string or Date, if known
 */
export function expirationProgress(expiration?: string | Date): number {
    if (!expiration) return 0;
    const days = daysUntil(expiration);
    if (days <= 0) return 0;
    return Math.min(100, Math.round((days / 365) * 100));
}

/**
 * Translation key and parameters saying how far an expiration date is.
 * @param days Days until the expiration, as returned by daysUntil
 */
export function expirationLabel(days: number): [string, { days?: number }] {
    if (days < 0) return ["domaininfo.expired", { days: Math.abs(days) }];
    if (days === 0) return ["domaininfo.expires-today", {}];
    return ["domaininfo.expires-in", { days }];
}

/**
 * Format a date as a long localized date, such as "June 1, 2027".
 * @param date ISO 8601 date string or Date, if known
 */
export function formatDate(date?: string | Date): string {
    if (!date) return "";
    return new Date(date).toLocaleDateString(undefined, {
        year: "numeric",
        month: "long",
        day: "numeric",
    });
}

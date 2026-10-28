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

import { writable } from "svelte/store";

import type { DomainLog } from "$lib/model/domain";

// Levels follow model/logs.go: the lower, the more severe.
const LOG_ERR = 3;
const LOG_WEIRD = 6;
const LOG_ACK = 9;

export type LogSeverity = "critical" | "error" | "warning" | "success" | "info" | "debug";

export function logSeverity(level: number): LogSeverity {
    if (level > LOG_ACK) return "debug";
    if (level === LOG_ACK) return "success";
    if (level > LOG_WEIRD) return "info";
    if (level > LOG_ERR) return "warning";
    if (level > 1) return "error";
    return "critical";
}

// Anything from a weird situation up to a crash deserves a look.
export function isProblem(log: DomainLog): boolean {
    return log.level <= LOG_WEIRD;
}

// The day of the log in the local timezone, as YYYY-MM-DD: it doubles as
// the anchor of the day on the logs page.
export function logDay(date: Date | string): string {
    const d = new Date(date);
    return [
        d.getFullYear(),
        String(d.getMonth() + 1).padStart(2, "0"),
        String(d.getDate()).padStart(2, "0"),
    ].join("-");
}

// Reads back a logDay() as the local date it is, not as UTC midnight.
export function parseLogDay(day: string): Date {
    const [y, m, d] = day.split("-").map(Number);
    return new Date(y, m - 1, d);
}

export interface LogDay {
    day: string;
    logs: DomainLog[];
    // The lowest level of the day, so its most severe entry.
    worst: number;
}

// Groups logs by day, the most recent first, whatever order they came in.
export function groupLogsByDay(logs: DomainLog[]): LogDay[] {
    const sorted = [...logs].sort(
        (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime(),
    );

    const res: LogDay[] = [];
    for (const log of sorted) {
        const day = logDay(log.date);
        if (res.length === 0 || res[res.length - 1].day !== day) {
            res.push({ day, logs: [], worst: log.level });
        }
        const group = res[res.length - 1];
        group.logs.push(log);
        group.worst = Math.min(group.worst, log.level);
    }
    return res;
}

// Imports and publications name the zone version they are about.
export function logZoneVersion(
    log: DomainLog,
    zoneHistory: string[] | undefined,
): string | undefined {
    return zoneHistory?.find((zid) => log.content.includes(zid));
}

// The day shown at the top of the logs page, followed by the sidebar.
export const activeLogDay = writable<string | undefined>(undefined);

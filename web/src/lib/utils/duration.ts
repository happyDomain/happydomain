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

// Units of the BIND time format, largest first.
const UNITS: [string, Intl.NumberFormatOptions["unit"], number][] = [
    ["w", "week", 604800],
    ["d", "day", 86400],
    ["h", "hour", 3600],
    ["m", "minute", 60],
    ["s", "second", 1],
];

/**
 * Parse a duration written in the BIND time format ("1w2d", "3h 30m", "90").
 * A bare number counts seconds.
 * @param input Text typed by the user
 * @returns Number of seconds, or null when the text is not a duration
 */
export function parseBindDuration(input: string): number | null {
    const text = input.trim().toLowerCase();
    if (/^\d+$/.test(text)) return parseInt(text, 10);
    if (!/^(\d+\s*[wdhms]\s*)+$/.test(text)) return null;

    let total = 0;
    for (const [, count, unit] of text.matchAll(/(\d+)\s*([wdhms])/g)) {
        total += parseInt(count, 10) * UNITS.find(([u]) => u === unit)![2];
    }
    return total;
}

function splitDuration(seconds: number): [number, (typeof UNITS)[number]][] {
    const parts: [number, (typeof UNITS)[number]][] = [];
    let rest = seconds;
    for (const unit of UNITS) {
        const count = Math.floor(rest / unit[2]);
        if (count > 0) parts.push([count, unit]);
        rest %= unit[2];
    }
    return parts;
}

/**
 * Write a number of seconds in the shortest BIND time format ("1w", "1h30m").
 * @param seconds Duration in seconds
 * @returns BIND time format string
 */
export function formatBindDuration(seconds: number): string {
    if (seconds === 0) return "0";
    return splitDuration(seconds)
        .map(([count, [u]]) => `${count}${u}`)
        .join("");
}

/**
 * Spell out a number of seconds in the given locale ("1 week, 2 days").
 * @param seconds Duration in seconds
 * @param locale BCP 47 language tag; the browser's when undefined
 * @returns Localized duration
 */
export function formatHumanDuration(seconds: number, locale?: string): string {
    const parts = seconds === 0 ? [[0, UNITS[UNITS.length - 1]] as const] : splitDuration(seconds);
    return new Intl.ListFormat(locale, { style: "long", type: "unit" }).format(
        parts.map(([count, [, unit]]) =>
            new Intl.NumberFormat(locale, { style: "unit", unit, unitDisplay: "long" }).format(
                count,
            ),
        ),
    );
}

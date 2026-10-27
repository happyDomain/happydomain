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

// Names are kept as the user writes them, as in a zone file: relative to an
// origin unless they end with a dot, "@" standing for the origin itself. Only
// the backend turns names within the origin into relative ones.

/**
 * Turn what the user typed into the name to store. The final dot is what
 * tells an absolute name from a relative one, as in a zone file: nothing is
 * guessed from the origin, "www.example.com" stays relative to it.
 * @param text Text typed by the user
 * @returns Name to store
 */
export function textToDomainName(text: string): string {
    return text.trim();
}

// A short list of the TLDs people use the most: enough to notice an absolute
// name typed without its final dot, without flagging www.blog or mail.eu.backup.
const COMMON_TLDS = new Set([
    "app",
    "arpa",
    "asia",
    "biz",
    "cloud",
    "com",
    "dev",
    "edu",
    "email",
    "gov",
    "info",
    "int",
    "mil",
    "name",
    "net",
    "online",
    "org",
    "pro",
    "site",
    "tech",
    "xyz",
]);

/**
 * Tell whether a relative name looks like an absolute one whose final dot was
 * forgotten: several labels, the last one being a common or a country TLD.
 * @param name Name as stored
 * @returns true when the name should be suggested as absolute
 */
export function looksAbsolute(name: string): boolean {
    if (name === "" || name === "@" || name.endsWith(".")) return false;
    const labels = name.split(".");
    if (labels.length < 2) return false;
    const tld = labels[labels.length - 1].toLowerCase();
    return COMMON_TLDS.has(tld) || /^[a-z]{2}$/.test(tld);
}

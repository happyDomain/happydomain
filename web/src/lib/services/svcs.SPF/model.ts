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

export interface SPFValue {
    /** Version, without the "v=" prefix. Defaults to "spf1". */
    v?: string;
    /** Directives and modifiers, in order, exactly as they appear in the record. */
    f: string[];
}

export function parseSPF(val: string): SPFValue {
    const trimmed = val.trim();
    if (!trimmed) return { v: undefined, f: [] };

    // SPF terms are space-separated (RFC 7208 sec. 4). Semicolons have no
    // syntactic role in SPF, but they are the separator used by DKIM, DMARC,
    // and other key=value TXT records. Splitting on both keeps an SPF parse
    // robust when foreign residue lands in the same TXT slot.
    const fields = trimmed.split(/[\s;]+/);
    const first = fields[0] ?? "";
    if (/^v=/i.test(first)) {
        return {
            v: first.replace(/^v=/i, ""),
            f: fields.slice(1),
        };
    }
    // No version prefix at the head: keep everything as directives so the
    // validator can flag the missing version.
    return { v: undefined, f: fields };
}

export function stringifySPF(val: SPFValue): string {
    return "v=" + (val.v ? val.v : "spf1") + (val.f.length ? " " + val.f.join(" ") : "");
}

// SPF mechanisms that consume a DNS lookup per RFC 7208 §4.6.4.
export const LOOKUP_MECHANISMS = ["include", "a", "mx", "ptr", "exists"] as const;
type LookupMechanism = (typeof LOOKUP_MECHANISMS)[number];

// Mechanisms defined by RFC 7208 sec. 5. Anything else is a typo.
export const KNOWN_MECHANISMS = new Set<string>([
    "all",
    "include",
    "a",
    "mx",
    "ptr",
    "ip4",
    "ip6",
    "exists",
]);

// Modifiers explicitly defined by the SPF RFCs. Unknown modifiers are allowed
// per RFC 7208 sec. 6 but are almost always typos in practice, so we surface
// them as warnings.
export const KNOWN_MODIFIERS = new Set<string>(["redirect", "exp"]);

const IPV4_RE = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/;
const IPV6_GROUP_RE = /^[0-9A-Fa-f]{1,4}$/;

/**
 * Dotted-quad, as SPF spells addresses (RFC 7208 sec. 5.6). Leading zeros are
 * refused: resolvers disagree on whether they mean octal, so an address
 * carrying them does not match the same hosts everywhere.
 */
export function isIPv4(s: string): boolean {
    const m = IPV4_RE.exec(s);
    if (!m) return false;
    return m.slice(1).every((o) => (o.length === 1 || o[0] !== "0") && Number(o) <= 255);
}

/**
 * The groups of one side of a "::", or null when one of them is not a group.
 * An embedded IPv4 (::ffff:192.0.2.1) stands for the last two groups, and is
 * only accepted at the very end of the address.
 */
function ipv6Groups(part: string, allowEmbeddedIPv4: boolean): string[] | null {
    if (part === "") return [];

    const items = part.split(":");
    const groups: string[] = [];
    for (let i = 0; i < items.length; i++) {
        const item = items[i];
        if (item.includes(".")) {
            if (!allowEmbeddedIPv4 || i !== items.length - 1 || !isIPv4(item)) return null;
            groups.push("0", "0");
            continue;
        }
        if (!IPV6_GROUP_RE.test(item)) return null;
        groups.push(item);
    }
    return groups;
}

export function isIPv6(s: string): boolean {
    const halves = s.split("::");
    if (halves.length > 2) return false;

    if (halves.length === 1) {
        const groups = ipv6Groups(s, true);
        return groups !== null && groups.length === 8;
    }

    const left = ipv6Groups(halves[0], false);
    const right = ipv6Groups(halves[1], true);
    if (left === null || right === null) return false;
    // "::" stands for at least one omitted group, so both sides never total 8.
    return left.length + right.length <= 7;
}

export interface ParsedTerm {
    raw: string;
    qualifier?: "+" | "-" | "~" | "?";
    /**
     * Mechanism or modifier name, lower-cased. For modifiers (e.g. "redirect")
     * `isModifier` is true.
     */
    name: string;
    value?: string;
    isModifier: boolean;
    isAll: boolean;
    consumesLookup: boolean;
    /**
     * First character of the term, when it is neither the start of a name nor
     * one of the four qualifiers. Set for e.g. "!all" or "*include:x".
     */
    badQualifier?: string;
    /**
     * A name=value term carrying a qualifier. Qualifiers belong to mechanisms
     * only, so this is never a valid modifier.
     */
    qualifiedModifier?: boolean;
}

export function parseTerm(raw: string): ParsedTerm {
    let s = raw;
    let qualifier: ParsedTerm["qualifier"];
    let badQualifier: string | undefined;
    if (s.length > 0 && (s[0] === "+" || s[0] === "-" || s[0] === "~" || s[0] === "?")) {
        qualifier = s[0] as ParsedTerm["qualifier"];
        s = s.slice(1);
    } else if (s.length > 0 && !/[A-Za-z0-9]/.test(s[0])) {
        badQualifier = s[0];
        s = s.slice(1);
    }

    // A modifier has the form name=value, but mechanisms may also carry a value
    // after a colon (e.g. include:domain.tld) or an equal sign in some legacy
    // forms. Modifiers per RFC: redirect=, exp=, plus unknown ones.
    const eqIdx = s.indexOf("=");
    const colonIdx = s.indexOf(":");
    const slashIdx = s.indexOf("/");

    let isModifier = false;
    let qualifiedModifier = false;
    let name = s;
    let value: string | undefined;

    if (
        eqIdx !== -1 &&
        (colonIdx === -1 || eqIdx < colonIdx) &&
        (slashIdx === -1 || eqIdx < slashIdx)
    ) {
        isModifier = qualifier === undefined;
        qualifiedModifier = !isModifier;
        name = s.slice(0, eqIdx);
        value = s.slice(eqIdx + 1);
    } else if (colonIdx !== -1) {
        name = s.slice(0, colonIdx);
        value = s.slice(colonIdx + 1);
    } else if (slashIdx !== -1) {
        name = s.slice(0, slashIdx);
        value = s.slice(slashIdx);
    }

    name = name.toLowerCase();
    const isAll = !isModifier && name === "all";
    const consumesLookup =
        (!isModifier && (LOOKUP_MECHANISMS as readonly string[]).includes(name)) ||
        (isModifier && name === "redirect");

    return {
        raw,
        qualifier,
        name,
        value,
        isModifier,
        isAll,
        consumesLookup,
        badQualifier,
        qualifiedModifier,
    };
}

export interface SPFLookupBudget {
    /** Number of mechanisms / modifiers that count toward the 10-lookup limit, locally. */
    count: number;
    /** Items contributing to the budget, with their indices in `val.f`. */
    contributors: { index: number; mechanism: LookupMechanism | "redirect" }[];
}

export function countLocalLookups(val: SPFValue): SPFLookupBudget {
    const contributors: SPFLookupBudget["contributors"] = [];
    val.f.forEach((raw, index) => {
        const term = parseTerm(raw);
        if (!term.consumesLookup) return;
        contributors.push({
            index,
            mechanism: (term.isModifier ? "redirect" : term.name) as LookupMechanism | "redirect",
        });
    });
    return { count: contributors.length, contributors };
}

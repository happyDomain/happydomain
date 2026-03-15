// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2024 happyDomain
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

import type { dnsRR } from "$lib/dns_rr";
import { newRR, nsrrtype, rdatatostr } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";

export { nsrrtype, rdatatostr };

export const dns_common_types: Array<string> = [
    "ANY",
    "A",
    "AAAA",
    "NS",
    "SRV",
    "MX",
    "TXT",
    "SOA",
];

/**
 * Resolves a domain name label to a fully qualified domain name (FQDN).
 *
 * - `"@"` and `""` are treated as the zone apex and return `origin` as-is.
 * - A label already ending with `"."` is already absolute and returned unchanged.
 * - Otherwise the label is appended to `origin` with a separating dot.
 *
 * @param input  - The relative label or special value (`"@"`, `""`).
 * @param origin - The zone origin (should itself be a FQDN ending with `"."`).
 * @returns The absolute domain name.
 */
export function fqdn(input: string, origin: string) {
    if (input === "@") {
        return origin;
    } else if (input.endsWith(".")) {
        return input;
    } else if (input === "") {
        return origin;
    } else {
        return input + "." + origin;
    }
}

/**
 * Joins domain labels into a single domain name.
 *
 * The labels are processed from left to right and appended with `"."`
 * separators until one of the following conditions stops the process:
 *
 * - `"@"` stops processing and represents the zone apex.
 * - `""` is ignored and skipped.
 * - If the resulting string ends with `"."`, it is considered absolute
 *   and no further labels are appended.
 *
 * The resulting domain is returned without a leading `"."`.
 *
 * @param domains - Domain labels to join (left to right).
 * @returns The combined domain name.
 */
export function domainJoin(...domains: string[]): string {
    let ret = "";

    for (const d of domains) {
        if (d === "@") {
            break;
        } else if (d !== "") {
            ret += "." + d;
        }

        if (ret.length > 0 && ret[ret.length - 1] === ".") {
            break;
        }
    }

    if (ret.length >= 1) {
        ret = ret.slice(1);
    }

    return ret;
}

/**
 * Compares two domain names from root to leaf for use in sort functions.
 *
 * Labels are compared right-to-left (TLD first), case-insensitively, so that
 * sibling zones are grouped together. Shorter domains sort before longer ones
 * when all their labels match.
 *
 * @param a - First domain name (string, `Domain`, or `{ domain: string }`).
 * @param b - Second domain name.
 * @returns Negative if `a < b`, positive if `a > b`, zero if equal.
 */
export function domainCompare(
    a: string | Domain | { domain: string },
    b: string | Domain | { domain: string },
) {
    // Convert to string if Domain
    let domainA = typeof a === "object" ? a.domain : a;
    let domainB = typeof b === "object" ? b.domain : b;

    const as = domainA.split(".").reverse();
    const bs = domainB.split(".").reverse();

    // Remove first item if empty
    if (!as[0].length) as.shift();
    if (!bs[0].length) bs.shift();

    const maxDepth = Math.min(as.length, bs.length);
    for (let i = 0; i < maxDepth; i++) {
        const cmp = as[i].toLowerCase().localeCompare(bs[i].toLowerCase());
        if (cmp !== 0) {
            return cmp;
        }
    }

    return as.length - bs.length;
}

/**
 * Compares two FQDNs with zone-aware ordering for use in sort functions.
 *
 * Similar to {@link domainCompare} but skips the TLD comparison so that names
 * within the same zone are ordered by their subdomain labels first, then by
 * the apex label. This keeps zone records grouped in a natural tree order.
 *
 * @param a - First domain name (string, `Domain`, or `{ domain: string }`).
 * @param b - Second domain name.
 * @returns Negative if `a < b`, positive if `a > b`, zero if equal.
 */
export function fqdnCompare(
    a: string | Domain | { domain: string },
    b: string | Domain | { domain: string },
) {
    // Convert to string if Domain
    let domainA = typeof a === "object" ? a.domain : a;
    let domainB = typeof b === "object" ? b.domain : b;

    const as = domainA.split(".").reverse();
    const bs = domainB.split(".").reverse();

    // Remove first item if empty
    if (!as[0].length) as.shift();
    if (!bs[0].length) bs.shift();

    const maxDepth = Math.min(as.length, bs.length);
    for (let i = Math.min(maxDepth, 1); i < maxDepth; i++) {
        const cmp = as[i].toLowerCase().localeCompare(bs[i].toLowerCase());
        if (cmp !== 0) {
            return cmp;
        } else if (i == 1) {
            const cmp = as[0].toLowerCase().localeCompare(bs[0].toLowerCase());
            if (cmp !== 0) {
                return cmp;
            }
        }
    }

    return as.length - bs.length;
}

/**
 * Converts a numeric DNS class value to its mnemonic string.
 *
 * Recognised values: 1 → `"IN"`, 3 → `"CH"`, 4 → `"HS"`, 254 → `"NONE"`.
 * Unknown values return `"##"`.
 *
 * @param input - Numeric DNS class as returned by the server.
 * @returns The class mnemonic string.
 */
export function nsclass(input: number): string {
    switch (input) {
        case 1:
            return "IN";
        case 3:
            return "CH";
        case 4:
            return "HS";
        case 254:
            return "NONE";
        default:
            return "##";
    }
}

/**
 * Formats a TTL value (in seconds) as a human-readable duration string.
 *
 * The result uses the largest applicable units in order: days (`d`), hours
 * (`h`), minutes (`m`), seconds (`s`). Zero-valued units are omitted.
 * Examples: `3661` → `"1h 1m 1s"`, `86400` → `"1d"`.
 *
 * @param input - TTL in seconds.
 * @returns Human-readable duration string.
 */
export function nsttl(input: number): string {
    let ret = "";

    if (input / 86400 >= 1) {
        ret += Math.floor(input / 86400) + "d ";
        input = input % 86400;
    }
    if (input / 3600 >= 1) {
        ret += Math.floor(input / 3600) + "h ";
        input = input % 3600;
    }
    if (input / 60 >= 1) {
        ret += Math.floor(input / 60) + "m ";
        input = input % 60;
    }
    if (input >= 1) {
        ret += Math.floor(input) + "s";
    }

    return ret.trim();
}

/**
 * Validates a domain name against DNS naming rules.
 *
 * - Returns `undefined` when `dn` is empty (no opinion).
 * - Returns `false` when `dn` does not belong to `origin` after FQDN resolution.
 * - Returns `true` / `false` based on RFC label rules: total length 1–254,
 *   each label 1–63 characters. By default labels may start with `_` (for
 *   service names); pass `only_ldh = true` to enforce strict LDH rules.
 * - A leading `*` label (wildcard) is accepted and skipped during validation.
 *
 * @param dn        - The domain name to validate (relative or absolute).
 * @param origin    - Zone origin used to resolve relative names (default `""`).
 * @param only_ldh  - When `true`, disallow leading underscores in labels.
 * @returns `true` if valid, `false` if invalid, `undefined` if empty input.
 */
export function validateDomain(
    dn: string,
    origin: string = "",
    only_ldh: boolean = false,
): boolean | undefined {
    let ret: boolean | undefined = undefined;
    if (dn.length !== 0) {
        dn = fqdn(dn, origin);
        if (!dn.endsWith(origin)) {
            return false;
        }

        ret = dn.length >= 1 && dn.length <= 254;

        if (ret) {
            const domains = dn.split(".");

            // Remove the last . if any, it's ok
            if (domains[domains.length - 1] === "") {
                domains.pop();
            }

            // Remove the first * if any, it's a valid wildcard domain
            if (domains[0] === "*") {
                domains.shift();
            }

            ret = domains.reduce(
                (acc, domain) =>
                    acc &&
                    domain.length >= 1 &&
                    domain.length <= 63 &&
                    ((only_ldh && /^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$/.test(domain)) ||
                        (!only_ldh && /^_?[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$/.test(domain))),
                ret as boolean,
            );
        }
    }

    return ret;
}

/**
 * Returns `true` when the given FQDN is a reverse DNS zone.
 *
 * Checks for both IPv4 (`*.in-addr.arpa.`) and IPv6 (`*.ip6.arpa.`) suffixes.
 *
 * @param fqdn - Fully qualified domain name to test.
 */
export function isReverseZone(fqdn: string) {
    return fqdn.endsWith("in-addr.arpa.") || fqdn.endsWith("ip6.arpa.");
}

/**
 * Expands a possibly-abbreviated IPv6 address into its full 8-group form.
 *
 * Handles the `::` zero-compression notation. Each group is zero-padded to
 * exactly 4 hex digits. Returns `null` if the address cannot be parsed or
 * does not yield exactly 8 groups.
 *
 * @param addr - IPv6 address string (may contain `::` abbreviation).
 * @returns Full 8-group colon-separated hex string, or `null` on error.
 */
function normalizeIPv6(addr: string): string | null {
    try {
        const parts = addr.split("::");
        let head = parts[0].split(":");
        let tail = parts[1] ? parts[1].split(":") : [];

        // Fill the "::" gap with zeroes
        const fullParts = [...head, ...Array(8 - head.length - tail.length).fill("0"), ...tail];

        // Ensure 8 segments
        if (fullParts.length !== 8) return null;

        return fullParts
            .map((part) => part.padStart(4, "0")) // ensure 4 digits
            .join(":");
    } catch {
        return null;
    }
}

/**
 * Converts an IP address to its reverse DNS domain name (PTR record owner).
 *
 * - IPv4: octets are reversed and `.in-addr.arpa.` is appended.
 *   Partial addresses are zero-padded to 4 octets before reversal.
 * - IPv6: the address is normalised to full form, colons removed, digits
 *   reversed individually, and `.ip6.arpa.` is appended.
 *
 * @param ip - IPv4 or IPv6 address string.
 * @returns The corresponding reverse-zone domain name.
 * @throws {Error} If the IPv6 address is invalid.
 */
export function reverseDomain(ip: string) {
    let suffix = ".in-addr.arpa.";

    let fields: Array<string>;
    if (ip.indexOf(":") > 0) {
        suffix = ".ip6.arpa.";

        const normalized = normalizeIPv6(ip);
        if (!normalized) throw new Error("Invalid IPv6 address");
        fields = normalized.replace(/:/g, "").split("");
    } else {
        fields = ip.split(".");
        while (fields.length < 4) {
            const last = fields.pop()!;
            fields.push("0", last);
        }
    }

    return fields.reverse().join(".") + suffix;
}

/**
 * Converts a reverse DNS domain name back to a human-readable IP address.
 *
 * - For `*.in-addr.arpa.` names the dot-separated nibbles are reversed to
 *   reconstruct the IPv4 address.
 * - For `*.ip6.arpa.` names the nibbles are grouped by 4, reversed, and
 *   joined with colons; the result is then abbreviated using standard IPv6
 *   `::` compression and leading-zero stripping.
 *
 * @param dn - Reverse DNS domain name (must end with `in-addr.arpa.` or `ip6.arpa.`).
 * @returns The corresponding IP address string.
 */
export function unreverseDomain(dn: string) {
    let split_char = ".";
    let group = 1;

    if (dn.endsWith("ip6.arpa.")) {
        split_char = ":";
        group = 4;
        dn = dn.substring(0, dn.indexOf(".ip6.arpa."));
    } else {
        dn = dn.substring(0, dn.indexOf(".in-addr.arpa."));
    }

    const fields = dn.split(".");
    let ip = fields.reduce((a, v, i) => v + (i % group == 0 ? split_char : "") + a, "");
    ip = ip.substring(0, ip.length - 1);
    return ip
        .replace(/:(0000:)+/, "::")
        .replace(/:0{1,3}/g, ":")
        .replace(/^0+/, "")
        .replace(/0+$/, "");
}

/**
 * Formats a DNS resource record as a zone-file line.
 *
 * The owner name is resolved via {@link fqdn} using the optional `dn` and
 * `origin` context. TTL and class fields are included only when non-zero.
 * Output format: `<owner> [<ttl>] [<class>] <type> <rdata>`
 *
 * @param rr     - The DNS resource record to format.
 * @param dn     - Optional subdomain context for resolving the owner name.
 * @param origin - Optional zone origin for resolving the owner name.
 * @returns Zone-file representation of the record.
 */
export function printRR(rr: dnsRR, dn?: string, origin?: string): string {
    let domain = rr.Hdr.Name || "@";
    if (dn && origin) domain = fqdn(domain, fqdn(dn, origin));
    else if (dn) domain = fqdn(domain, dn);
    else if (origin) domain = fqdn(domain, origin);

    return (
        domain +
        (rr.Hdr.Ttl ? "\t" + rr.Hdr.Ttl : "") +
        (rr.Hdr.Class ? "\t" + nsclass(rr.Hdr.Class) : "") +
        "\t" +
        nsrrtype(rr.Hdr.Rrtype) +
        "\t" +
        rdatatostr(rr)
    );
}

/**
 * Parses a semicolon-delimited `key=value` TXT record string into an object.
 *
 * Surrounding double-quotes are stripped before parsing. Each pair is split on
 * the first `=` only, so values may contain additional `=` characters. Pairs
 * missing a key or a value are silently ignored.
 *
 * @param input - Raw TXT record string (e.g. `"v=spf1; a; ~all"`).
 * @returns Object mapping trimmed keys to trimmed values.
 */
export function parseKeyValueTxt(input: string): Record<string, string> {
    // Remove surrounding quotes if present
    const trimmed = input.trim().replace(/^"|"$/g, "");

    // Split the string by semicolons to separate key-value pairs
    const pairs = trimmed.split(";");

    const result: Record<string, string> = {};

    for (const pair of pairs) {
        // Trim whitespace around the pair
        const cleaned = pair.trim();
        if (!cleaned) continue;

        // Split by the first '=' only, in case values contain '='
        const [key, ...rest] = cleaned.split("=");
        const value = rest.join("=");

        if (key && value) {
            result[key.trim()] = value.trim();
        }
    }

    return result;
}

/**
 * Creates a new, blank DNS resource record with an empty name and type 0.
 *
 * Useful as a default/placeholder value when a `dnsRR` object is required
 * before real data is available.
 *
 * @returns A zeroed-out `dnsRR` instance.
 */
export function emptyRR(): dnsRR {
    return newRR("", 0);
}

/**
 * Recursively traverses a service field value according to its type descriptor
 * and appends any leaf DNS resource records found to the provided array.
 *
 * Handles array types (prefixed with "[]"), pointer types (prefixed with "*"),
 * and leaf DNS types ("dns.*", "happydns.Record", "happydns.TXT", "happydns.SPF").
 * Non-DNS composite types (sub-services) are not traversed.
 *
 * @param type  - The field type string as defined in the service spec.
 * @param value - The corresponding runtime value from the service instance.
 * @param rrs   - Accumulator array that receives the collected dnsRR objects.
 */
function collectFieldRRs(type: string, value: any, rrs: dnsRR[]) {
    if (value === null || value === undefined) return;
    if (type.startsWith("[]")) {
        if (Array.isArray(value)) {
            for (const item of value) {
                collectFieldRRs(type.substring(2), item, rrs);
            }
        }
    } else if (type.startsWith("*")) {
        collectFieldRRs(type.substring(1), value, rrs);
    } else if (
        type.startsWith("dns.") ||
        type === "happydns.Record" ||
        type === "happydns.TXT" ||
        type === "happydns.SPF"
    ) {
        rrs.push(value);
    }
}

/**
 * Extracts all DNS resource records from a service instance given its spec fields.
 *
 * Iterates over the top-level fields of a service spec and delegates to
 * {@link collectFieldRRs} for each field, accumulating every leaf dnsRR found
 * in the service value.
 *
 * @param fields - The array of field descriptors from a ServiceSpec, or null.
 * @param value  - The service instance value (ServiceCombined.Service).
 * @returns      An array of dnsRR objects contained in the service.
 */
export function collectRRs(fields: Array<{ type: string; id: string }> | null, value: any): dnsRR[] {
    if (!fields || !value) return [];
    const rrs: dnsRR[] = [];
    for (const field of fields) {
        collectFieldRRs(field.type, value[field.id], rrs);
    }
    return rrs;
}

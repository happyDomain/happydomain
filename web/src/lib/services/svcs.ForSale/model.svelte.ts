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

// RFC 10023 "For Sale" TXT records: each record at the _for-sale leaf node
// starts with the version tag and carries at most one tag-value pair.

import { getRrtype, newRR, type dnsTypeTXT } from "$lib/dns_rr";
import type { SvcsForSaleBody } from "$lib/services_bodies";

/**
 * The body as the editor may receive it: the API always sends the whole RRset,
 * but a lone record is tolerated, as older zones may hold one.
 */
export type ForSaleValue = { txt?: SvcsForSaleBody["txt"] | dnsTypeTXT };

export const FORSALE_LABEL = "_for-sale";
export const FORSALE_VERSION = "v=FORSALE1;";

/** Maximum length, in octets, of a fcod or ftxt value. */
export const FORSALE_MAX_VALUE_LEN = 239;

export const FORSALE_TAGS = ["fval", "ftxt", "furi", "fcod"] as const;
export type ForSaleTag = (typeof FORSALE_TAGS)[number];

export interface ForSalePair {
    /** One of the RFC 10023 tags, or null when the record only holds the version tag. */
    tag: string | null;
    value: string;
    /** True when the record does not start with the mandatory version tag. */
    invalidVersion: boolean;
    /** True when the content is not a `tag=value` pair. */
    malformed: boolean;
}

/** Length of a string in UTF-8 octets, which is what the RFC limits. */
export function byteLength(value: string): number {
    return new TextEncoder().encode(value).length;
}

export function isForSaleTag(tag: string | null): tag is ForSaleTag {
    return tag !== null && (FORSALE_TAGS as readonly string[]).includes(tag);
}

/**
 * Split a for-sale record into its tag and value. A single space following the
 * version tag is tolerated, as RFC 10023 section 3.6 allows.
 */
export function parseForSale(txt: string): ForSalePair {
    const trimmed = txt.trim();

    if (!trimmed.startsWith(FORSALE_VERSION)) {
        return { tag: null, value: trimmed, invalidVersion: true, malformed: false };
    }

    let content = trimmed.slice(FORSALE_VERSION.length);
    if (content.startsWith(" ")) content = content.slice(1);

    if (content === "") {
        return { tag: null, value: "", invalidVersion: false, malformed: false };
    }

    const eq = content.indexOf("=");
    if (eq < 0) {
        return { tag: null, value: content, invalidVersion: false, malformed: true };
    }

    return {
        tag: content.slice(0, eq),
        value: content.slice(eq + 1),
        invalidVersion: false,
        malformed: false,
    };
}

export function stringifyForSale(tag: string | null, value: string): string {
    if (!tag) return FORSALE_VERSION;

    return FORSALE_VERSION + tag + "=" + value;
}

export interface ForSalePrice {
    currency: string;
    amount: string;
}

/** Split a fval value, eg. "USD750.50", into its currency and amount parts. */
export function parsePrice(value: string): ForSalePrice {
    const m = /^([A-Z]*)(.*)$/.exec(value);

    return { currency: m ? m[1] : "", amount: m ? m[2] : value };
}

export function stringifyPrice(currency: string, amount: string): string {
    return currency + amount;
}

export function isValidCurrency(currency: string): boolean {
    return /^[A-Z]+$/.test(currency);
}

export function isValidAmount(amount: string): boolean {
    return /^[0-9]+(\.[0-9]+)?$/.test(amount);
}

/** Build a new TXT record for the _for-sale node. */
export function newForSaleRecord(tag: string | null, value: string): dnsTypeTXT {
    // The header name stays the bare relative label: the backend joins it with
    // the service subdomain when generating the zone.
    const rr = newRR(FORSALE_LABEL, getRrtype("TXT")) as dnsTypeTXT;
    rr.Txt = stringifyForSale(tag, value);
    return rr;
}

/** One editable entry, keeping a link back to its index in the RRset. */
export interface ForSaleEntry {
    index: number;
    pair: ForSalePair;
}

/**
 * Editable view over the whole _for-sale RRset. The records array is the
 * source of truth: the editor mutates it directly, exactly like CAAPolicy.
 */
export class ForSaleService {
    records = $state<Array<dnsTypeTXT>>([]);

    constructor(value: ForSaleValue) {
        const txt = value.txt;

        if (txt) {
            this.records = Array.isArray(txt) ? txt : [txt];
        } else {
            this.records = [];
        }
    }

    /** Every entry of the RRset, in order. */
    get entries(): ForSaleEntry[] {
        return this.records.map((rr, index) => ({ index, pair: parseForSale(rr.Txt) }));
    }

    /**
     * Entries the editor has something to show: the tag-value pairs, plus the
     * records whose content is broken, so the user can repair them.
     */
    get editableEntries(): ForSaleEntry[] {
        return this.entries.filter((e) => e.pair.tag !== null || e.pair.malformed);
    }

    getValue(index: number): string {
        return parseForSale(this.records[index].Txt).value;
    }

    setValue(index: number, value: string): void {
        const pair = parseForSale(this.records[index].Txt);
        this.records[index].Txt = stringifyForSale(pair.tag, value);
    }

    /** Rewrite everything that follows the version tag, broken content included. */
    setRaw(index: number, content: string): void {
        this.records[index].Txt = content ? FORSALE_VERSION + content : FORSALE_VERSION;
    }

    add(tag: ForSaleTag, value: string = ""): void {
        // A bare version record is a placeholder: replace it rather than
        // publishing a redundant pair alongside the first real one.
        const bare = this.entries.find((e) => e.pair.tag === null && !e.pair.malformed);
        if (bare) {
            this.records.splice(bare.index, 1);
        }

        this.records.push(newForSaleRecord(tag, value));
    }

    remove(index: number): void {
        this.records.splice(index, 1);

        // The service must always publish at least one record, otherwise the
        // _for-sale node disappears entirely.
        if (this.records.length === 0) {
            this.records.push(newForSaleRecord(null, ""));
        }
    }
}

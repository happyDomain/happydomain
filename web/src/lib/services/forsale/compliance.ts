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

import {
    asArray,
    type ComplianceContext,
    type ComplianceIssue,
    registerValidators,
} from "$lib/services/compliance";
import {
    byteLength,
    FORSALE_LABEL,
    FORSALE_MAX_VALUE_LEN,
    FORSALE_VERSION,
    isForSaleTag,
    isValidAmount,
    isValidCurrency,
    parseForSale,
    parsePrice,
} from "$lib/services/forsale.svelte";

const RFC = "https://www.rfc-editor.org/rfc/rfc10023";

/** TTL of 3600 seconds or less is RECOMMENDED (RFC 10023 sec. 3.4). */
const MAX_TTL = 3600;

const RECOMMENDED_URI_SCHEMES = ["http:", "https:", "mailto:", "tel:"];

interface rawTXT {
    Hdr?: { Name?: string; Ttl?: number };
    Txt?: string;
}

function ownerNameIsForSale(name: string): boolean {
    return name === FORSALE_LABEL || name.startsWith(FORSALE_LABEL + ".");
}

function forsaleSync(raw: Record<string, any>, _ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const records = asArray<rawTXT>(raw?.txt);

    if (records.length === 0) return issues;

    const seenPairs = new Set<string>();
    const ttls = new Set<number>();
    let hasContent = false;

    for (const record of records) {
        const txt = typeof record?.Txt === "string" ? record.Txt : "";
        const name = typeof record?.Hdr?.Name === "string" ? record.Hdr.Name : "";
        const ttl = typeof record?.Hdr?.Ttl === "number" ? record.Hdr.Ttl : 0;

        // Sec. 2.6: the records live at a _for-sale leaf node.
        if (name && !ownerNameIsForSale(name)) {
            issues.push({
                id: "forsale.wrong-owner-name",
                severity: "error",
                params: { name },
                docUrl: RFC + "#section-2.6",
            });
        }

        if (ttl) {
            ttls.add(ttl);
            if (ttl > MAX_TTL) {
                issues.push({
                    id: "forsale.ttl-too-high",
                    severity: "warning",
                    params: { ttl },
                    docUrl: RFC + "#section-3.4",
                });
            }
        }

        // Sec. 2.1: every record MUST begin with the version tag.
        if (!txt.trim().startsWith(FORSALE_VERSION)) {
            issues.push({
                id: "forsale.missing-version",
                severity: "error",
                params: { txt },
                docUrl: RFC + "#section-2.1",
            });
            continue;
        }

        const pair = parseForSale(txt);

        if (pair.malformed) {
            issues.push({
                id: "forsale.malformed-content",
                severity: "error",
                params: { content: pair.value },
                docUrl: RFC + "#section-2.1",
            });
            continue;
        }

        if (pair.tag === null) continue;

        hasContent = true;

        // Sec. 2.1: at most one tag-value pair per record.
        if (pair.value.includes(";")) {
            issues.push({
                id: "forsale.multiple-pairs",
                severity: "error",
                params: { tag: pair.tag },
                docUrl: RFC + "#section-2.1",
            });
        }

        // Sec. 2.4: every tag-value pair in the RRset MUST be unique.
        const key = pair.tag + "=" + pair.value;
        if (seenPairs.has(key)) {
            issues.push({
                id: "forsale.duplicate-pair",
                severity: "error",
                params: { pair: key },
                docUrl: RFC + "#section-2.4",
            });
        }
        seenPairs.add(key);

        if (!isForSaleTag(pair.tag)) {
            issues.push({
                id: "forsale.unknown-tag",
                severity: "warning",
                params: { tag: pair.tag },
                docUrl: RFC + "#section-2.2.5",
            });
            continue;
        }

        if (pair.value === "") {
            issues.push({
                id: "forsale.empty-value",
                severity: "error",
                field: pair.tag,
                params: { tag: pair.tag },
                docUrl: RFC + "#section-2.1",
            });
            continue;
        }

        // Sec. 2.3: fcod and ftxt values are limited to 239 octets.
        if (pair.tag === "fcod" || pair.tag === "ftxt") {
            const len = byteLength(pair.value);
            if (len > FORSALE_MAX_VALUE_LEN) {
                issues.push({
                    id: "forsale.value-too-long",
                    severity: "error",
                    field: pair.tag,
                    params: { tag: pair.tag, length: len, max: FORSALE_MAX_VALUE_LEN },
                    docUrl: RFC + "#section-2.3",
                });
            }
        }

        if (pair.tag === "fval") {
            const { currency, amount } = parsePrice(pair.value);
            if (!isValidCurrency(currency) || !isValidAmount(amount)) {
                issues.push({
                    id: "forsale.invalid-price",
                    severity: "error",
                    field: "fval",
                    params: { value: pair.value },
                    docUrl: RFC + "#section-2.2.4",
                });
            }
        }

        if (pair.tag === "furi") {
            let scheme: string | null = null;
            try {
                scheme = new URL(pair.value).protocol;
            } catch {
                issues.push({
                    id: "forsale.invalid-uri",
                    severity: "error",
                    field: "furi",
                    params: { uri: pair.value },
                    docUrl: RFC + "#section-2.2.3",
                });
            }

            if (scheme && !RECOMMENDED_URI_SCHEMES.includes(scheme)) {
                issues.push({
                    id: "forsale.unusual-uri-scheme",
                    severity: "warning",
                    field: "furi",
                    params: { scheme: scheme.replace(/:$/, ""), uri: pair.value },
                    docUrl: RFC + "#section-2.2.3",
                });
            }
        }
    }

    // Sec. 3.4: all records of the RRset should share the same TTL.
    if (ttls.size > 1) {
        issues.push({
            id: "forsale.inconsistent-ttl",
            severity: "warning",
            params: { ttls: [...ttls].sort((a, b) => a - b).join(", ") },
            docUrl: RFC + "#section-3.4",
        });
    }

    if (!hasContent) {
        issues.push({
            id: "forsale.no-content",
            severity: "info",
            docUrl: RFC + "#section-2.1",
        });
    }

    return issues;
}

registerValidators("svcs.ForSale", { sync: forsaleSync });

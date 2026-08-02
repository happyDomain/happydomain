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

import { flattenSPF } from "$lib/api/resolver";
import { fqdn } from "$lib/dns";
import {
    registerValidators,
    type ComplianceContext,
    type ComplianceIssue,
} from "$lib/services/compliance";
import {
    KNOWN_MECHANISMS,
    KNOWN_MODIFIERS,
    countLocalLookups,
    parseSPF,
    parseTerm,
    stringifySPF,
    type SPFValue,
} from "./model";

const SPF_LOOKUP_WARN_THRESHOLD = 8;
const SPF_LOOKUP_MAX = 10;
const SPF_TXT_LENGTH_WARN = 255;

const SPF_RFC_URL = "https://datatracker.ietf.org/doc/html/rfc7208";

export function validateSPF(val: SPFValue, _ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    // 1. Version
    if (!val.v) {
        issues.push({
            id: "spf.missing-version",
            severity: "error",
            docUrl: SPF_RFC_URL + "#section-4.5",
        });
        return issues;
    }
    if (val.v.toLowerCase() !== "spf1") {
        issues.push({
            id: "spf.wrong-version",
            severity: "error",
            params: { version: val.v },
            docUrl: SPF_RFC_URL + "#section-4.5",
        });
        return issues;
    }

    // 2. Walk terms
    const terms = val.f.map((raw, index) => ({ index, term: parseTerm(raw) }));
    const allTerms = terms.filter((t) => t.term.isAll);
    const redirectTerms = terms.filter((t) => t.term.isModifier && t.term.name === "redirect");
    const ptrTerms = terms.filter((t) => !t.term.isModifier && t.term.name === "ptr");

    // 3. all mechanism rules
    if (allTerms.length === 0 && redirectTerms.length === 0) {
        issues.push({
            id: "spf.no-all-mechanism",
            severity: "warning",
            docUrl: SPF_RFC_URL + "#section-5.1",
        });
    }
    if (allTerms.length > 1) {
        issues.push({
            id: "spf.multiple-all",
            severity: "error",
            params: { count: allTerms.length },
            field: `f[${allTerms[1].index}]`,
            docUrl: SPF_RFC_URL + "#section-5.1",
        });
    }
    if (allTerms.length === 1) {
        const allIdx = allTerms[0].index;
        if (allIdx !== val.f.length - 1) {
            issues.push({
                id: "spf.all-not-last",
                severity: "warning",
                field: `f[${allIdx}]`,
                docUrl: SPF_RFC_URL + "#section-5.1",
            });
        }
    }
    if (allTerms.length > 0 && redirectTerms.length > 0) {
        issues.push({
            id: "spf.redirect-with-all",
            severity: "warning",
            field: `f[${redirectTerms[0].index}]`,
            docUrl: SPF_RFC_URL + "#section-6.1",
        });
    }
    if (redirectTerms.length > 1) {
        issues.push({
            id: "spf.multiple-redirect",
            severity: "error",
            field: `f[${redirectTerms[1].index}]`,
            docUrl: SPF_RFC_URL + "#section-6.1",
        });
    }

    // 4. ptr is deprecated
    if (ptrTerms.length > 0) {
        issues.push({
            id: "spf.ptr-deprecated",
            severity: "warning",
            field: `f[${ptrTerms[0].index}]`,
            docUrl: SPF_RFC_URL + "#section-5.5",
        });
    }

    // 5. Lookup budget: handled authoritatively by the async recursive walk
    // (validateSPFRecursive). Emitting a local warning here would duplicate
    // its result.

    // 6. Per-term checks: empty terms, unknown names, missing values, duplicates.
    const seen = new Set<string>();
    terms.forEach(({ index, term }) => {
        if (term.raw.trim() === "") {
            issues.push({
                id: "spf.empty-term",
                severity: "warning",
                field: `f[${index}]`,
            });
        } else if (!term.isModifier && !KNOWN_MECHANISMS.has(term.name)) {
            issues.push({
                id: "spf.unknown-mechanism",
                severity: "error",
                params: { mechanism: term.raw },
                field: `f[${index}]`,
                docUrl: SPF_RFC_URL + "#section-5",
            });
        } else if (term.isModifier && !KNOWN_MODIFIERS.has(term.name)) {
            issues.push({
                id: "spf.unknown-modifier",
                severity: "warning",
                params: { modifier: term.name },
                field: `f[${index}]`,
                docUrl: SPF_RFC_URL + "#section-6",
            });
        } else if (term.consumesLookup && !term.value && term.name !== "a" && term.name !== "mx") {
            // include / exists / redirect / ptr require a domain. Bare "a" and
            // "mx" mean "the current zone" so they are valid without value.
            issues.push({
                id: "spf.mechanism-missing-value",
                severity: "error",
                params: { mechanism: term.name },
                field: `f[${index}]`,
            });
        }

        const key = term.raw.toLowerCase();
        if (seen.has(key)) {
            issues.push({
                id: "spf.duplicate-mechanism",
                severity: "info",
                params: { mechanism: term.raw },
                field: `f[${index}]`,
            });
        } else {
            seen.add(key);
        }
    });

    // 8. Length
    const fullRecord = stringifySPF(val);
    if (fullRecord.length > SPF_TXT_LENGTH_WARN) {
        issues.push({
            id: "spf.length-exceeds-txt-string",
            severity: "info",
            params: { length: fullRecord.length, max: SPF_TXT_LENGTH_WARN },
            docUrl: SPF_RFC_URL + "#section-3.3",
        });
    }

    return issues;
}

export async function validateSPFRecursive(
    val: SPFValue,
    ctx: ComplianceContext,
    signal: AbortSignal,
): Promise<ComplianceIssue[]> {
    if (!val.v || val.v.toLowerCase() !== "spf1") return [];

    const localBudget = countLocalLookups(val);
    if (localBudget.count === 0) return [];

    const domain = fqdn(ctx.dn || "@", ctx.origin?.domain ?? "");
    if (!domain) return [];

    const record = stringifySPF(val);
    const resp = await flattenSPF({ domain, record }, signal);

    const issues: ComplianceIssue[] = [];
    const total = resp.lookupCount ?? 0;

    if (resp.exceeded) {
        issues.push({
            id: "spf.recursive-too-many-lookups",
            severity: "error",
            params: { count: total, max: SPF_LOOKUP_MAX },
            docUrl: SPF_RFC_URL + "#section-4.6.4",
        });
    } else if (total >= SPF_LOOKUP_WARN_THRESHOLD) {
        issues.push({
            id: "spf.recursive-many-lookups",
            severity: "warning",
            params: { count: total, max: SPF_LOOKUP_MAX },
            docUrl: SPF_RFC_URL + "#section-4.6.4",
        });
    }

    if (resp.voidExceeded) {
        issues.push({
            id: "spf.too-many-void-lookups",
            severity: "warning",
            params: { count: resp.voidLookups ?? 0, max: 2 },
            docUrl: SPF_RFC_URL + "#section-4.6.4",
        });
    }

    // Surface unreachable / loop / no-spf children as individual issues so the
    // user can see exactly which include misbehaves. Budget/depth overruns are
    // already reported as a top-level issue, so we skip them here.
    const errorToId: Record<string, string> = {
        loop: "spf.include-loop",
        "no-spf": "spf.include-no-spf",
        nxdomain: "spf.include-no-spf",
        timeout: "spf.include-resolver-error",
        resolver: "spf.include-resolver-error",
    };
    const walk = (node: { domain?: string; mechanism?: string; error?: string; children?: any[] } | undefined) => {
        if (!node) return;
        const err = node.error;
        if (err && err !== "budget" && err !== "depth") {
            const id = errorToId[err] ?? "spf.include-error";
            issues.push({
                id,
                severity: id === "spf.include-resolver-error" ? "info" : "warning",
                params: { domain: node.domain ?? "", mechanism: node.mechanism ?? "" },
            });
        }
        for (const c of node.children ?? []) walk(c);
    };
    walk(resp.tree as any);

    return issues;
}

registerValidators("svcs.SPF", {
    sync: (raw, ctx) => validateSPF(parseSPF(raw?.txt?.Txt ?? ""), ctx),
    async: (raw, ctx, signal) =>
        validateSPFRecursive(parseSPF(raw?.txt?.Txt ?? ""), ctx, signal),
});

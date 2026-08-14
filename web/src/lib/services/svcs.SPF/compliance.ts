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
    isValidDnsName,
    normalizeFqdn,
    registerValidators,
    type ComplianceContext,
    type ComplianceIssue,
} from "$lib/services/compliance";
import {
    KNOWN_MECHANISMS,
    KNOWN_MODIFIERS,
    countLocalLookups,
    isIPv4,
    isIPv6,
    parseSPF,
    parseTerm,
    stringifySPF,
    type ParsedTerm,
    type SPFValue,
} from "./model";

const SPF_LOOKUP_WARN_THRESHOLD = 8;
const SPF_LOOKUP_MAX = 10;
const SPF_TXT_LENGTH_WARN = 255;

const SPF_RFC_URL = "https://datatracker.ietf.org/doc/html/rfc7208";

const CIDR_LENGTH_RE = /^(0|[1-9][0-9]{0,2})$/;

/**
 * An ip4/ip6 mechanism is the only one a receiver answers from the record
 * alone: a typo there silently drops every message from the addresses it was
 * meant to cover, without a DNS error anywhere to point at it.
 */
function validateIpMechanism(term: ParsedTerm, index: number): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const field = `f[${index}]`;
    const docUrl = SPF_RFC_URL + "#section-5.6";
    const v6 = term.name === "ip6";

    const value = (term.value ?? "").trim();
    if (value === "") {
        issues.push({
            id: "spf.ip-missing-value",
            severity: "error",
            params: { mechanism: term.name },
            field,
            docUrl,
        });
        return issues;
    }

    const slash = value.indexOf("/");
    const address = slash === -1 ? value : value.slice(0, slash);
    const prefix = slash === -1 ? undefined : value.slice(slash + 1);

    if (!(v6 ? isIPv6(address) : isIPv4(address))) {
        // Telling apart "not an address" from "an address of the other family"
        // matters: the second is a copy/paste away from being correct.
        const swapped = v6 ? isIPv4(address) : isIPv6(address);
        issues.push({
            id: swapped ? "spf.ip-family-mismatch" : "spf.invalid-ip",
            severity: "error",
            params: {
                mechanism: term.name,
                address,
                expected: v6 ? "IPv6" : "IPv4",
                found: v6 ? "IPv4" : "IPv6",
            },
            field,
            docUrl,
        });
    }

    if (prefix !== undefined) {
        const max = v6 ? 128 : 32;
        if (!CIDR_LENGTH_RE.test(prefix) || Number(prefix) > max) {
            issues.push({
                id: "spf.invalid-cidr",
                severity: "error",
                params: { mechanism: term.name, prefix, max },
                field,
                docUrl,
            });
        }
    }

    return issues;
}

// Mechanisms whose value is a domain the receiver will query.
const DOMAIN_MECHANISMS = new Set<string>(["include", "exists", "a", "mx", "ptr"]);

/**
 * The domain an include, a redirect or any other domain-carrying term points
 * at. A name that cannot be resolved is not a soft failure here: RFC 7208
 * sec. 7.1 asks verifiers to answer permerror, which voids the whole policy.
 */
function validateDomainTarget(mechanism: string, target: string, index: number): ComplianceIssue[] {
    const field = `f[${index}]`;
    const docUrl = SPF_RFC_URL + "#section-7.1";

    // What a macro expands to is only known to the verifier, so it stands in as
    // an ordinary label character: everything written around it still has to
    // read as a name. Macro syntax itself is checked apart.
    const hasMacro = target.includes("%");
    const domain = normalizeFqdn(target).replace(/%\{[^}]*\}|%[%_-]/g, "m");
    const labels = domain.split(".");

    // A top label made only of digits is excluded by the grammar, where it
    // would be indistinguishable from the address literal of an ip4 mechanism.
    if (!isValidDnsName(domain) || (!hasMacro && /^[0-9]+$/.test(labels[labels.length - 1]))) {
        return [
            {
                id: "spf.invalid-target",
                severity: "error",
                params: { mechanism, domain: target },
                field,
                docUrl,
            },
        ];
    }

    // A lone %{d} already expands to a complete name.
    if (labels.length < 2 && !hasMacro) {
        return [
            {
                id: "spf.target-not-fqdn",
                severity: "warning",
                params: { mechanism, domain: target },
                field,
                docUrl,
            },
        ];
    }

    return [];
}

// "/24", "//64" or "/24//64", the two prefix lengths an a or mx mechanism
// applies to the addresses it resolves.
const DUAL_CIDR_RE = /^(?:\/([0-9]+))?(?:\/\/([0-9]+))?$/;

/**
 * The prefix lengths of an a or mx mechanism. Left unchecked, a stray slash
 * widens or voids the set of hosts the mechanism authorizes, without any of
 * the DNS lookups it performs failing.
 */
function validateDualCidr(term: ParsedTerm, index: number): ComplianceIssue[] {
    const value = term.value ?? "";
    const field = `f[${index}]`;
    const docUrl = SPF_RFC_URL + "#section-5.3";

    // A macro body may contain a slash as a delimiter, so only what follows the
    // last one closing can be a prefix length.
    const slash = value.indexOf("/", value.lastIndexOf("}") + 1);
    if (slash === -1) return [];

    if (term.name !== "a" && term.name !== "mx") {
        return [
            {
                id: "spf.dual-cidr-not-allowed",
                severity: "error",
                params: { mechanism: term.name },
                field,
                docUrl,
            },
        ];
    }

    const cidr = value.slice(slash);
    const lengths = DUAL_CIDR_RE.exec(cidr);
    if (!lengths || cidr === "/") {
        return [
            {
                id: "spf.invalid-dual-cidr",
                severity: "error",
                params: { mechanism: term.name, cidr },
                field,
                docUrl,
            },
        ];
    }

    const issues: ComplianceIssue[] = [];
    for (const [prefix, max] of [
        [lengths[1], 32],
        [lengths[2], 128],
    ] as [string | undefined, number][]) {
        if (prefix === undefined) continue;
        if (!CIDR_LENGTH_RE.test(prefix) || Number(prefix) > max) {
            issues.push({
                id: "spf.invalid-cidr",
                severity: "error",
                params: { mechanism: term.name, prefix, max },
                field,
                docUrl,
            });
        }
    }

    return issues;
}

// Terms whose value is a domain-spec, the only place a macro may appear.
const MACRO_TERMS = new Set<string>([...DOMAIN_MECHANISMS, "redirect", "exp"]);

// %{ letter transformers delimiters }, per RFC 7208 sec. 7.1.
const MACRO_BODY_RE = /^([A-Za-z])([0-9]*)([rR]?)([-.+,/_=]*)$/;

const MACRO_LETTERS = "slodiphcrtv";
// c, r and t describe the connection being explained, so they only make sense
// in the explanation text an exp= modifier points at, never in a domain-spec.
const EXPLANATION_ONLY_LETTERS = "crt";

/**
 * Macros let a record describe the sender it is being checked against
 * (%{i}._spf.example.com). They are expanded by the verifier, which means a
 * malformed one is only ever seen by the remote side, as a permerror.
 */
function validateMacroString(mechanism: string, value: string, index: number): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const field = `f[${index}]`;
    const docUrl = SPF_RFC_URL + "#section-7.1";
    const invalid = (macro: string) =>
        issues.push({
            id: "spf.invalid-macro",
            severity: "error",
            params: { mechanism, macro },
            field,
            docUrl,
        });

    for (let i = 0; i < value.length; i++) {
        if (value[i] !== "%") continue;

        const next = value[i + 1];
        // %% %_ and %- stand for a literal percent, a space and a URL-escaped
        // space; everything else must open a macro.
        if (next === "%" || next === "_" || next === "-") {
            i++;
            continue;
        }
        if (next !== "{") {
            invalid(value.slice(i, i + 2));
            continue;
        }

        const close = value.indexOf("}", i);
        if (close === -1) {
            invalid(value.slice(i));
            break;
        }

        const macro = value.slice(i, close + 1);
        const body = MACRO_BODY_RE.exec(value.slice(i + 2, close));
        i = close;

        if (!body) {
            invalid(macro);
            continue;
        }

        // The optional digits say how many right-hand parts to keep, so zero
        // parts, or a count written with a leading zero, means nothing.
        if (body[2] !== "" && (Number(body[2]) === 0 || body[2][0] === "0")) {
            invalid(macro);
            continue;
        }

        const letter = body[1].toLowerCase();
        if (!MACRO_LETTERS.includes(letter)) {
            issues.push({
                id: "spf.unknown-macro-letter",
                severity: "error",
                params: { mechanism, macro, letter: body[1] },
                field,
                docUrl,
            });
        } else if (EXPLANATION_ONLY_LETTERS.includes(letter)) {
            issues.push({
                id: "spf.macro-explanation-only",
                severity: "error",
                params: { mechanism, macro, letter },
                field,
                docUrl,
            });
        } else if (letter === "p") {
            issues.push({
                id: "spf.macro-ptr-discouraged",
                severity: "warning",
                params: { mechanism, macro },
                field,
                docUrl: SPF_RFC_URL + "#section-7.3",
            });
        }
    }

    return issues;
}

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
    const expTerms = terms.filter((t) => t.term.isModifier && t.term.name === "exp");

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

    // 4. exp modifier rules
    if (expTerms.length > 1) {
        issues.push({
            id: "spf.multiple-exp",
            severity: "error",
            field: `f[${expTerms[1].index}]`,
            docUrl: SPF_RFC_URL + "#section-6.2",
        });
    }
    if (expTerms.length > 0 && !terms.some((t) => t.term.qualifier === "-")) {
        // An explanation is only ever published on a fail, so a record that
        // never fails carries one nobody will read.
        issues.push({
            id: "spf.exp-without-fail",
            severity: "info",
            field: `f[${expTerms[0].index}]`,
            docUrl: SPF_RFC_URL + "#section-6.2",
        });
    }

    // 5. ptr is deprecated
    if (ptrTerms.length > 0) {
        issues.push({
            id: "spf.ptr-deprecated",
            severity: "warning",
            field: `f[${ptrTerms[0].index}]`,
            docUrl: SPF_RFC_URL + "#section-5.5",
        });
    }

    // 6. Lookup budget: handled authoritatively by the async recursive walk
    // (validateSPFRecursive). Emitting a local warning here would duplicate
    // its result.

    // 7. Per-term checks: empty terms, unknown names, missing values, duplicates.
    const seen = new Set<string>();
    terms.forEach(({ index, term }) => {
        if (term.raw.trim() === "") {
            issues.push({
                id: "spf.empty-term",
                severity: "warning",
                field: `f[${index}]`,
            });
        } else if (term.badQualifier) {
            // Only "+", "-", "~" and "?" prefix a mechanism. Anything else is a
            // stray character that turns the whole record into a permerror.
            issues.push({
                id: "spf.invalid-qualifier",
                severity: "error",
                params: { qualifier: term.badQualifier, term: term.raw },
                field: `f[${index}]`,
                docUrl: SPF_RFC_URL + "#section-4.6.2",
            });
        } else if (term.qualifiedModifier && KNOWN_MODIFIERS.has(term.name)) {
            issues.push({
                id: "spf.qualifier-on-modifier",
                severity: "error",
                params: { qualifier: term.qualifier ?? "", modifier: term.name },
                field: `f[${index}]`,
                docUrl: SPF_RFC_URL + "#section-6",
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
        } else if (!term.isModifier && (term.name === "ip4" || term.name === "ip6")) {
            issues.push(...validateIpMechanism(term, index));
        } else if (!term.isModifier && DOMAIN_MECHANISMS.has(term.name) && term.value) {
            // a and mx carry their dual-CIDR length in the same value; only the
            // part before that slash is a name. A macro body may itself contain
            // a slash as a delimiter, so only what follows the last "}" closing
            // can be the prefix length.
            const slash = term.value.indexOf("/", term.value.lastIndexOf("}") + 1);
            const target = slash === -1 ? term.value : term.value.slice(0, slash);
            if (target !== "") issues.push(...validateDomainTarget(term.name, target, index));
            issues.push(...validateDualCidr(term, index));
        } else if (term.isModifier && term.name === "redirect" && term.value) {
            issues.push(...validateDomainTarget(term.name, term.value, index));
        } else if (term.isModifier && term.name === "exp") {
            // exp= names a TXT record holding the explanation sent back on a
            // fail. It is a domain, not the message itself.
            const target = (term.value ?? "").trim();
            if (target === "") {
                issues.push({
                    id: "spf.exp-missing-value",
                    severity: "error",
                    field: `f[${index}]`,
                    docUrl: SPF_RFC_URL + "#section-6.2",
                });
            } else {
                issues.push(...validateDomainTarget(term.name, target, index));
            }
        }

        // Macros live inside the value the branches above left as opaque, so
        // they are checked on their own rather than as part of the chain.
        if (term.value?.includes("%") && MACRO_TERMS.has(term.name)) {
            issues.push(...validateMacroString(term.name, term.value, index));
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
    const walk = (
        node: { domain?: string; mechanism?: string; error?: string; children?: any[] } | undefined,
    ) => {
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
    async: (raw, ctx, signal) => validateSPFRecursive(parseSPF(raw?.txt?.Txt ?? ""), ctx, signal),
});

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

/**
 * RFC 8461 rules shared by the two MTA-STS services: svcs.MTA_STS, which only
 * carries the DNS record, and abstract.MTASTS, which also has happyDomain
 * serve the policy file.
 *
 * Both report under the same `mta_sts.` issue ids, so the translations live
 * once, in svcs.MTA_STS/locales — every service's locales are merged into a
 * single payload.
 */

import type { ComplianceContext, ComplianceIssue } from "$lib/services/compliance";

export const RFC = "https://www.rfc-editor.org/rfc/rfc8461";

// RFC 8461 sec. 3.1: id is 1..32 alphanumeric characters.
export const ID_RE = /^[A-Za-z0-9]{1,32}$/;

export const VALID_MODES = new Set(["enforce", "testing", "none"]);

// RFC 8461 sec. 3.2: max_age is in [0, 31557600] (1 year).
export const MAX_AGE_HARD_LIMIT = 31557600;

// sec. 3.2 recommends "at least one week" but anything below a day is suspicious.
export const MAX_AGE_RECOMMENDED_MIN = 86400;

/**
 * RFC 8461 sec. 4.1: a "*." prefix matches exactly one DNS label; otherwise an
 * exact (case-insensitive) FQDN match is required.
 */
export function mtaStsPatternMatches(pattern: string, host: string): boolean {
    const p = pattern.toLowerCase().replace(/\.$/, "");
    const h = host.toLowerCase().replace(/\.$/, "");
    if (p.startsWith("*.")) {
        const suffix = p.slice(2);
        if (!suffix) return false;
        if (!h.endsWith("." + suffix)) return false;
        const head = h.slice(0, h.length - suffix.length - 1);
        return head.length > 0 && !head.includes(".");
    }
    return p === h;
}

/** Strips the trailing dot a FQDN may carry, as the policy file never has one. */
export function stripTrailingDot(name: string): string {
    return name.replace(/\.$/, "");
}

/** The URL the policy of the given domain is served from (RFC 8461 sec. 3.3). */
export function policyURL(domain: string): string {
    return "https://mta-sts." + stripTrailingDot(domain) + "/.well-known/mta-sts.txt";
}

/**
 * Hosts the given services' MX records point at, verbatim. Callers differ only
 * in where they get the service list from: the compliance context, or the zone
 * store the editor reads.
 */
export function mxHostsFrom(services: Array<{ _svctype?: string; Service?: unknown }>): string[] {
    const hosts: string[] = [];
    for (const s of services) {
        const mx = (s.Service as Record<string, any> | undefined)?.mx;
        if (!Array.isArray(mx)) continue;
        for (const entry of mx) {
            const target = entry?.Mx;
            if (typeof target === "string" && target) hosts.push(target);
        }
    }
    return hosts;
}

/** Hosts the zone's apex MX records point at, as the policy should list them. */
export function getZoneApexMxHosts(ctx: ComplianceContext): string[] {
    return mxHostsFrom(ctx.findServices("", "svcs.MXs"));
}

/**
 * Validates the `_mta-sts` TXT record: its owner name, version and policy id.
 *
 * `parse` is injected because the two services store the record under
 * different field names but agree on its syntax.
 */
export function txtRecordIssues(
    txt: { Txt?: unknown; Hdr?: { Name?: unknown } } | undefined | null,
    parse: (raw: string) => { v?: string; id?: string },
): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    if (!txt) return issues;

    const txtValue: string = typeof txt.Txt === "string" ? txt.Txt : "";
    const name: string = typeof txt.Hdr?.Name === "string" ? txt.Hdr.Name : "";

    // Owner name must be _mta-sts.<domain>.
    if (name && !/^_mta-sts(\.|$)/i.test(name)) {
        issues.push({
            id: "mta_sts.wrong-owner-name",
            severity: "error",
            params: { name },
            docUrl: RFC + "#section-3.1",
        });
    }

    if (!txtValue.trim()) return issues;

    let val: { v?: string; id?: string };
    try {
        val = parse(txtValue);
    } catch {
        issues.push({ id: "mta_sts.parse-error", severity: "error", field: "txt" });
        return issues;
    }

    if (!val.v) {
        issues.push({
            id: "mta_sts.missing-version",
            severity: "error",
            field: "v",
            docUrl: RFC + "#section-3.1",
        });
    } else if (val.v !== "STSv1") {
        issues.push({
            id: "mta_sts.invalid-version",
            severity: "error",
            params: { version: val.v },
            field: "v",
            docUrl: RFC + "#section-3.1",
        });
    }

    if (val.id === undefined || val.id === "") {
        issues.push({
            id: "mta_sts.missing-id",
            severity: "error",
            field: "id",
            docUrl: RFC + "#section-3.1",
        });
    } else if (!ID_RE.test(val.id)) {
        issues.push({
            id: "mta_sts.invalid-id",
            severity: "error",
            params: { id: val.id },
            field: "id",
            docUrl: RFC + "#section-3.1",
        });
    }

    return issues;
}

/** A policy, however it was obtained: fetched from the network or edited here. */
export interface PolicyFields {
    version?: string;
    mode?: string;
    mx?: string[];
    maxAge?: number;
}

/**
 * Validates the policy itself: version, mode, mx patterns and max_age, plus a
 * cross-check of the patterns against the zone's apex MX records.
 *
 * `url` is only ever used to build the message, so a policy that is still
 * being edited passes the URL it *will* be served from.
 */
export function policyIssues(
    policy: PolicyFields,
    ctx: ComplianceContext,
    url: string,
): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];

    if (!policy.version) {
        issues.push({
            id: "mta_sts.policy-missing-version",
            severity: "error",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    } else if (policy.version !== "STSv1") {
        issues.push({
            id: "mta_sts.policy-invalid-version",
            severity: "error",
            params: { url, version: policy.version },
            docUrl: RFC + "#section-3.2",
        });
    }

    const mode = policy.mode ?? "";
    if (!mode) {
        issues.push({
            id: "mta_sts.policy-missing-mode",
            severity: "error",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    } else if (!VALID_MODES.has(mode)) {
        issues.push({
            id: "mta_sts.policy-invalid-mode",
            severity: "error",
            params: { url, mode },
            docUrl: RFC + "#section-3.2",
        });
    } else if (mode === "none") {
        issues.push({
            id: "mta_sts.policy-mode-none",
            severity: "warning",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    } else if (mode === "testing") {
        issues.push({
            id: "mta_sts.policy-mode-testing",
            severity: "info",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    }

    const mxList = policy.mx ?? [];
    if ((mode === "enforce" || mode === "testing") && mxList.length === 0) {
        issues.push({
            id: "mta_sts.policy-missing-mx",
            severity: "error",
            params: { url, mode },
            docUrl: RFC + "#section-3.2",
        });
    }

    // Cross-check the policy patterns against the apex MX records of the
    // current zone (RFC 8461 sec. 4.1). Only meaningful when the policy
    // actually filters mail and the zone state is known.
    if ((mode === "enforce" || mode === "testing") && ctx.zone) {
        const zoneMx = getZoneApexMxHosts(ctx);
        if (zoneMx.length === 0 && mxList.length > 0) {
            issues.push({
                id: "mta_sts.zone-no-mx",
                severity: "warning",
                params: { url },
                docUrl: RFC + "#section-4.1",
            });
        } else if (zoneMx.length > 0 && mxList.length > 0) {
            for (const host of zoneMx) {
                const matched = mxList.some((p) => mtaStsPatternMatches(p, host));
                if (!matched) {
                    issues.push({
                        id: "mta_sts.zone-mx-not-covered",
                        severity: mode === "enforce" ? "error" : "warning",
                        params: { url, host, mode },
                        field: host,
                        docUrl: RFC + "#section-4.1",
                    });
                }
            }
            for (const pattern of mxList) {
                const matched = zoneMx.some((h) => mtaStsPatternMatches(pattern, h));
                if (!matched) {
                    issues.push({
                        id: "mta_sts.policy-mx-unused",
                        severity: "info",
                        params: { url, pattern },
                        docUrl: RFC + "#section-4.1",
                    });
                }
            }
        }
    }

    const maxAge = policy.maxAge ?? 0;
    if (!maxAge) {
        issues.push({
            id: "mta_sts.policy-missing-max-age",
            severity: "error",
            params: { url },
            docUrl: RFC + "#section-3.2",
        });
    } else if (maxAge < 0 || maxAge > MAX_AGE_HARD_LIMIT) {
        issues.push({
            id: "mta_sts.policy-invalid-max-age",
            severity: "error",
            params: { url, maxAge },
            docUrl: RFC + "#section-3.2",
        });
    } else if (maxAge < MAX_AGE_RECOMMENDED_MIN) {
        issues.push({
            id: "mta_sts.policy-short-max-age",
            severity: "warning",
            params: { url, maxAge },
            docUrl: RFC + "#section-3.2",
        });
    }

    return issues;
}

/**
 * Maps a failed policy fetch to an issue. Returns null when the fetch
 * succeeded and the policy fields should be validated instead.
 */
export function fetchFailureIssue(resp: {
    status?: string;
    url?: string;
    errorMsg?: string;
    httpCode?: number;
    redirected?: boolean;
}): ComplianceIssue | null {
    const url = resp.url ?? "";

    switch (resp.status) {
        case "ok":
            return null;
        case "dns-error":
            return {
                id: "mta_sts.policy-dns-error",
                severity: "error",
                params: { url },
                docUrl: RFC + "#section-3.3",
            };
        case "tls-error":
            return {
                id: "mta_sts.policy-tls-error",
                severity: "error",
                params: { url, error: resp.errorMsg ?? "" },
                docUrl: RFC + "#section-3.3",
            };
        case "not-found":
            return {
                id: "mta_sts.policy-not-found",
                severity: "error",
                params: { url },
                docUrl: RFC + "#section-3.3",
            };
        case "http-error":
            return {
                id: resp.redirected ? "mta_sts.policy-redirect" : "mta_sts.policy-http-error",
                severity: "warning",
                params: { url, code: resp.httpCode ?? 0 },
                docUrl: RFC + "#section-3.3",
            };
        case "fetch-error":
            return {
                id: "mta_sts.policy-fetch-error",
                severity: "warning",
                params: { url, error: resp.errorMsg ?? "" },
            };
        case "too-large":
            return {
                id: "mta_sts.policy-too-large",
                severity: "error",
                params: { url },
            };
        default:
            // Unknown status: ignore so a future backend addition does not
            // surface a localized "undefined" string.
            return null;
    }
}

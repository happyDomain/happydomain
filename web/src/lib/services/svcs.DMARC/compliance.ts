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

import { checkDMARCReportAuth } from "$lib/api/resolver";
import { fqdn } from "$lib/dns";
import {
    type ComplianceContext,
    type ComplianceIssue,
    isValidHostname,
    registerValidators,
} from "$lib/services/compliance";
import { parseDMARC, type DMARCValue } from "./model";
import { parseDKIM } from "$lib/services/svcs.DKIMRecord/model.svelte";
import type { ServiceWithValue } from "$lib/model/service.svelte";

const POLICY_VALUES = new Set(["none", "quarantine", "reject"]);
const ALIGNMENT_VALUES = new Set(["r", "s"]);
const FO_VALUES = new Set(["0", "1", "d", "s"]);
const RF_VALUES = new Set(["afrf"]);
const RFC = "https://www.rfc-editor.org/rfc/rfc7489";

function isMailto(uri: string): boolean {
    return /^mailto:/i.test(uri.trim());
}

function isHttp(uri: string): boolean {
    return /^https?:/i.test(uri.trim());
}

// protectedDomainOf returns the apex domain that the DMARC record protects,
// derived from the editing context. The DMARC owner name lives at
// "_dmarc.<protected>", so we strip the "_dmarc" leading label of ctx.dn and
// resolve against ctx.origin.
function protectedDomainOf(ctx: ComplianceContext): string {
    const sub = (ctx.dn ?? "").replace(/^_dmarc(\.|$)/i, "$1").replace(/^\./, "");
    const origin = ctx.origin?.domain ?? "";
    const protectedFqdn = fqdn(sub || "@", origin);
    return protectedFqdn.replace(/\.$/, "").toLowerCase();
}

// A dmarc-uri carries an optional size limit: "!" followed by a count and an
// optional unit (RFC 7489 sec. 6.4). It is not part of the URI itself.
const SIZE_SUFFIX_RE = /!\d+[kmgt]?$/i;

function stripSizeLimit(uri: string): string {
    return uri.trim().replace(SIZE_SUFFIX_RE, "");
}

/**
 * Reads the destination out of an http(s) report URI. Returns null when the
 * URI does not parse or carries no host, since neither leaves anything to
 * check further.
 */
function httpTarget(uri: string): { host: string; secure: boolean } | null {
    let url: URL;
    try {
        url = new URL(stripSizeLimit(uri));
    } catch {
        return null;
    }
    if (!url.hostname) return null;
    return { host: url.hostname.toLowerCase(), secure: url.protocol === "https:" };
}

const IPV4_RE = /^\d{1,3}(\.\d{1,3}){3}$/;

// mailtoTarget extracts the destination domain from a "mailto:" URI, dropping
// the optional "!size" suffix allowed by RFC 7489 sec. 6.2 ("!10m" etc.).
// Returns null when the URI is not a syntactically valid mailto address.
function mailtoTarget(uri: string): { address: string; domain: string } | null {
    const stripped = uri.trim().replace(/^mailto:/i, "");
    if (!stripped) return null;
    const address = stripped.split("!")[0];
    const at = address.lastIndexOf("@");
    if (at <= 0 || at === address.length - 1) return null;
    return { address, domain: address.slice(at + 1).toLowerCase() };
}

/**
 * Tells whether two names belong to the same Organizational Domain, which is
 * what RFC 7489 sec. 7.1 compares before requiring an authorization record.
 * Deriving the real Organizational Domain needs a public suffix list, which
 * the front-end does not carry, so only the direction that is safe without one
 * is trusted: a report sent to the protected domain itself or to one of its
 * own subdomains, since the Domain Owner already controls that whole subtree.
 * A destination that is instead a parent of the protected domain is NOT
 * assumed to be the real Organizational Domain, because without a PSL there
 * is no way to tell a genuine parent organization apart from a shared-hosting
 * or public-suffix-like domain that merely happens to be an ancestor label,
 * so that direction still goes through the authorization-record check.
 */
function sameOrganizationalDomain(host: string, protectedDomain: string): boolean {
    if (!host || !protectedDomain) return false;
    return host === protectedDomain || host.endsWith("." + protectedDomain);
}

/**
 * Lists the report destinations of a record, one entry per rua/ruf URI whose
 * host could be read. Both schemes are collected: sec. 7.1 keys the
 * authorization on the destination domain, not on the way reports reach it.
 */
function reportDestinations(
    val: DMARCValue,
): { tag: "rua" | "ruf"; host: string; destination: string }[] {
    const destinations: { tag: "rua" | "ruf"; host: string; destination: string }[] = [];

    for (const tag of ["rua", "ruf"] as const) {
        for (const uri of val[tag] ?? []) {
            const u = uri.trim();
            if (isMailto(u)) {
                const target = mailtoTarget(u);
                if (target) {
                    destinations.push({ tag, host: target.domain, destination: target.address });
                }
            } else if (isHttp(u)) {
                const target = httpTarget(u);
                if (target && !target.host.startsWith("[") && !IPV4_RE.test(target.host)) {
                    destinations.push({ tag, host: target.host, destination: stripSizeLimit(u) });
                }
            }
        }
    }

    return destinations;
}

/**
 * Keeps the destinations that live outside the Organizational Domain of the
 * protected domain, the ones sec. 7.1 asks an authorization record for, first
 * reference wins so the issue can point back at a URI.
 */
function externalDestinations(
    val: DMARCValue,
    protectedDomain: string,
): Map<string, { tag: "rua" | "ruf"; destination: string }> {
    const externals = new Map<string, { tag: "rua" | "ruf"; destination: string }>();

    for (const { tag, host, destination } of reportDestinations(val)) {
        if (sameOrganizationalDomain(host, protectedDomain)) continue;
        if (!externals.has(host)) externals.set(host, { tag, destination });
    }

    return externals;
}

/**
 * Tells whether the DKIM selectors of the zone can actually sign anything: a
 * selector whose key is revoked (empty p=) or still in testing mode (t=y) is
 * published, but yields no alignment DMARC can rely on. Services whose record
 * is unreadable are assumed usable, so a partially loaded zone stays quiet.
 */
function dkimUsabilityIssues(services: ServiceWithValue[]): ComplianceIssue[] {
    let known = 0;
    let revoked = 0;
    let testing = 0;

    for (const service of services) {
        const txt = (service.Service as Record<string, any> | undefined)?.txt;
        const value = typeof txt?.Txt === "string" ? txt.Txt.trim() : "";
        if (!value) continue;

        let dkim;
        try {
            dkim = parseDKIM(value);
        } catch {
            continue;
        }

        known++;
        if (!dkim.p) revoked++;
        else if ((dkim.t ?? []).includes("y")) testing++;
    }

    if (known === 0) return [];
    if (revoked === known) {
        return [{ id: "dmarc.all-dkim-revoked", severity: "warning", docUrl: RFC + "#section-3" }];
    }
    if (revoked + testing === known) {
        return [{ id: "dmarc.all-dkim-testing", severity: "warning", docUrl: RFC + "#section-3" }];
    }
    return [];
}

function dmarcSync(raw: Record<string, any>, ctx: ComplianceContext): ComplianceIssue[] {
    const issues: ComplianceIssue[] = [];
    const txt = raw?.txt;
    if (!txt) return issues;

    const txtValue: string = typeof txt.Txt === "string" ? txt.Txt : "";
    const name: string = typeof txt.Hdr?.Name === "string" ? txt.Hdr.Name : "";

    // The DMARC TXT must live at _dmarc.<domain>. The editor controls the
    // owner name, but a rename could land it elsewhere.
    if (name && !/^_dmarc(\.|$)/i.test(name)) {
        issues.push({
            id: "dmarc.wrong-owner-name",
            severity: "error",
            params: { name },
            docUrl: RFC + "#section-6.1",
        });
    }

    if (!txtValue.trim()) return issues;

    let val: DMARCValue;
    try {
        val = parseDMARC(txtValue);
    } catch {
        issues.push({ id: "dmarc.parse-error", severity: "error", field: "txt" });
        return issues;
    }

    // v=DMARC1 must be present and first (RFC 7489 sec. 6.3).
    if (!val.v) {
        issues.push({
            id: "dmarc.missing-version",
            severity: "error",
            field: "v",
            docUrl: RFC + "#section-6.3",
        });
    } else if (val.v !== "DMARC1") {
        issues.push({
            id: "dmarc.invalid-version",
            severity: "error",
            params: { version: val.v },
            field: "v",
            docUrl: RFC + "#section-6.3",
        });
    }

    // p= is mandatory (RFC 7489 sec. 6.3).
    if (!val.p) {
        issues.push({
            id: "dmarc.missing-policy",
            severity: "error",
            field: "p",
            docUrl: RFC + "#section-6.3",
        });
    } else if (!POLICY_VALUES.has(val.p)) {
        issues.push({
            id: "dmarc.invalid-policy",
            severity: "error",
            params: { policy: val.p },
            field: "p",
            docUrl: RFC + "#section-6.3",
        });
    } else if (val.p === "none") {
        issues.push({
            id: "dmarc.monitoring-only",
            severity: "info",
            field: "p",
            docUrl: RFC + "#section-6.3",
        });
    }

    // sp= subdomain policy.
    if (val.sp !== undefined && val.sp !== "" && !POLICY_VALUES.has(val.sp)) {
        issues.push({
            id: "dmarc.invalid-sp",
            severity: "error",
            params: { policy: val.sp },
            field: "sp",
        });
    }

    // adkim / aspf alignment.
    if (val.adkim !== undefined && val.adkim !== "" && !ALIGNMENT_VALUES.has(val.adkim)) {
        issues.push({
            id: "dmarc.invalid-alignment",
            severity: "error",
            params: { tag: "adkim", value: val.adkim },
            field: "adkim",
        });
    }
    if (val.aspf !== undefined && val.aspf !== "" && !ALIGNMENT_VALUES.has(val.aspf)) {
        issues.push({
            id: "dmarc.invalid-alignment",
            severity: "error",
            params: { tag: "aspf", value: val.aspf },
            field: "aspf",
        });
    }

    // pct must be 0..100.
    if (val.pct !== undefined && val.pct !== "" && val.pct !== null) {
        const pct = typeof val.pct === "number" ? val.pct : Number.parseInt(String(val.pct), 10);
        if (!Number.isInteger(pct) || pct < 0 || pct > 100) {
            issues.push({
                id: "dmarc.invalid-pct",
                severity: "error",
                params: { pct: String(val.pct) },
                field: "pct",
            });
        } else if (pct < 100) {
            issues.push({
                id: "dmarc.partial-deployment",
                severity: "info",
                params: { pct },
                field: "pct",
                docUrl: RFC + "#section-6.6.4",
            });
        }
    }

    // ri must be a positive integer.
    if (val.ri !== undefined && val.ri !== "") {
        const ri = Number.parseInt(String(val.ri), 10);
        if (!Number.isInteger(ri) || ri <= 0) {
            issues.push({
                id: "dmarc.invalid-ri",
                severity: "error",
                params: { ri: String(val.ri) },
                field: "ri",
            });
        }
    }

    // fo values must be in {0,1,d,s}. Combinations like "d:s" are allowed.
    for (const f of val.fo ?? []) {
        const trimmed = f.trim();
        if (!trimmed) continue;
        if (!FO_VALUES.has(trimmed)) {
            issues.push({
                id: "dmarc.invalid-fo",
                severity: "warning",
                params: { value: trimmed },
                field: "fo",
            });
        }
    }

    // rf format. Only "afrf" is defined.
    for (const r of val.rf ?? []) {
        const trimmed = r.trim();
        if (!trimmed) continue;
        if (!RF_VALUES.has(trimmed)) {
            issues.push({
                id: "dmarc.unknown-rf",
                severity: "warning",
                params: { value: trimmed },
                field: "rf",
            });
        }
    }

    // rua / ruf URIs.
    const uriCheck = (uri: string, tag: "rua" | "ruf") => {
        const u = uri.trim();
        if (!u) {
            issues.push({
                id: "dmarc.empty-uri",
                severity: "warning",
                params: { tag },
                field: tag,
            });
            return;
        }
        if (!isMailto(u) && !isHttp(u)) {
            issues.push({
                id: "dmarc.invalid-uri-scheme",
                severity: "error",
                params: { tag, uri: u },
                field: tag,
                docUrl: RFC + "#section-6.2",
            });
            return;
        }
        if (isHttp(u)) {
            const target = httpTarget(u);
            if (!target) {
                issues.push({
                    id: "dmarc.invalid-http-uri",
                    severity: "error",
                    params: { tag, uri: u },
                    field: tag,
                    docUrl: RFC + "#section-6.2",
                });
                return;
            }
            if (!target.secure) {
                issues.push({
                    id: "dmarc.report-uri-insecure",
                    severity: "warning",
                    params: { tag, host: target.host },
                    field: tag,
                });
            }
            // An address literal resolves, but no certificate matches it and
            // sec. 7.1 has no domain left to ask for an authorization.
            if (target.host.startsWith("[") || IPV4_RE.test(target.host)) {
                issues.push({
                    id: "dmarc.report-host-ip-literal",
                    severity: "warning",
                    params: { tag, host: target.host },
                    field: tag,
                    docUrl: RFC + "#section-7.1",
                });
                return;
            }
            if (!isValidHostname(target.host)) {
                issues.push({
                    id: "dmarc.invalid-report-host",
                    severity: "error",
                    params: { tag, host: target.host },
                    field: tag,
                });
                return;
            }
            if (!target.host.includes(".")) {
                issues.push({
                    id: "dmarc.report-host-single-label",
                    severity: "warning",
                    params: { tag, host: target.host },
                    field: tag,
                });
            }
            return;
        }
        if (isMailto(u)) {
            const addr = u.replace(/^mailto:/i, "");
            // Strip optional !size suffix (RFC 7489 sec. 6.2 allows "!10m" etc.).
            const local = addr.split("!")[0];
            if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(local)) {
                issues.push({
                    id: "dmarc.invalid-mailto",
                    severity: "error",
                    params: { tag, uri: u },
                    field: tag,
                });
            }
        }
    };
    for (const u of val.rua ?? []) uriCheck(u, "rua");
    for (const u of val.ruf ?? []) uriCheck(u, "ruf");

    // Surface external reporting destinations. RFC 7489 sec. 7.1 requires the
    // external domain to publish an authorization record; we hint at it here
    // and the async validator does the actual lookup.
    const protectedDomain = protectedDomainOf(ctx);
    if (protectedDomain) {
        for (const [domain, ref] of externalDestinations(val, protectedDomain)) {
            issues.push({
                id: "dmarc.external-reporting",
                severity: "info",
                params: { domain },
                field: ref.tag,
                docUrl: RFC + "#section-7.1",
            });
        }
    }

    // Cross-record checks: DMARC depends on at least one aligned mechanism
    // (DKIM or SPF). When the zone state is available, surface configurations
    // where alignment is structurally impossible.
    const policy = val.p ?? "";
    const enforcing = policy === "quarantine" || policy === "reject";
    const dkimRecords = ctx.findAllServices("svcs.DKIMRecord");
    const spfRecords = ctx.findAllServices("svcs.SPF");
    const hasDkim = dkimRecords.length > 0;
    const hasSpf = spfRecords.length > 0;

    if (ctx.zone) {
        if (!hasDkim && !hasSpf) {
            issues.push({
                id: enforcing ? "dmarc.no-alignment-source-enforcing" : "dmarc.no-alignment-source",
                severity: enforcing ? "error" : "warning",
                docUrl: RFC + "#section-3",
            });
        } else if (!hasDkim) {
            // SPF alone stops aligning as soon as a message is relayed: only a
            // DKIM signature survives a mailing list or a forwarder.
            issues.push({
                id: "dmarc.no-dkim-record",
                severity: "warning",
                docUrl: RFC + "#section-3",
            });
            if (val.adkim === "s") {
                issues.push({
                    id: "dmarc.strict-dkim-no-record",
                    severity: "warning",
                    docUrl: RFC + "#section-3.1",
                });
            }
        } else {
            issues.push(...dkimUsabilityIssues(dkimRecords));
        }
    }

    return issues;
}

// dmarcAsync verifies the RFC 7489 sec. 7.1 cross-domain reporting
// authorization for every distinct external destination found in the rua/ruf
// lists, whatever its scheme. The check is skipped when no external destination is in use,
// or when the protected owner cannot be derived from the editing context.
async function dmarcAsync(
    raw: Record<string, any>,
    ctx: ComplianceContext,
    signal: AbortSignal,
): Promise<ComplianceIssue[]> {
    const txt = raw?.txt;
    const txtValue: string = typeof txt?.Txt === "string" ? txt.Txt : "";
    if (!txtValue.trim()) return [];

    let val: DMARCValue;
    try {
        val = parseDMARC(txtValue);
    } catch {
        return [];
    }

    const protectedDomain = protectedDomainOf(ctx);
    if (!protectedDomain) return [];

    const externals = externalDestinations(val, protectedDomain);
    if (externals.size === 0) return [];

    const issues: ComplianceIssue[] = [];
    for (const [domain, ref] of externals) {
        const resp = await checkDMARCReportAuth(
            { owner: protectedDomain, externalDomain: domain },
            signal,
        );
        if (signal.aborted) return [];

        const queriedName = resp.queriedName ?? "";
        switch (resp.status) {
            case "ok":
                break;
            case "no-dmarc-record":
                issues.push({
                    id: "dmarc.report-auth-no-dmarc",
                    severity: "error",
                    params: { domain, queriedName, destination: ref.destination, tag: ref.tag },
                    field: ref.tag,
                    docUrl: RFC + "#section-7.1",
                });
                break;
            case "not-found":
                issues.push({
                    id: "dmarc.report-auth-missing",
                    severity: "error",
                    params: { domain, queriedName, destination: ref.destination, tag: ref.tag },
                    field: ref.tag,
                    docUrl: RFC + "#section-7.1",
                });
                break;
            case "dns-error":
            case "resolver-error":
            default:
                issues.push({
                    id: "dmarc.report-auth-resolver-error",
                    severity: "warning",
                    params: {
                        domain,
                        queriedName,
                        tag: ref.tag,
                        error: resp.errorMsg ?? "",
                    },
                    field: ref.tag,
                    docUrl: RFC + "#section-7.1",
                });
                break;
        }
    }
    return issues;
}

registerValidators("svcs.DMARC", { sync: dmarcSync, async: dmarcAsync });

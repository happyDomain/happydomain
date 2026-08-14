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

import type { Domain } from "$lib/model/domain";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import type { Zone } from "$lib/model/zone";

export type Severity = "error" | "warning" | "info";

export interface ComplianceIssue {
    /**
     * Stable identifier of the rule, e.g. "spf.too-many-lookups".
     * Used as the i18n key under `compliance.<id>.title` and `compliance.<id>.detail`.
     */
    id: string;
    severity: Severity;
    /**
     * Optional interpolation parameters for the i18n message.
     */
    params?: Record<string, string | number>;
    /**
     * Optional path inside the edited value, used for future inline highlighting.
     * Examples: "f[3]" (4th SPF directive), "rua[0]", "p", "selector".
     */
    field?: string;
    /**
     * Optional documentation URL (RFC, project docs, ...).
     */
    docUrl?: string;
}

export interface ComplianceContext {
    /** Subdomain currently being edited (relative to the origin). */
    dn: string;
    /** Domain that hosts the zone. */
    origin: Domain;
    /** Current zone state, when known. */
    zone: Zone | null;
    /**
     * Look up sibling services in the zone.
     * @param subdomain Subdomain (relative to origin) to search in. Empty string for apex.
     * @param type Optional service type filter (e.g. "svcs.DKIM").
     */
    findServices(subdomain: string, type?: string): ServiceWithValue[];
    /**
     * Iterate every service in the zone, optionally filtered by service type.
     * Used by cross-record validators (e.g. DMARC checking that DKIM or SPF
     * records exist anywhere in the zone).
     */
    findAllServices(type?: string): ServiceWithValue[];
}

export type SyncValidator = (raw: Record<string, any>, ctx: ComplianceContext) => ComplianceIssue[];

export type AsyncValidator = (
    raw: Record<string, any>,
    ctx: ComplianceContext,
    signal: AbortSignal,
) => Promise<ComplianceIssue[]>;

export interface ServiceValidators {
    sync?: SyncValidator;
    async?: AsyncValidator;
}

const registry: Record<string, ServiceValidators> = {};

export function registerValidators(svctype: string, validators: ServiceValidators): void {
    registry[svctype] = validators;
}

export function getValidators(svctype: string): ServiceValidators | undefined {
    return registry[svctype];
}

export function hasValidators(svctype: string): boolean {
    return registry[svctype] !== undefined;
}

/**
 * Editor values usually come in as either a single record object or an array
 * of them, depending on the underlying ServiceBody. asArray normalizes them
 * into an iterable shape for record-list validators.
 */
export function asArray<T>(raw: unknown): T[] {
    if (!raw) return [];
    return Array.isArray(raw) ? (raw as T[]) : [raw as T];
}

/** RRtype of a CNAME, as carried in the Hdr of an alias record. */
export const TYPE_CNAME = 5;

const HOSTNAME_LABEL_RE = /^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$/;

/** Case-insensitive hexadecimal string test, for fingerprints and digests. */
export const HEX_RE = /^[0-9a-f]*$/i;

/** Whether a value is a valid 16-bit unsigned integer, as DNS uint16 fields require. */
export function isUint16(v: unknown): v is number {
    return typeof v === "number" && Number.isInteger(v) && v >= 0 && v <= 65535;
}

/** Drops the trailing dots and lowercases, so two spellings of the same FQDN compare equal. */
export function normalizeFqdn(name: string): string {
    return name.replace(/\.+$/, "").toLowerCase();
}

/** Checks the LDH syntax of a hostname, expected already normalized. */
export function isValidHostname(name: string): boolean {
    if (!name || name.length > 253) return false;
    const labels = name.split(".");
    return labels.every((l) => HOSTNAME_LABEL_RE.test(l));
}

/**
 * Same as isValidHostname, but tolerates the underscore-prefixed labels the
 * attribute leaves of RFC 8552 are made of (_sip._tcp, _dmarc, _domainkey...).
 * They are perfectly valid domain names, they are just not host names.
 */
export function isValidDnsName(name: string): boolean {
    if (!name || name.length > 253) return false;
    return name.split(".").every((l) => HOSTNAME_LABEL_RE.test(l.replace(/^_/, "")));
}

/**
 * Returns the in-zone subdomain (relative to origin) for a target FQDN,
 * or null when the target is outside the edited zone.
 */
export function inZoneSubdomain(target: string, originFqdn: string): string | null {
    const t = normalizeFqdn(target);
    const o = normalizeFqdn(originFqdn);
    if (!o) return null;
    if (t === o) return "";
    if (t.endsWith("." + o)) return t.slice(0, -(o.length + 1));
    return null;
}

/** The FQDN of the zone being edited. */
export function originFqdn(ctx: ComplianceContext): string {
    return (ctx.origin as { domain?: string })?.domain ?? "";
}

/** Builds the FQDN of a subdomain of the edited zone. */
export function subdomainFqdn(subdomain: string, origin: string): string {
    const o = normalizeFqdn(origin);
    return subdomain ? normalizeFqdn(subdomain) + "." + o : o;
}

/**
 * Builds the FQDN a record is published at, from the owner name it carries.
 * Records travel relative to the subdomain holding their service, so an empty
 * Hdr.Name means the service subdomain itself.
 */
export function recordFqdn(hdrName: string | undefined, ctx: ComplianceContext): string {
    const labels = [hdrName ?? "", ctx.dn].filter((l) => l !== "" && l !== "@");
    return subdomainFqdn(labels.join("."), originFqdn(ctx));
}

/** Tells whether a name of the edited zone is the owner of a CNAME. */
export function isCnameOwner(ctx: ComplianceContext, subdomain: string): boolean {
    const cname = (service: { Service?: unknown }) =>
        (service.Service as Record<string, any> | undefined)?.record?.Hdr?.Rrtype === TYPE_CNAME;

    // A SubAlias is owned by an underscore name carried in its own header, not
    // by the subdomain it is attached to: only an empty header name means it
    // is published at the subdomain itself.
    const specialCname = (service: { Service?: unknown }) =>
        ((service.Service as Record<string, any> | undefined)?.cname?.Hdr?.Name ?? "") === "";

    return (
        ctx.findServices(subdomain, "svcs.Alias").some(cname) ||
        ctx.findServices(subdomain, "svcs.SpecialCNAME").some(specialCname)
    );
}

/**
 * Tells whether a name of the edited zone resolves to an address: either an
 * A/AAAA of its own, or an alias the provider will resolve for us.
 */
export function hasAddress(ctx: ComplianceContext, subdomain: string): boolean {
    return (
        ctx.findServices(subdomain, "abstract.Server").length > 0 ||
        ctx.findServices(subdomain, "svcs.Alias").length > 0
    );
}

export function buildContext(dn: string, origin: Domain, zone: Zone | null): ComplianceContext {
    return {
        dn,
        origin,
        zone,
        findServices(subdomain, type) {
            if (!zone) return [];
            const services = zone.services?.[subdomain] ?? [];
            return type ? services.filter((s) => s._svctype === type) : services.slice();
        },
        findAllServices(type) {
            if (!zone?.services) return [];
            const out: ServiceWithValue[] = [];
            for (const services of Object.values(zone.services)) {
                for (const svc of services) {
                    if (!type || svc._svctype === type) out.push(svc);
                }
            }
            return out;
        },
    };
}

export function runSyncValidators(
    svctype: string,
    raw: Record<string, any>,
    ctx: ComplianceContext,
): ComplianceIssue[] {
    const v = registry[svctype];
    if (!v?.sync) return [];
    try {
        return v.sync(raw, ctx);
    } catch (err) {
        // A failing validator must not break the editor.
        console.error(`compliance[${svctype}] sync error`, err);
        return [];
    }
}

export async function runAsyncValidators(
    svctype: string,
    raw: Record<string, any>,
    ctx: ComplianceContext,
    signal: AbortSignal,
): Promise<ComplianceIssue[]> {
    const v = registry[svctype];
    if (!v?.async) return [];
    try {
        return await v.async(raw, ctx, signal);
    } catch (err) {
        if ((err as { name?: string })?.name === "AbortError") return [];
        console.error(`compliance[${svctype}] async error`, err);
        return [];
    }
}

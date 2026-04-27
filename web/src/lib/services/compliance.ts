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
}

export type SyncValidator = (
    raw: Record<string, any>,
    ctx: ComplianceContext,
) => ComplianceIssue[];

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

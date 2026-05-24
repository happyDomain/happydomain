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

import type { CheckerCheckerOptionDocumentation, HappydnsExecutionStatus, HappydnsStatus } from "$lib/api-base/types.gen";

// HappydnsStatus numeric constants matching Go's checker.Status values.
// Severity ordering: UNKNOWN (0) < OK (1) < INFO (2) < WARN (3) < CRIT (4) < ERROR (5).
export const StatusUnknown: HappydnsStatus = 0;
export const StatusOK: HappydnsStatus = 1;
export const StatusInfo: HappydnsStatus = 2;
export const StatusWarn: HappydnsStatus = 3;
export const StatusCrit: HappydnsStatus = 4;
export const StatusError: HappydnsStatus = 5;

export function getStatusColor(status: HappydnsStatus | undefined): string {
    switch (status) {
        case StatusOK: return "success";
        case StatusInfo: return "info";
        case StatusUnknown: return "secondary";
        case StatusWarn: return "warning";
        case StatusCrit: return "danger";
        case StatusError: return "danger";
        default: return "secondary";
    }
}

export function getStatusI18nKey(status: HappydnsStatus | undefined): string {
    switch (status) {
        case StatusOK: return "checkers.status.ok";
        case StatusInfo: return "checkers.status.info";
        case StatusUnknown: return "checkers.status.unknown";
        case StatusWarn: return "checkers.status.warning";
        case StatusCrit: return "checkers.status.critical";
        case StatusError: return "checkers.status.error";
        default: return "checkers.status.not-run";
    }
}

export function getStatusIcon(status: HappydnsStatus | undefined): string {
    switch (status) {
        case StatusOK: return "check-circle-fill";
        case StatusInfo: return "info-circle-fill";
        case StatusWarn: return "exclamation-triangle-fill";
        case StatusCrit: return "exclamation-octagon-fill";
        case StatusError: return "exclamation-octagon-fill";
        default: return "question-circle-fill";
    }
}

// HappydnsExecutionStatus: 0=Pending, 1=Running, 2=Done, 3=Failed, 4=RateLimited

export function getExecutionStatusColor(status: HappydnsExecutionStatus | undefined): string {
    switch (status) {
        case 0: return "secondary";
        case 1: return "primary";
        case 2: return "success";
        case 3: return "danger";
        case 4: return "warning";
        default: return "secondary";
    }
}

export function getExecutionStatusI18nKey(status: HappydnsExecutionStatus | undefined): string {
    switch (status) {
        case 0: return "checkers.execution.status.pending";
        case 1: return "checkers.execution.status.running";
        case 2: return "checkers.execution.status.done";
        case 3: return "checkers.execution.status.failed";
        case 4: return "checkers.execution.status.rate-limited";
        default: return "checkers.execution.status.unknown";
    }
}

export function withInheritedPlaceholders(
    opts: CheckerCheckerOptionDocumentation[],
    optionValues: Record<string, unknown>,
    inheritedValues: Record<string, unknown>,
): CheckerCheckerOptionDocumentation[] {
    return opts.map((opt) => {
        if (
            opt.id &&
            optionValues[opt.id] === undefined &&
            inheritedValues[opt.id] !== undefined
        ) {
            return { ...opt, placeholder: String(inheritedValues[opt.id]) };
        }
        return opt;
    });
}

interface OptionDocGroup {
    runOpts?: CheckerCheckerOptionDocumentation[];
    adminOpts?: CheckerCheckerOptionDocumentation[];
    userOpts?: CheckerCheckerOptionDocumentation[];
    domainOpts?: CheckerCheckerOptionDocumentation[];
    serviceOpts?: CheckerCheckerOptionDocumentation[];
}

interface CheckerWithOptions {
    options?: OptionDocGroup;
    rules?: { options?: OptionDocGroup }[];
}

export function collectAllOptionDocs(
    status: CheckerWithOptions,
): CheckerCheckerOptionDocumentation[] {
    return [
        ...(status.options?.runOpts || []),
        ...(status.options?.adminOpts || []),
        ...(status.options?.userOpts || []),
        ...(status.options?.domainOpts || []),
        ...(status.options?.serviceOpts || []),
        ...(status.rules || []).flatMap((r) => [
            ...(r.options?.runOpts || []),
            ...(r.options?.adminOpts || []),
            ...(r.options?.userOpts || []),
            ...(r.options?.domainOpts || []),
            ...(r.options?.serviceOpts || []),
        ]),
    ].filter((o) => !o.noOverride);
}

// CheckerPageScope identifies which scope a configuration page targets. It
// drives the editable/read-only split of the checker's option groups.
//
// Semantics (matching the existing per-route layouts):
//   - "admin":   broadest. User/admin groups are editable; domain/service/run
//                are read-only (set by deeper pages or at trigger time).
//   - "domain":  domain/user/admin editable; service/run read-only.
//   - "service": domain/service/user/admin editable; run read-only.
export type CheckerPageScope = "admin" | "domain" | "service";

export interface OptionGroup {
    label: string;
    opts: CheckerCheckerOptionDocumentation[];
}

export interface KeyedOptionGroup extends OptionGroup {
    key: string;
}

// buildOptionGroupLayout derives the editable / read-only group lists from
// the checker's option documentation and the page's scope. Replaces the
// per-route hand-curated arrays so the scope → groups mapping lives in one
// place. Groups with no fields are omitted.
export function buildOptionGroupLayout(
    status: CheckerWithOptions | null | undefined,
    scope: CheckerPageScope,
    t: (key: string) => string,
): { editableGroups: OptionGroup[]; readOnlyGroups: KeyedOptionGroup[] } {
    const opts = status?.options ?? {};

    // Each entry pairs a group key with how the page should label it.
    const definitions: { key: "domainOpts" | "serviceOpts" | "userOpts" | "adminOpts" | "runOpts"; labelKey: string }[] = [
        { key: "domainOpts",  labelKey: "checkers.option-groups.domain-settings" },
        { key: "serviceOpts", labelKey: "checkers.option-groups.service-settings" },
        { key: "userOpts",    labelKey: "checkers.detail.configuration" },
        { key: "adminOpts",   labelKey: "checkers.detail.admin-options" },
        { key: "runOpts",     labelKey: "checkers.option-groups.checker-parameters" },
    ];

    // Per-scope group classification. Keys not listed are omitted entirely.
    const layout: Record<CheckerPageScope, { editable: string[]; readOnly: string[] }> = {
        admin:   { editable: ["userOpts", "adminOpts"],                            readOnly: ["domainOpts", "serviceOpts", "runOpts"] },
        domain:  { editable: ["domainOpts", "userOpts", "adminOpts"],              readOnly: ["serviceOpts", "runOpts"] },
        service: { editable: ["domainOpts", "serviceOpts", "userOpts", "adminOpts"], readOnly: ["runOpts"] },
    };

    const byKey = new Map(definitions.map((d) => [d.key, d]));
    const editableGroups: OptionGroup[] = [];
    const readOnlyGroups: KeyedOptionGroup[] = [];

    for (const key of layout[scope].editable) {
        const def = byKey.get(key as any);
        if (!def) continue;
        editableGroups.push({ label: t(def.labelKey), opts: opts[def.key] || [] });
    }
    for (const key of layout[scope].readOnly) {
        const def = byKey.get(key as any);
        if (!def) continue;
        readOnlyGroups.push({ key: def.key, label: t(def.labelKey), opts: opts[def.key] || [] });
    }

    return { editableGroups, readOnlyGroups };
}

// splitPositionalOptions separates the user-editable "current scope" options
// from less-specific scopes' options (presented as inherited placeholders).
//
// The controller appends an extra positional carrying resolved auto-fill
// values; it is not user-stored configuration and must not be treated as the
// current scope (otherwise the actual stored values get pushed into
// `inherited` and saving the form sends nil overrides). Callers pass the set
// of auto-fill option ids derived from the option documentation so we can
// drop that positional.
//
// isCurrentScope, when provided, identifies which positional belongs to the
// page's target scope. If no matching positional exists (nothing has been
// saved at this scope yet), current is empty and all stored values are
// inherited. Without this predicate the last stored entry is assumed current,
// which causes broader-scope values to masquerade as local overrides.
export function splitPositionalOptions(
    positionals: { options?: Record<string, unknown> | null; domainId?: unknown; serviceId?: unknown }[],
    autoFillKeys: Set<string> = new Set(),
    isCurrentScope: (p: { domainId?: unknown; serviceId?: unknown }) => boolean = () => false,
): { current: Record<string, unknown>; inherited: Record<string, unknown> } {
    const stored = positionals.filter((p) => {
        const keys = Object.keys(p.options ?? {});
        return keys.length === 0 || keys.some((k) => !autoFillKeys.has(k));
    });

    let currentIdx = -1;
    for (let i = stored.length - 1; i >= 0; i--) {
        if (isCurrentScope(stored[i])) {
            currentIdx = i;
            break;
        }
    }

    const current = currentIdx >= 0 ? (stored[currentIdx]?.options ?? {}) : {};
    const inherited: Record<string, unknown> = {};
    for (let i = 0; i < stored.length; i++) {
        if (i === currentIdx) continue;
        for (const [k, v] of Object.entries(stored[i].options ?? {})) {
            inherited[k] = v;
        }
    }
    return { current: { ...current }, inherited };
}

// collectAutoFillKeys returns the set of option ids flagged as auto-fill in
// the checker's option documentation (top-level groups and per-rule groups).
export function collectAutoFillKeys(status: CheckerWithOptions): Set<string> {
    const keys = new Set<string>();
    const addAll = (opts?: CheckerCheckerOptionDocumentation[]) => {
        opts?.forEach((o) => {
            if (o.id && o.autoFill) keys.add(o.id);
        });
    };
    addAll(status.options?.runOpts);
    addAll(status.options?.adminOpts);
    addAll(status.options?.userOpts);
    addAll(status.options?.domainOpts);
    status.rules?.forEach((r) => {
        addAll(r.options?.runOpts);
        addAll(r.options?.adminOpts);
        addAll(r.options?.userOpts);
        addAll(r.options?.domainOpts);
        addAll(r.options?.serviceOpts);
    });
    return keys;
}

export function downloadBlob(content: string, filename: string, mime: string) {
    const blob = new Blob([content], { type: mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
}

export function availabilityBadges(
    availability: { applyToDomain?: boolean; applyToZone?: boolean; applyToService?: boolean } | undefined,
    t?: (key: string) => string,
): { label: string; color: string }[] {
    if (!availability) return [];
    const badges = [];
    if (availability.applyToDomain) badges.push({ label: t ? t("checkers.availability.domain-level") : "Domain", color: "success" });
    if (availability.applyToZone) badges.push({ label: t ? t("checkers.availability.zone-level") : "Zone", color: "info" });
    if (availability.applyToService) badges.push({ label: t ? t("checkers.availability.service-level") : "Service", color: "primary" });
    return badges;
}

export function getOrphanedOptionKeys(
    optionValues: Record<string, unknown>,
    validOpts: { id?: string }[],
): string[] {
    const validOptIds = new Set(validOpts.map((opt) => opt.id));
    return Object.keys(optionValues).filter((key) => !validOptIds.has(key));
}

export function filterValidOptions(
    optionValues: Record<string, unknown>,
    validOpts: { id?: string }[],
): Record<string, unknown> {
    const validOptIds = new Set(validOpts.map((opt) => opt.id));
    const cleaned: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(optionValues)) {
        if (validOptIds.has(key)) {
            cleaned[key] = value;
        }
    }
    return cleaned;
}

export function formatCheckDate(date: string | Date | undefined): string {
    if (!date) return "";
    try {
        if (date instanceof Date) return date.toLocaleString();
        return new Date(date).toLocaleString();
    } catch {
        return String(date);
    }
}

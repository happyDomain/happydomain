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

// HappydnsExecutionStatus: 0=Pending, 1=Running, 2=Done, 3=Failed

export function getExecutionStatusColor(status: HappydnsExecutionStatus | undefined): string {
    switch (status) {
        case 0: return "secondary";
        case 1: return "primary";
        case 2: return "success";
        case 3: return "danger";
        default: return "secondary";
    }
}

export function getExecutionStatusI18nKey(status: HappydnsExecutionStatus | undefined): string {
    switch (status) {
        case 0: return "checkers.execution.status.pending";
        case 1: return "checkers.execution.status.running";
        case 2: return "checkers.execution.status.done";
        case 3: return "checkers.execution.status.failed";
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
        ...(status.rules || []).flatMap((r) => [
            ...(r.options?.runOpts || []),
            ...(r.options?.adminOpts || []),
            ...(r.options?.userOpts || []),
            ...(r.options?.domainOpts || []),
        ]),
    ];
}

export function splitPositionalOptions(
    positionals: { options?: Record<string, unknown> | null }[],
): { current: Record<string, unknown>; inherited: Record<string, unknown> } {
    const current =
        positionals.length > 0 ? (positionals[positionals.length - 1]?.options ?? {}) : {};
    const inherited: Record<string, unknown> = {};
    for (let i = 0; i < positionals.length - 1; i++) {
        for (const [k, v] of Object.entries(positionals[i].options ?? {})) {
            inherited[k] = v;
        }
    }
    return { current: { ...current }, inherited };
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

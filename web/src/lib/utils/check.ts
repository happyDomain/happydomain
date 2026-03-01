import type { HappydnsCheckResultStatus } from "$lib/api-base/types.gen";
import { CheckResultStatus } from "$lib/model/checker";

export function getStatusColor(
    status: CheckResultStatus | HappydnsCheckResultStatus | undefined,
): string {
    switch (status) {
        case CheckResultStatus.OK:
            return "success";
        case CheckResultStatus.Info:
            return "info";
        case CheckResultStatus.Warn:
            return "warning";
        case CheckResultStatus.Crit:
            return "danger";
        default:
            return "secondary";
    }
}

export function getStatusKey(
    status: CheckResultStatus | HappydnsCheckResultStatus | undefined,
): string {
    switch (status) {
        case CheckResultStatus.OK:
            return "checkers.status.ok";
        case CheckResultStatus.Info:
            return "checkers.status.info";
        case CheckResultStatus.Warn:
            return "checkers.status.warning";
        case CheckResultStatus.Crit:
            return "checkers.status.error";
        default:
            return "checkers.status.unknown";
    }
}

export function getStatusIcon(status: CheckResultStatus | HappydnsCheckResultStatus | undefined): string {
    switch (status) {
        case CheckResultStatus.OK:
            return "check-circle-fill";
        case CheckResultStatus.Info:
            return "info-circle-fill";
        case CheckResultStatus.Warn:
            return "exclamation-triangle-fill";
        case CheckResultStatus.Crit:
            return "exclamation-octagon-fill";
        default:
            return "question-circle-fill";
    }
}

export function formatDuration(duration: number | undefined, t: (k: string) => string): string {
    if (!duration) return t("checkers.na");
    const seconds = duration / 1000000000;
    if (seconds < 1) return `${(seconds * 1000).toFixed(0)} ${t("checkers.result.milliseconds")}`;
    return `${seconds.toFixed(2)} ${t("checkers.result.seconds")}`;
}

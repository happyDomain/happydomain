import type { HappydnsCheckResultStatus } from "$lib/api-base/types.gen";
import { CheckResultStatus } from "$lib/model/check";

export function getStatusColor(status: CheckResultStatus | HappydnsCheckResultStatus | undefined): string {
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

export function getStatusKey(status: CheckResultStatus | HappydnsCheckResultStatus | undefined): string {
    switch (status) {
        case CheckResultStatus.OK:
            return "checks.status.ok";
        case CheckResultStatus.Info:
            return "checks.status.info";
        case CheckResultStatus.Warn:
            return "checks.status.warning";
        case CheckResultStatus.Crit:
            return "checks.status.error";
        default:
            return "checks.status.unknown";
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
    if (!duration) return t("checks.na");
    const seconds = duration / 1000000000;
    if (seconds < 1) return `${(seconds * 1000).toFixed(0)} ${t("checks.result.milliseconds")}`;
    return `${seconds.toFixed(2)} ${t("checks.result.seconds")}`;
}

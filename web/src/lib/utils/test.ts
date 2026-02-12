import { PluginResultStatus } from "$lib/model/test";

export function getStatusColor(status: PluginResultStatus): string {
    switch (status) {
        case PluginResultStatus.OK:
            return "success";
        case PluginResultStatus.Info:
            return "info";
        case PluginResultStatus.Warn:
            return "warning";
        case PluginResultStatus.KO:
            return "danger";
        default:
            return "secondary";
    }
}

export function getStatusKey(status: PluginResultStatus): string {
    switch (status) {
        case PluginResultStatus.OK:
            return "tests.status.ok";
        case PluginResultStatus.Info:
            return "tests.status.info";
        case PluginResultStatus.Warn:
            return "tests.status.warning";
        case PluginResultStatus.KO:
            return "tests.status.error";
        default:
            return "tests.status.unknown";
    }
}

export function formatDuration(duration: number | undefined, t: (k: string) => string): string {
    if (!duration) return t("tests.na");
    const seconds = duration / 1000000000;
    if (seconds < 1)
        return `${(seconds * 1000).toFixed(0)} ${t("tests.result.milliseconds")}`;
    return `${seconds.toFixed(2)} ${t("tests.result.seconds")}`;
}

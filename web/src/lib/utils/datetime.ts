/**
 * Convert ISO 8601 datetime string to datetime-local format (YYYY-MM-DDTHH:mm)
 * @param isoString ISO 8601 datetime string
 * @returns Datetime-local format string, or empty string if invalid
 */
export function toDatetimeLocal(isoString: string | null | undefined): string {
    if (!isoString) return "";
    try {
        const date = new Date(isoString);
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, "0");
        const day = String(date.getDate()).padStart(2, "0");
        const hours = String(date.getHours()).padStart(2, "0");
        const minutes = String(date.getMinutes()).padStart(2, "0");
        return `${year}-${month}-${day}T${hours}:${minutes}`;
    } catch (e) {
        return "";
    }
}

/**
 * Convert datetime-local format back to ISO 8601 string
 * @param datetimeLocal Datetime-local format string (YYYY-MM-DDTHH:mm)
 * @returns ISO 8601 datetime string, or null if invalid
 */
export function fromDatetimeLocal(datetimeLocal: string): string | null {
    if (!datetimeLocal) return null;
    try {
        return new Date(datetimeLocal).toISOString();
    } catch (e) {
        return null;
    }
}

/**
 * Format a date string for display in check UI
 * @param dateString ISO date string or undefined
 * @param style Display style: "short", "medium", or "long"
 * @param t i18n translation function
 * @returns Formatted date string, or $t("checks.never") if undefined/invalid
 */
export function formatCheckDate(
    dateString: string | undefined,
    style: "short" | "medium" | "long",
    t: (k: string) => string,
): string {
    if (!dateString) return t("checks.never");
    const d = new Date(dateString);
    if (isNaN(d.getTime())) return t("checks.never");
    return new Intl.DateTimeFormat(undefined, {
        dateStyle: style,
        timeStyle: "short",
    }).format(d);
}

/**
 * Format a date string as a relative time (e.g. "in 3h 20m" or "5m ago")
 * @param dateString ISO date string or undefined
 * @param t i18n translation function
 * @returns Relative time string, or empty string if undefined/invalid
 */
export function formatRelative(dateString: string | undefined, t: (k: string, params?: Record<string, string>) => string): string {
    if (!dateString) return "";
    const d = new Date(dateString);
    if (isNaN(d.getTime())) return "";
    const now = new Date();
    const diffMs = d.getTime() - now.getTime();
    const absDiffMs = Math.abs(diffMs);

    if (absDiffMs < 60_000)
        return diffMs > 0
            ? t("checks.relative.in-less-than-a-minute")
            : t("checks.relative.just-now");

    const minutes = Math.floor(absDiffMs / 60_000);
    const hours = Math.floor(absDiffMs / 3_600_000);
    const days = Math.floor(absDiffMs / 86_400_000);

    let label: string;
    if (days > 0) {
        label = `${days}d ${hours % 24}h`;
    } else if (hours > 0) {
        label = `${hours}h ${minutes % 60}m`;
    } else {
        label = `${minutes}m`;
    }

    return diffMs > 0
        ? t("checks.relative.in", { label })
        : t("checks.relative.ago", { label });
}

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
 * Format a Go time.Duration (nanoseconds) into a human-readable string.
 * @param ns Duration in nanoseconds
 * @returns Human-readable string such as "30s", "5m 30s", "2h 15m", "3d 4h"
 */
export function formatDuration(ns: number | undefined): string {
    if (ns == null) return "—";
    const totalSeconds = Math.floor(ns / 1e9);
    if (totalSeconds < 60) return `${totalSeconds}s`;
    const minutes = Math.floor(totalSeconds / 60);
    const remainSeconds = totalSeconds % 60;
    if (minutes < 60) {
        return remainSeconds > 0 ? `${minutes}m ${remainSeconds}s` : `${minutes}m`;
    }
    const hours = Math.floor(minutes / 60);
    const remainMinutes = minutes % 60;
    if (hours < 24) {
        return remainMinutes > 0 ? `${hours}h ${remainMinutes}m` : `${hours}h`;
    }
    const days = Math.floor(hours / 24);
    const remainHours = hours % 24;
    return remainHours > 0 ? `${days}d ${remainHours}h` : `${days}d`;
}

/**
 * Format a date string to relative time (e.g. "in 5m", "3h ago").
 * @param dateStr ISO 8601 date string
 * @returns Human-readable relative time string
 */
export function formatRelative(dateStr: string | undefined): string {
    if (!dateStr) return "—";
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = date.getTime() - now.getTime();
    const absSeconds = Math.floor(Math.abs(diffMs) / 1000);
    const absMinutes = Math.floor(absSeconds / 60);
    const absHours = Math.floor(absMinutes / 60);
    const absDays = Math.floor(absHours / 24);

    let rel: string;
    if (absSeconds < 60) rel = `${absSeconds}s`;
    else if (absMinutes < 60) rel = `${absMinutes}m`;
    else if (absHours < 24) rel = `${absHours}h`;
    else rel = `${absDays}d`;

    return diffMs >= 0 ? `in ${rel}` : `${rel} ago`;
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
 * Format a countdown timer string from a target date.
 * Displays the remaining time in a human-readable format (e.g., "2d 5h", "3h 20m").
 * @param date Target date to count down to
 * @returns Formatted countdown string (e.g., "2d 5h", "30m", "45s"), or "0m" if date is in the past
 */
export function formatCountdown(date: Date): string {
    const diff = date.getTime() - Date.now();
    if (diff <= 0) return "0m";

    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) {
        return `${days}d ${hours % 24}h`;
    } else if (hours > 0) {
        return `${hours}h ${minutes % 60}m`;
    } else if (minutes > 9) {
        return `${minutes}m`;
    } else if (minutes > 0) {
        return `${minutes}m ${seconds % 60}s`;
    } else {
        return `${seconds}s`;
    }
}

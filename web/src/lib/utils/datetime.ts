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

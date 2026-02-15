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

/**
 * Format a byte count into a human-readable IEC string (KiB, MiB, ...).
 * @param n Number of bytes
 * @returns Human-readable string such as "1.4 MiB", or "—" if undefined
 */
export function formatBytes(n: number | undefined): string {
    if (n === undefined || !Number.isFinite(n)) return "—";
    const units = ["B", "KiB", "MiB", "GiB", "TiB"];
    let i = 0;
    let v = n;
    while (v >= 1024 && i < units.length - 1) {
        v /= 1024;
        i++;
    }
    return `${v.toFixed(v >= 100 || i === 0 ? 0 : 1)} ${units[i]}`;
}

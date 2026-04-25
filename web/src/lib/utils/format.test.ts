import { describe, it, expect } from "vitest";
import { formatBytes } from "./format";

describe("formatBytes", () => {
    it("returns em dash for undefined", () => {
        expect(formatBytes(undefined)).toBe("—");
    });

    it("returns em dash for NaN", () => {
        expect(formatBytes(Number.NaN)).toBe("—");
    });

    it("returns em dash for Infinity", () => {
        expect(formatBytes(Number.POSITIVE_INFINITY)).toBe("—");
    });

    it("formats zero bytes without decimals", () => {
        expect(formatBytes(0)).toBe("0 B");
    });

    it("formats small byte counts as integer B", () => {
        expect(formatBytes(1)).toBe("1 B");
        expect(formatBytes(512)).toBe("512 B");
        expect(formatBytes(1023)).toBe("1023 B");
    });

    it("crosses to KiB at 1024", () => {
        expect(formatBytes(1024)).toBe("1.0 KiB");
    });

    it("formats KiB with one decimal under 100", () => {
        expect(formatBytes(1536)).toBe("1.5 KiB");
    });

    it("drops decimals once value is at least 100 in its unit", () => {
        expect(formatBytes(102 * 1024)).toBe("102 KiB");
    });

    it("uses MiB once large enough", () => {
        expect(formatBytes(1024 * 1024)).toBe("1.0 MiB");
        expect(formatBytes(2.5 * 1024 * 1024)).toBe("2.5 MiB");
    });

    it("uses GiB once large enough", () => {
        expect(formatBytes(1024 ** 3)).toBe("1.0 GiB");
    });

    it("uses TiB once large enough", () => {
        expect(formatBytes(1024 ** 4)).toBe("1.0 TiB");
    });

    it("does not exceed TiB even for petabyte-scale inputs", () => {
        // The unit list tops out at TiB; very large numbers are reported in TiB.
        const result = formatBytes(1024 ** 5);
        expect(result.endsWith(" TiB")).toBe(true);
    });
});

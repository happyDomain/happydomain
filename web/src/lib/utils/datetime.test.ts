import { describe, it, expect, vi, afterEach } from "vitest";
import { formatDuration, formatRelative } from "./datetime";

describe("formatDuration", () => {
    it("returns em dash for undefined", () => {
        expect(formatDuration(undefined)).toBe("—");
    });

    it("returns 0s for zero nanoseconds", () => {
        expect(formatDuration(0)).toBe("0s");
    });

    it("formats seconds only", () => {
        expect(formatDuration(30e9)).toBe("30s");
        expect(formatDuration(1e9)).toBe("1s");
        expect(formatDuration(59e9)).toBe("59s");
    });

    it("formats exact minutes", () => {
        expect(formatDuration(60e9)).toBe("1m");
        expect(formatDuration(300e9)).toBe("5m");
    });

    it("formats minutes with remaining seconds", () => {
        expect(formatDuration(90e9)).toBe("1m 30s");
        expect(formatDuration(61e9)).toBe("1m 1s");
    });

    it("formats exact hours", () => {
        expect(formatDuration(3600e9)).toBe("1h");
        expect(formatDuration(7200e9)).toBe("2h");
    });

    it("formats hours with remaining minutes", () => {
        expect(formatDuration(5400e9)).toBe("1h 30m");
        expect(formatDuration(3660e9)).toBe("1h 1m");
    });

    it("formats exact days", () => {
        expect(formatDuration(86400e9)).toBe("1d");
        expect(formatDuration(172800e9)).toBe("2d");
    });

    it("formats days with remaining hours", () => {
        expect(formatDuration(90000e9)).toBe("1d 1h");
        expect(formatDuration(129600e9)).toBe("1d 12h");
    });

    it("truncates sub-second precision", () => {
        expect(formatDuration(1.5e9)).toBe("1s");
        expect(formatDuration(999999999)).toBe("0s");
    });
});

describe("formatRelative", () => {
    afterEach(() => {
        vi.useRealTimers();
    });

    it("returns em dash for undefined", () => {
        expect(formatRelative(undefined)).toBe("—");
    });

    it("returns em dash for empty string", () => {
        expect(formatRelative("")).toBe("—");
    });

    it("formats seconds in the past", () => {
        vi.useFakeTimers({ now: new Date("2026-01-01T12:00:30Z") });
        expect(formatRelative("2026-01-01T12:00:00Z")).toBe("30s ago");
    });

    it("formats seconds in the future", () => {
        vi.useFakeTimers({ now: new Date("2026-01-01T12:00:00Z") });
        expect(formatRelative("2026-01-01T12:00:45Z")).toBe("in 45s");
    });

    it("formats minutes in the past", () => {
        vi.useFakeTimers({ now: new Date("2026-01-01T12:05:00Z") });
        expect(formatRelative("2026-01-01T12:00:00Z")).toBe("5m ago");
    });

    it("formats minutes in the future", () => {
        vi.useFakeTimers({ now: new Date("2026-01-01T12:00:00Z") });
        expect(formatRelative("2026-01-01T12:10:00Z")).toBe("in 10m");
    });

    it("formats hours in the past", () => {
        vi.useFakeTimers({ now: new Date("2026-01-01T15:00:00Z") });
        expect(formatRelative("2026-01-01T12:00:00Z")).toBe("3h ago");
    });

    it("formats hours in the future", () => {
        vi.useFakeTimers({ now: new Date("2026-01-01T12:00:00Z") });
        expect(formatRelative("2026-01-01T18:00:00Z")).toBe("in 6h");
    });

    it("formats days in the past", () => {
        vi.useFakeTimers({ now: new Date("2026-01-03T12:00:00Z") });
        expect(formatRelative("2026-01-01T12:00:00Z")).toBe("2d ago");
    });

    it("formats days in the future", () => {
        vi.useFakeTimers({ now: new Date("2026-01-01T12:00:00Z") });
        expect(formatRelative("2026-01-04T12:00:00Z")).toBe("in 3d");
    });

    it("uses floor not round for thresholds", () => {
        // 50 seconds should still show as seconds, not round up to 1m
        vi.useFakeTimers({ now: new Date("2026-01-01T12:00:50Z") });
        expect(formatRelative("2026-01-01T12:00:00Z")).toBe("50s ago");
    });
});

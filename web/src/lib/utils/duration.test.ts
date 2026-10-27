import { describe, expect, it } from "vitest";

import { formatBindDuration, formatHumanDuration, parseBindDuration } from "./duration";

describe("parseBindDuration", () => {
    it("reads a bare number as seconds", () => {
        expect(parseBindDuration("604800")).toBe(604800);
        expect(parseBindDuration("0")).toBe(0);
    });

    it("reads every BIND unit", () => {
        expect(parseBindDuration("30s")).toBe(30);
        expect(parseBindDuration("5m")).toBe(300);
        expect(parseBindDuration("2h")).toBe(7200);
        expect(parseBindDuration("1d")).toBe(86400);
        expect(parseBindDuration("1w")).toBe(604800);
    });

    it("adds up combined units, whatever the case and spacing", () => {
        expect(parseBindDuration("1w2d")).toBe(777600);
        expect(parseBindDuration(" 1H 30M ")).toBe(5400);
        expect(parseBindDuration("1 h 30 m")).toBe(5400);
    });

    it("rejects what is not a duration", () => {
        expect(parseBindDuration("")).toBeNull();
        expect(parseBindDuration("abc")).toBeNull();
        expect(parseBindDuration("1y")).toBeNull();
        expect(parseBindDuration("h")).toBeNull();
        expect(parseBindDuration("1h30")).toBeNull();
        expect(parseBindDuration("-5m")).toBeNull();
        expect(parseBindDuration("1.5h")).toBeNull();
    });
});

describe("formatBindDuration", () => {
    it("picks the largest units", () => {
        expect(formatBindDuration(604800)).toBe("1w");
        expect(formatBindDuration(86400)).toBe("1d");
        expect(formatBindDuration(5400)).toBe("1h30m");
        expect(formatBindDuration(45)).toBe("45s");
        expect(formatBindDuration(1209600 + 3600 + 1)).toBe("2w1h1s");
    });

    it("writes zero as a bare number", () => {
        expect(formatBindDuration(0)).toBe("0");
    });

    it("round-trips through the parser", () => {
        for (const s of [1, 59, 61, 3599, 3601, 86399, 90061, 2419200]) {
            expect(parseBindDuration(formatBindDuration(s))).toBe(s);
        }
    });
});

describe("formatHumanDuration", () => {
    it("spells out each unit", () => {
        expect(formatHumanDuration(604800, "en")).toBe("1 week");
        expect(formatHumanDuration(5400, "en")).toBe("1 hour, 30 minutes");
        expect(formatHumanDuration(0, "en")).toBe("0 seconds");
    });

    it("follows the locale", () => {
        expect(formatHumanDuration(7200, "fr")).toMatch(/^2\s+heures$/);
    });
});

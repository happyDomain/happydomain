import { describe, it, expect } from "vitest";
import {
    getStatusColor,
    getStatusIcon,
    getStatusI18nKey,
    getExecutionStatusColor,
    getExecutionStatusI18nKey,
    withInheritedPlaceholders,
    collectAllOptionDocs,
    formatCheckDate,
    getOrphanedOptionKeys,
    filterValidOptions,
    StatusOK,
    StatusInfo,
    StatusUnknown,
    StatusWarn,
    StatusCrit,
    StatusError,
} from "./checkers";
import type { CheckerCheckerOptionDocumentation } from "$lib/api-base/types.gen";

describe("getStatusColor", () => {
    it("maps each status to the correct color", () => {
        expect(getStatusColor(StatusOK)).toBe("success");
        expect(getStatusColor(StatusInfo)).toBe("info");
        expect(getStatusColor(StatusUnknown)).toBe("secondary");
        expect(getStatusColor(StatusWarn)).toBe("warning");
        expect(getStatusColor(StatusCrit)).toBe("danger");
        expect(getStatusColor(StatusError)).toBe("danger");
    });

    it("returns secondary for undefined", () => {
        expect(getStatusColor(undefined)).toBe("secondary");
    });
});

describe("getStatusI18nKey", () => {
    it("maps each status to the correct i18n key", () => {
        expect(getStatusI18nKey(StatusOK)).toBe("checkers.status.ok");
        expect(getStatusI18nKey(StatusInfo)).toBe("checkers.status.info");
        expect(getStatusI18nKey(StatusUnknown)).toBe("checkers.status.unknown");
        expect(getStatusI18nKey(StatusWarn)).toBe("checkers.status.warning");
        expect(getStatusI18nKey(StatusCrit)).toBe("checkers.status.critical");
        expect(getStatusI18nKey(StatusError)).toBe("checkers.status.error");
    });

    it("returns not-run for undefined", () => {
        expect(getStatusI18nKey(undefined)).toBe("checkers.status.not-run");
    });
});

describe("getStatusIcon", () => {
    it("maps each status to the correct icon", () => {
        expect(getStatusIcon(StatusOK)).toBe("check-circle-fill");
        expect(getStatusIcon(StatusInfo)).toBe("info-circle-fill");
        expect(getStatusIcon(StatusWarn)).toBe("exclamation-triangle-fill");
        expect(getStatusIcon(StatusCrit)).toBe("exclamation-octagon-fill");
        expect(getStatusIcon(StatusError)).toBe("exclamation-octagon-fill");
    });

    it("returns question-circle-fill for undefined", () => {
        expect(getStatusIcon(undefined)).toBe("question-circle-fill");
    });
});

describe("getExecutionStatusColor", () => {
    it("maps each execution status to the correct color", () => {
        expect(getExecutionStatusColor(0)).toBe("secondary");
        expect(getExecutionStatusColor(1)).toBe("primary");
        expect(getExecutionStatusColor(2)).toBe("success");
        expect(getExecutionStatusColor(3)).toBe("danger");
        expect(getExecutionStatusColor(4)).toBe("warning");
    });

    it("returns secondary for undefined", () => {
        expect(getExecutionStatusColor(undefined)).toBe("secondary");
    });
});

describe("getExecutionStatusI18nKey", () => {
    it("maps each execution status to the correct i18n key", () => {
        expect(getExecutionStatusI18nKey(0)).toBe("checkers.execution.status.pending");
        expect(getExecutionStatusI18nKey(1)).toBe("checkers.execution.status.running");
        expect(getExecutionStatusI18nKey(2)).toBe("checkers.execution.status.done");
        expect(getExecutionStatusI18nKey(3)).toBe("checkers.execution.status.failed");
        expect(getExecutionStatusI18nKey(4)).toBe("checkers.execution.status.rate-limited");
    });

    it("returns unknown for undefined", () => {
        expect(getExecutionStatusI18nKey(undefined)).toBe("checkers.execution.status.unknown");
    });
});

describe("withInheritedPlaceholders", () => {
    const makeOpt = (id: string, placeholder?: string): CheckerCheckerOptionDocumentation => ({
        id,
        type: "string",
        ...(placeholder !== undefined ? { placeholder } : {}),
    });

    it("adds placeholder from inherited when option value is undefined", () => {
        const opts = [makeOpt("host")];
        const result = withInheritedPlaceholders(opts, {}, { host: "example.com" });
        expect(result[0].placeholder).toBe("example.com");
    });

    it("does not override when option value is already set", () => {
        const opts = [makeOpt("host")];
        const result = withInheritedPlaceholders(opts, { host: "mine.com" }, { host: "example.com" });
        expect(result[0].placeholder).toBeUndefined();
    });

    it("does not add placeholder when inherited value is undefined", () => {
        const opts = [makeOpt("host")];
        const result = withInheritedPlaceholders(opts, {}, {});
        expect(result[0].placeholder).toBeUndefined();
    });

    it("returns original opt when id is empty", () => {
        const opts: CheckerCheckerOptionDocumentation[] = [{ id: "", type: "string" }];
        const result = withInheritedPlaceholders(opts, {}, { host: "example.com" });
        expect(result[0]).toEqual({ id: "", type: "string" });
    });

    it("handles multiple options", () => {
        const opts = [makeOpt("host"), makeOpt("port")];
        const result = withInheritedPlaceholders(
            opts,
            { port: "443" },
            { host: "example.com", port: "80" },
        );
        expect(result[0].placeholder).toBe("example.com");
        expect(result[1].placeholder).toBeUndefined();
    });
});

describe("collectAllOptionDocs", () => {
    const opt = (id: string, noOverride?: boolean): CheckerCheckerOptionDocumentation => ({
        id,
        type: "string",
        ...(noOverride !== undefined ? { noOverride } : {}),
    });

    it("returns empty array for empty status", () => {
        expect(collectAllOptionDocs({})).toEqual([]);
    });

    it("collects from all option groups", () => {
        const result = collectAllOptionDocs({
            options: {
                runOpts: [opt("a")],
                adminOpts: [opt("b")],
                userOpts: [opt("c")],
                domainOpts: [opt("d")],
            },
        });
        expect(result.map((o) => o.id)).toEqual(["a", "b", "c", "d"]);
    });

    it("collects from rules", () => {
        const result = collectAllOptionDocs({
            rules: [
                { options: { runOpts: [opt("r1")] } },
                { options: { userOpts: [opt("r2")] } },
            ],
        });
        expect(result.map((o) => o.id)).toEqual(["r1", "r2"]);
    });

    it("combines top-level and rule options", () => {
        const result = collectAllOptionDocs({
            options: { runOpts: [opt("top")] },
            rules: [{ options: { runOpts: [opt("rule")] } }],
        });
        expect(result.map((o) => o.id)).toEqual(["top", "rule"]);
    });

    it("filters out noOverride options", () => {
        const result = collectAllOptionDocs({
            options: {
                runOpts: [opt("keep", false), opt("skip", true)],
            },
        });
        expect(result.map((o) => o.id)).toEqual(["keep"]);
    });

    it("handles rules with missing options", () => {
        const result = collectAllOptionDocs({
            rules: [{ options: undefined }],
        });
        expect(result).toEqual([]);
    });
});

describe("formatCheckDate", () => {
    it("returns empty string for undefined", () => {
        expect(formatCheckDate(undefined)).toBe("");
    });

    it("returns empty string for empty string", () => {
        expect(formatCheckDate("")).toBe("");
    });

    it("formats a valid ISO string", () => {
        const result = formatCheckDate("2026-01-01T12:00:00Z");
        expect(result).toBeTruthy();
        expect(result).not.toBe("");
    });

    it("formats a Date object", () => {
        const result = formatCheckDate(new Date("2026-01-01T12:00:00Z"));
        expect(result).toBeTruthy();
        expect(result).not.toBe("");
    });
});

describe("getOrphanedOptionKeys", () => {
    it("returns keys not in validOpts", () => {
        const result = getOrphanedOptionKeys(
            { a: 1, b: 2, c: 3 },
            [{ id: "a" }, { id: "c" }],
        );
        expect(result).toEqual(["b"]);
    });

    it("returns empty when all keys are valid", () => {
        const result = getOrphanedOptionKeys(
            { a: 1, b: 2 },
            [{ id: "a" }, { id: "b" }],
        );
        expect(result).toEqual([]);
    });

    it("returns all keys when validOpts is empty", () => {
        const result = getOrphanedOptionKeys({ x: 1, y: 2 }, []);
        expect(result).toEqual(["x", "y"]);
    });

    it("returns empty when optionValues is empty", () => {
        const result = getOrphanedOptionKeys({}, [{ id: "a" }]);
        expect(result).toEqual([]);
    });
});

describe("filterValidOptions", () => {
    it("keeps only keys present in validOpts", () => {
        const result = filterValidOptions(
            { a: 1, b: 2, c: 3 },
            [{ id: "a" }, { id: "c" }],
        );
        expect(result).toEqual({ a: 1, c: 3 });
    });

    it("returns empty object when no keys are valid", () => {
        const result = filterValidOptions({ a: 1 }, [{ id: "z" }]);
        expect(result).toEqual({});
    });

    it("returns empty object for empty input", () => {
        const result = filterValidOptions({}, [{ id: "a" }]);
        expect(result).toEqual({});
    });
});

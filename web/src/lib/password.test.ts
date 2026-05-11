import { describe, it, expect } from "vitest";
import { checkWeakPassword, checkPasswordConfirmation } from "./password";

describe("checkWeakPassword", () => {
    it("returns undefined for empty input (no signal yet)", () => {
        expect(checkWeakPassword("")).toBeUndefined();
    });

    it("rejects passwords shorter than 8 chars", () => {
        expect(checkWeakPassword("Aa1!aaa")).toBe(false);
    });

    it("rejects passwords missing an uppercase letter", () => {
        expect(checkWeakPassword("aaaa1234!")).toBe(false);
    });

    it("rejects passwords missing a lowercase letter", () => {
        expect(checkWeakPassword("AAAA1234!")).toBe(false);
    });

    it("rejects passwords missing a digit", () => {
        expect(checkWeakPassword("Abcdefgh!")).toBe(false);
    });

    it("rejects 8-10 char passwords with no special character", () => {
        expect(checkWeakPassword("Abcdefg1")).toBe(false);
        expect(checkWeakPassword("Abcdefgh12")).toBe(false);
    });

    it("accepts 11+ char passwords without specials when other classes are present", () => {
        expect(checkWeakPassword("Abcdefghi12")).toBe(true);
    });

    it("accepts strong passwords with a special character at length 8", () => {
        expect(checkWeakPassword("Abcdef1!")).toBe(true);
    });

    it("treats _ as a special character (regex \\W matches non-word)", () => {
        // \W is [^A-Za-z0-9_], so underscore is NOT special — must rely on length.
        expect(checkWeakPassword("Abcdefg_1")).toBe(false);
    });
});

describe("checkPasswordConfirmation", () => {
    it("returns undefined when confirmation is empty", () => {
        expect(checkPasswordConfirmation("hunter2", "")).toBeUndefined();
    });

    it("returns true when both match", () => {
        expect(checkPasswordConfirmation("hunter2", "hunter2")).toBe(true);
    });

    it("returns false when confirmation differs", () => {
        expect(checkPasswordConfirmation("hunter2", "hunter3")).toBe(false);
    });

    it("is case-sensitive", () => {
        expect(checkPasswordConfirmation("Hunter2", "hunter2")).toBe(false);
    });

    it("treats both empty as undefined (no signal)", () => {
        expect(checkPasswordConfirmation("", "")).toBeUndefined();
    });
});

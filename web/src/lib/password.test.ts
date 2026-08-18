// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { describe, it, expect } from "vitest";
import { checkWeakPassword, checkPasswordConfirmation } from "./password";

describe("checkWeakPassword", () => {
    it("returns undefined for an empty password", () => {
        expect(checkWeakPassword("")).toBeUndefined();
    });

    it("returns true for a strong password with special char", () => {
        expect(checkWeakPassword("Abcdefg1!")).toBe(true);
    });

    it("returns true for a strong password without special char but long enough", () => {
        expect(checkWeakPassword("Abcdefghij1")).toBe(true);
    });

    it("returns false when shorter than 11 chars and missing a special char", () => {
        expect(checkWeakPassword("Abcdefg1")).toBe(false);
    });

    it("returns false when shorter than 8 chars", () => {
        expect(checkWeakPassword("Ab1!")).toBe(false);
    });

    it("returns false when missing an uppercase letter", () => {
        expect(checkWeakPassword("abcdefg1!")).toBe(false);
    });

    it("returns false when missing a lowercase letter", () => {
        expect(checkWeakPassword("ABCDEFG1!")).toBe(false);
    });

    it("returns false when missing a digit", () => {
        expect(checkWeakPassword("Abcdefgh!")).toBe(false);
    });

    it("returns false for only letters, regardless of length", () => {
        expect(checkWeakPassword("Abcdefghijklmno")).toBe(false);
    });

    it("returns false for only digits", () => {
        expect(checkWeakPassword("12345678901")).toBe(false);
    });

    it("accepts exactly 8 characters with a special char", () => {
        expect(checkWeakPassword("Abcdefg1!")).toBe(true);
    });

    it("rejects 7 characters even with all classes and a special char", () => {
        expect(checkWeakPassword("Abcde1!")).toBe(false);
    });

    it("accepts exactly 11 characters without a special char", () => {
        expect(checkWeakPassword("Abcdefghij1")).toBe(true);
    });

    it("rejects 10 characters without a special char", () => {
        expect(checkWeakPassword("Abcdefghi1")).toBe(false);
    });

    it("treats underscore as a word character, not special", () => {
        // \W does not match underscore, so length must reach 11 to pass
        expect(checkWeakPassword("Abcdefg_1")).toBe(false);
        expect(checkWeakPassword("Abcdefghi_1")).toBe(true);
    });

    it("accepts unicode/space as a special character", () => {
        expect(checkWeakPassword("Abcdefg 1")).toBe(true);
    });

    it("handles a whitespace-only password as non-empty", () => {
        expect(checkWeakPassword("        ")).toBe(false);
    });
});

describe("checkPasswordConfirmation", () => {
    it("returns undefined when confirmation is empty", () => {
        expect(checkPasswordConfirmation("password", "")).toBeUndefined();
    });

    it("returns undefined when both are empty", () => {
        expect(checkPasswordConfirmation("", "")).toBeUndefined();
    });

    it("returns true when password and confirmation match", () => {
        expect(checkPasswordConfirmation("password", "password")).toBe(true);
    });

    it("returns false when password and confirmation differ", () => {
        expect(checkPasswordConfirmation("password", "differen")).toBe(false);
    });

    it("returns false when confirmation is non-empty but password is empty", () => {
        expect(checkPasswordConfirmation("", "password")).toBe(false);
    });

    it("is case-sensitive", () => {
        expect(checkPasswordConfirmation("Password", "password")).toBe(false);
    });
});

// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
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
import { parseBIMI, stringifyBIMI } from "./bimi";

describe("parseBIMI", () => {
    it("parses a minimal BIMI record", () => {
        expect(parseBIMI("v=BIMI1;l=https://example.com/logo.svg")).toEqual({
            v: "BIMI1",
            l: "https://example.com/logo.svg",
            a: undefined,
            e: undefined,
        });
    });

    it("parses a BIMI record with all fields", () => {
        expect(
            parseBIMI(
                "v=BIMI1; l=https://example.com/logo.svg; a=https://example.com/vmc.pem; e=https://example.com/evidence",
            ),
        ).toEqual({
            v: "BIMI1",
            l: "https://example.com/logo.svg",
            a: "https://example.com/vmc.pem",
            e: "https://example.com/evidence",
        });
    });

    it("tolerates spaces after semicolons", () => {
        expect(parseBIMI("v=BIMI1; l=https://example.com/logo.svg")).toEqual({
            v: "BIMI1",
            l: "https://example.com/logo.svg",
            a: undefined,
            e: undefined,
        });
    });
});

describe("stringifyBIMI", () => {
    it("uses BIMI1 as default version", () => {
        expect(stringifyBIMI({ l: "https://example.com/logo.svg" })).toBe(
            "v=BIMI1;l=https://example.com/logo.svg",
        );
    });

    it("respects an explicit version", () => {
        expect(stringifyBIMI({ v: "BIMI2", l: "https://example.com/logo.svg" })).toBe(
            "v=BIMI2;l=https://example.com/logo.svg",
        );
    });

    it("includes a and e when provided", () => {
        expect(
            stringifyBIMI({
                v: "BIMI1",
                l: "https://example.com/logo.svg",
                a: "https://example.com/vmc.pem",
                e: "https://example.com/evidence",
            }),
        ).toBe(
            "v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc.pem;e=https://example.com/evidence",
        );
    });

    it("preserves the separator style of the existing TXT", () => {
        expect(
            stringifyBIMI(
                { v: "BIMI1", l: "https://example.com/logo.svg" },
                "v=BIMI1; l=https://example.com/logo.svg",
            ),
        ).toBe("v=BIMI1; l=https://example.com/logo.svg");
    });

    it("omits empty optional fields", () => {
        const out = stringifyBIMI({ v: "BIMI1", l: "https://example.com/logo.svg" });
        expect(out).not.toContain("a=");
        expect(out).not.toContain("e=");
    });
});

describe("parseBIMI and stringifyBIMI roundtrip", () => {
    it("preserves all fields through a full roundtrip", () => {
        const original =
            "v=BIMI1;l=https://example.com/logo.svg;a=https://example.com/vmc.pem;e=https://example.com/evidence";
        const parsed = parseBIMI(original);
        const stringified = stringifyBIMI(parsed, original);
        expect(parseBIMI(stringified)).toEqual(parsed);
    });

    it("preserves spaced separators through a roundtrip", () => {
        const original = "v=BIMI1; l=https://example.com/logo.svg";
        const stringified = stringifyBIMI(parseBIMI(original), original);
        expect(stringified).toContain("; ");
    });
});

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
import { formatBytes } from "./format";

describe("formatBytes", () => {
    it("returns an em dash for undefined", () => {
        expect(formatBytes(undefined)).toBe("—");
    });

    it("returns an em dash for non-finite numbers", () => {
        expect(formatBytes(Infinity)).toBe("—");
        expect(formatBytes(-Infinity)).toBe("—");
        expect(formatBytes(NaN)).toBe("—");
    });

    it("formats zero bytes", () => {
        expect(formatBytes(0)).toBe("0 B");
    });

    it("formats small byte counts without decimals", () => {
        expect(formatBytes(1)).toBe("1 B");
        expect(formatBytes(512)).toBe("512 B");
        expect(formatBytes(1023)).toBe("1023 B");
    });

    it("formats exactly 1024 bytes as 1.0 KiB", () => {
        expect(formatBytes(1024)).toBe("1.0 KiB");
    });

    it("formats KiB values with one decimal below 100", () => {
        expect(formatBytes(1536)).toBe("1.5 KiB");
        expect(formatBytes(1024 * 99)).toBe("99.0 KiB");
    });

    it("drops the decimal once the value reaches 100 or more in a unit", () => {
        expect(formatBytes(1024 * 100)).toBe("100 KiB");
        expect(formatBytes(1024 * 999)).toBe("999 KiB");
    });

    it("formats MiB values", () => {
        expect(formatBytes(1024 * 1024)).toBe("1.0 MiB");
        expect(formatBytes(1024 * 1024 * 1.5)).toBe("1.5 MiB");
        expect(formatBytes(1024 * 1024 * 100)).toBe("100 MiB");
    });

    it("formats GiB values", () => {
        expect(formatBytes(1024 ** 3)).toBe("1.0 GiB");
        expect(formatBytes(1024 ** 3 * 2.25)).toBe("2.3 GiB");
    });

    it("formats TiB values", () => {
        expect(formatBytes(1024 ** 4)).toBe("1.0 TiB");
        expect(formatBytes(1024 ** 4 * 3.4)).toBe("3.4 TiB");
    });

    it("caps at TiB and keeps scaling for values beyond it", () => {
        expect(formatBytes(1024 ** 5)).toBe("1024 TiB");
        expect(formatBytes(1024 ** 5 * 2)).toBe("2048 TiB");
    });

    it("handles negative byte counts", () => {
        expect(formatBytes(-1)).toBe("-1 B");
        expect(formatBytes(-1024)).toBe("-1024 B");
    });

    it("rounds fractional sub-KiB byte counts to nearest integer", () => {
        expect(formatBytes(500.6)).toBe("501 B");
        expect(formatBytes(500.4)).toBe("500 B");
    });
});

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

import { describe, expect, it, afterEach, vi } from "vitest";

import { digestHex, hasSubtleCrypto } from "./crypto";

describe("hasSubtleCrypto", () => {
    afterEach(() => {
        vi.unstubAllGlobals();
    });

    it("is true in a secure context", () => {
        expect(hasSubtleCrypto()).toBe(true);
    });

    it("is false when crypto.subtle is missing", () => {
        vi.stubGlobal("crypto", { getRandomValues: () => undefined });

        expect(hasSubtleCrypto()).toBe(false);
    });
});

describe("digestHex", () => {
    afterEach(() => {
        vi.unstubAllGlobals();
    });

    const abc = new TextEncoder().encode("abc");

    it("hashes with SHA-256", async () => {
        await expect(digestHex("SHA-256", abc)).resolves.toBe(
            "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
        );
    });

    it("hashes with SHA-512", async () => {
        await expect(digestHex("SHA-512", abc)).resolves.toBe(
            "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a" +
                "2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f",
        );
    });

    it("truncates to the requested number of bytes", async () => {
        await expect(digestHex("SHA-256", abc, 6)).resolves.toBe("ba7816bf8f01");
    });

    it("gives up outside a secure context", async () => {
        vi.stubGlobal("crypto", {});

        await expect(digestHex("SHA-256", abc)).resolves.toBeUndefined();
    });
});

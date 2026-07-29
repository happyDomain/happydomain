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

import { randomUUID } from "./uuid";

const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

describe("randomUUID", () => {
    afterEach(() => {
        vi.unstubAllGlobals();
    });

    it("returns a v4 UUID in a secure context", () => {
        expect(randomUUID()).toMatch(UUID_V4);
    });

    it("falls back to getRandomValues when randomUUID is missing", () => {
        vi.stubGlobal("crypto", { getRandomValues: crypto.getRandomValues.bind(crypto) });

        expect(randomUUID()).toMatch(UUID_V4);
    });

    it("falls back to Math.random when Web Crypto is unavailable", () => {
        vi.stubGlobal("crypto", {});

        expect(randomUUID()).toMatch(UUID_V4);
    });

    it("does not repeat itself", () => {
        const ids = new Set(Array.from({ length: 1000 }, () => randomUUID()));

        expect(ids.size).toBe(1000);
    });
});

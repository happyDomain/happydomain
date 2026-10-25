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

import { afterEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ getEmailIdentifier: vi.fn() }));
vi.mock("$lib/api/email_identifier", () => api);

// Outside a secure context, the browser cannot hash.
vi.mock("$lib/utils/crypto", () => ({
    digestHex: vi.fn().mockResolvedValue(undefined),
    hasSubtleCrypto: () => false,
}));

import { computeEmailIdentifier } from "./email_identifier.svelte";

afterEach(() => {
    api.getEmailIdentifier.mockReset();
});

describe("computeEmailIdentifier outside a secure context", () => {
    it("asks the backend through the domain", async () => {
        api.getEmailIdentifier.mockResolvedValue("abcd");

        await expect(computeEmailIdentifier("d1", "alice")).resolves.toBe("abcd");
        expect(api.getEmailIdentifier).toHaveBeenCalledWith("d1", "alice");
    });

    it("fails without a domain to ask through, as for a zone template", async () => {
        await expect(computeEmailIdentifier("", "alice")).rejects.toThrow(/HTTPS/);
        expect(api.getEmailIdentifier).not.toHaveBeenCalled();
    });
});

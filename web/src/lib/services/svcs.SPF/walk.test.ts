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

import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("$lib/api/resolver", () => ({
    flattenSPF: vi.fn(),
}));

import { flattenSPF } from "$lib/api/resolver";
import type { HappydnsSpfFlattenResponse } from "$lib/api-base/types.gen";
import { forgetSPFWalk, walkSPF } from "./walk";

const RESPONSE: HappydnsSpfFlattenResponse = {
    record: "v=spf1 include:a.com -all",
    lookupCount: 3,
    exceeded: false,
    voidExceeded: false,
    voidLookups: 0,
    truncated: false,
};

const signal = () => new AbortController().signal;

describe("walkSPF", () => {
    beforeEach(() => {
        vi.mocked(flattenSPF).mockReset();
        forgetSPFWalk();
    });

    it("asks the resolver to walk the record as published at the domain", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(RESPONSE);
        await expect(walkSPF("example.com.", "v=spf1 include:a.com -all", signal())).resolves.toBe(
            RESPONSE,
        );
        expect(flattenSPF).toHaveBeenCalledWith({
            domain: "example.com.",
            record: "v=spf1 include:a.com -all",
        });
    });

    it("shares one walk between the callers of the same record", async () => {
        vi.mocked(flattenSPF).mockResolvedValue(RESPONSE);
        await Promise.all([
            walkSPF("example.com.", "v=spf1 include:a.com -all", signal()),
            walkSPF("example.com.", "v=spf1 include:a.com -all", signal()),
        ]);
        expect(flattenSPF).toHaveBeenCalledTimes(1);
    });

    it("walks again once the record or the domain changes", async () => {
        vi.mocked(flattenSPF).mockResolvedValue(RESPONSE);
        await walkSPF("example.com.", "v=spf1 include:a.com -all", signal());
        await walkSPF("example.com.", "v=spf1 include:b.com -all", signal());
        await walkSPF("mail.example.com.", "v=spf1 include:b.com -all", signal());
        expect(flattenSPF).toHaveBeenCalledTimes(3);
    });

    it("does not keep a failed walk, so that the next caller tries again", async () => {
        vi.mocked(flattenSPF).mockRejectedValueOnce(new Error("boom"));
        vi.mocked(flattenSPF).mockResolvedValueOnce(RESPONSE);
        await expect(
            walkSPF("example.com.", "v=spf1 include:a.com -all", signal()),
        ).rejects.toThrow("boom");
        await expect(walkSPF("example.com.", "v=spf1 include:a.com -all", signal())).resolves.toBe(
            RESPONSE,
        );
    });

    it("rejects with an AbortError for a caller that gives up, without failing the others", async () => {
        let resolve!: (r: HappydnsSpfFlattenResponse) => void;
        vi.mocked(flattenSPF).mockReturnValueOnce(new Promise((r) => (resolve = r)));

        const quitter = new AbortController();
        const given = walkSPF("example.com.", "v=spf1 include:a.com -all", quitter.signal);
        const kept = walkSPF("example.com.", "v=spf1 include:a.com -all", signal());
        quitter.abort();
        resolve(RESPONSE);

        await expect(given).rejects.toMatchObject({ name: "AbortError" });
        await expect(kept).resolves.toBe(RESPONSE);
    });

    it("rejects at once when the signal is already aborted", async () => {
        vi.mocked(flattenSPF).mockResolvedValueOnce(RESPONSE);
        const ctrl = new AbortController();
        ctrl.abort();
        await expect(
            walkSPF("example.com.", "v=spf1 include:a.com -all", ctrl.signal),
        ).rejects.toMatchObject({ name: "AbortError" });
    });
});

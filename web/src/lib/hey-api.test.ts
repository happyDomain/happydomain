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

const unlock = vi.hoisted(() => ({ answer: true, asked: [] as string[] }));
vi.mock("$lib/stores/unlock", () => ({
    requestUnlock: async (locked: { ref: string }) => {
        unlock.asked.push(locked.ref);
        return unlock.answer;
    },
}));

import { customFetch, KeystoreLockedError } from "./hey-api";

function locked(): Response {
    return new Response(
        JSON.stringify({ errmsg: "keystore abc is locked", locked: { ref: "abc", form: [] } }),
        {
            status: 423,
            headers: { "content-type": "application/json" },
        },
    );
}

describe("customFetch on a locked keystore", () => {
    afterEach(() => {
        vi.unstubAllGlobals();
        unlock.asked = [];
    });

    it("asks to unlock it, then sends the request again, body included", async () => {
        unlock.answer = true;
        const bodies: string[] = [];
        const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
            bodies.push(await (input as Request).text());
            return bodies.length === 1 ? locked() : new Response("{}", { status: 200 });
        });
        vi.stubGlobal("fetch", fetchMock);

        const res = await customFetch(
            new Request("http://x/api/providers", { method: "POST", body: '{"a":1}' }),
        );

        expect(res.status).toBe(200);
        expect(unlock.asked).toEqual(["abc"]);
        expect(bodies).toEqual(['{"a":1}', '{"a":1}']);
    });

    it("fails with KeystoreLockedError when the user gives up", async () => {
        unlock.answer = false;
        vi.stubGlobal(
            "fetch",
            vi.fn(async () => locked()),
        );

        await expect(customFetch("http://x/api/providers")).rejects.toBeInstanceOf(
            KeystoreLockedError,
        );
    });

    it("returns a 423 naming no keystore as is, without asking", async () => {
        vi.stubGlobal(
            "fetch",
            vi.fn(async () => new Response("locked", { status: 423 })),
        );

        const res = await customFetch("http://x/api/providers");

        expect(res.status).toBe(423);
        expect(unlock.asked).toEqual([]);
    });

    it("asks again when the keystore is still locked after unlocking", async () => {
        unlock.answer = true;
        let calls = 0;
        vi.stubGlobal(
            "fetch",
            vi.fn(async () => (++calls < 3 ? locked() : new Response("{}", { status: 200 }))),
        );

        const res = await customFetch("http://x/api/providers");

        expect(res.status).toBe(200);
        expect(unlock.asked).toEqual(["abc", "abc"]);
    });
});

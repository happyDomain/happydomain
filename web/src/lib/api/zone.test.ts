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

// The network lives here: the tests decide what the backend answers.
const sdk = vi.hoisted(() => ({ getDomainsByDomainIdRetrieveZone: vi.fn() }));
vi.mock("$lib/api-base/sdk.gen", async (importOriginal) => ({
    ...(await importOriginal<typeof import("$lib/api-base/sdk.gen")>()),
    ...sdk,
}));

import type { Domain } from "$lib/model/domain";
import { getRetrieveZoneProgress } from "./zone";

const domain = { id: "d1", domain: "example.com." } as unknown as Domain;

afterEach(() => {
    vi.resetAllMocks();
});

describe("getRetrieveZoneProgress", () => {
    it("tells no retrieval is in flight", async () => {
        sdk.getDomainsByDomainIdRetrieveZone.mockResolvedValue({ response: { status: 204 } });

        expect(await getRetrieveZoneProgress(domain)).toBeNull();
        expect(sdk.getDomainsByDomainIdRetrieveZone).toHaveBeenCalledWith({
            path: { domainId: "d1" },
        });
    });

    it("tells how far the retrieval in flight is", async () => {
        const progress = { step: "exploring", percent: 42 };
        sdk.getDomainsByDomainIdRetrieveZone.mockResolvedValue({
            data: progress,
            response: { status: 200 },
        });

        expect(await getRetrieveZoneProgress(domain)).toEqual(progress);
    });

    it("throws the error of the backend", async () => {
        sdk.getDomainsByDomainIdRetrieveZone.mockResolvedValue({
            error: { errmsg: "domain not found" },
            response: { status: 404 },
        });

        await expect(getRetrieveZoneProgress(domain)).rejects.toThrow("domain not found");
    });
});

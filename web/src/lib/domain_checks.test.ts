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

import { afterEach, describe, expect, it, vi } from "vitest";

// The network lives here: the tests decide what each scope answers.
const api = vi.hoisted(() => ({ listScopedCheckers: vi.fn() }));
vi.mock("$lib/api/checkers", () => api);

import type { HappydnsCheckerStatus } from "$lib/api-base/types.gen";
import { makeService, makeZone } from "$lib/test-utils/fixtures";
import { StatusCrit, StatusOK, StatusWarn } from "$lib/utils/checkers";
import { loadCheckSections, worstStatus, type CheckItem } from "./domain_checks";

function checker(id: string, status?: number): HappydnsCheckerStatus {
    return {
        id,
        name: id,
        latestExecution: status === undefined ? undefined : { result: { status } },
    } as unknown as HappydnsCheckerStatus;
}

function item(status?: number): CheckItem {
    return { status: checker("c", status), scope: { domainId: "d1" } };
}

afterEach(() => {
    vi.resetAllMocks();
});

describe("worstStatus", () => {
    it("tells the worst status of the checks that ran", () => {
        expect(worstStatus([item(StatusOK), item(), item(StatusCrit), item(StatusWarn)])).toBe(
            StatusCrit,
        );
    });

    it("tells nothing when no check ran", () => {
        expect(worstStatus([item(), item()])).toBeUndefined();
        expect(worstStatus([])).toBeUndefined();
    });
});

describe("loadCheckSections", () => {
    it("groups the checks by subdomain, in the order of the zone, skipping those without", async () => {
        const zone = makeZone({
            id: "z1",
            services: {
                "": [makeService("svcs.MXs", {}, { _id: "s1" })],
                www: [makeService("svcs.CNAME", {}, { _id: "s2", _domain: "www" })],
                "a.b": [makeService("svcs.TXT", {}, { _id: "s3", _domain: "a.b" })],
                ftp: [makeService("svcs.TXT", {}, { _id: "s4", _domain: "ftp" })],
            },
        });
        api.listScopedCheckers.mockImplementation(async (scope) => {
            if (!scope.serviceId) return [checker("dns", StatusOK)];
            if (scope.serviceId === "s4") return [];
            return [checker("svc-" + scope.serviceId, StatusWarn)];
        });

        const sections = await loadCheckSections("d1", zone);

        expect(sections.map((s) => s.subdomain)).toEqual(["@", "a.b", "www"]);
        // The domain's own checks belong to the apex, with those of its services.
        expect(sections[0].items.map((i) => i.status.id)).toEqual(["svc-s1", "dns"]);
        expect(sections[0].items[0].target).toEqual({ subdomain: "@", serviceId: "s1" });
        expect(sections[0].items[1].target).toBeUndefined();
        expect(api.listScopedCheckers).toHaveBeenCalledWith({
            domainId: "d1",
            zoneId: "z1",
            subdomain: "www",
            serviceId: "s2",
        });
    });

    it("lists only the checks of the domain without a zone", async () => {
        api.listScopedCheckers.mockResolvedValue([checker("dns", StatusOK)]);

        const sections = await loadCheckSections("d1", null);

        expect(sections).toHaveLength(1);
        expect(sections[0].subdomain).toBe("@");
        expect(api.listScopedCheckers).toHaveBeenCalledTimes(1);
    });
});

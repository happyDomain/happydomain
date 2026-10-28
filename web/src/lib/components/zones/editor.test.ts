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

const api = vi.hoisted(() => ({
    addZoneService: vi.fn(),
    updateZoneService: vi.fn(),
    deleteZoneService: vi.fn(),
    addZoneRecord: vi.fn(),
    updateZoneRecord: vi.fn(),
    deleteZoneRecord: vi.fn(),
}));
vi.mock("$lib/api/zone", () => api);

// Each app resolves the paths its own way: tell what the editor asks for.
vi.mock("$links", () => ({
    domainLinks: () => ({
        subdomain: (dn: string, subdomain: string) => `subdomain ${dn} ${subdomain}`,
        service: (dn: string, subdomain: string, serviceid: string) =>
            `service ${dn} ${subdomain} ${serviceid}`,
        historySubdomain: (dn: string, zid: string, subdomain: string) =>
            `history ${zid} subdomain ${dn} ${subdomain}`,
        historyService: (dn: string, zid: string, subdomain: string, serviceid: string) =>
            `history ${zid} service ${dn} ${subdomain} ${serviceid}`,
    }),
}));

import type { dnsRR } from "$lib/dns_rr";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";
import { domainZoneEditor } from "./editor";

afterEach(() => {
    vi.resetAllMocks();
});

describe("domainZoneEditor", () => {
    it("saves the changes to the zone of the domain read at each call", async () => {
        let domain = makeDomain({ id: "d1" });
        const editor = domainZoneEditor(() => domain);
        const derived = makeZone({ id: "z2" });
        api.addZoneService.mockResolvedValue(derived);

        const svc = makeService("svcs.MXs");
        expect(await editor.addService("z1", svc)).toBe(derived);
        expect(api.addZoneService).toHaveBeenCalledWith(domain, "z1", svc);

        domain = makeDomain({ id: "d2" });
        const rr = { Hdr: { Name: "www", Rrtype: 1 } } as unknown as dnsRR;
        await editor.updateRecord("z1", "www", rr, rr);
        expect(api.updateZoneRecord).toHaveBeenCalledWith(domain, "z1", "www", rr, rr);
    });

    it("tells it is the zone of a domain, left to its provider to make read-only", () => {
        const editor = domainZoneEditor(() => makeDomain());

        expect(editor.isDomain).toBe(true);
        expect(editor.readOnly).toBe(false);
    });

    it("links to the pages of the current zone, whatever the id of the zone changed", () => {
        const editor = domainZoneEditor(() => makeDomain({ id: "d/1" }));

        expect(editor.subdomainHref("z1", "")).toBe("subdomain d%2F1 %40");
        expect(editor.subdomainHref("z2", "www")).toBe("subdomain d%2F1 www");
        expect(editor.serviceHref("z1", "", "new")).toBe("service d%2F1 %40 new");
        expect(editor.serviceHref("z1", "www", "s1")).toBe("service d%2F1 www s1");
    });

    it("shows a past version read-only, linking to its own pages", () => {
        let version: string | undefined = "z/0";
        const editor = domainZoneEditor(
            () => makeDomain({ id: "d1" }),
            () => version,
        );

        expect(editor.readOnly).toBe(true);
        expect(editor.subdomainHref("z0", "www")).toBe("history z%2F0 subdomain d1 www");
        expect(editor.serviceHref("z0", "", "s1")).toBe("history z%2F0 service d1 %40 s1");

        version = undefined;
        expect(editor.readOnly).toBe(false);
        expect(editor.subdomainHref("z1", "www")).toBe("subdomain d1 www");
    });
});

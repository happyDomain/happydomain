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

const sdk = vi.hoisted(() => ({
    postTemplatesImport: vi.fn(),
    deleteTemplatesByTemplateId: vi.fn(),
    postTemplatesByTemplateIdBySubdomainServices: vi.fn(),
    putTemplatesByTemplateIdBySubdomainServicesByServiceId: vi.fn(),
    deleteTemplatesByTemplateIdBySubdomainServicesByServiceId: vi.fn(),
    postTemplatesByTemplateIdRecords: vi.fn(),
    postTemplatesByTemplateIdRecordsDelete: vi.fn(),
    patchTemplatesByTemplateIdRecords: vi.fn(),
    postDomainsByDomainIdZoneByZoneIdApplyTemplate: vi.fn(),
}));
vi.mock("$lib/api-base/sdk.gen", async (importOriginal) => ({
    ...(await importOriginal<typeof import("$lib/api-base/sdk.gen")>()),
    ...sdk,
}));

import type { dnsRR } from "$lib/dns_rr";
import { makeDomain, makeService, makeZone } from "$lib/test-utils/fixtures";
import {
    addZoneTemplateRecord,
    addZoneTemplateService,
    applyZoneTemplate,
    deleteZoneTemplate,
    deleteZoneTemplateRecord,
    deleteZoneTemplateService,
    importZoneTemplate,
    updateZoneTemplateRecord,
    updateZoneTemplateService,
} from "./zone_template";

const saved = { id: "t1", name: "Mail" };
const ok = { data: saved, response: { status: 200 } };

function a(address: string): dnsRR {
    return { Hdr: { Name: "", Rrtype: 1, Class: 1, Ttl: 300 }, A: address } as unknown as dnsRR;
}

afterEach(() => {
    vi.resetAllMocks();
});

describe("zone templates", () => {
    it("imports a zone file, the name sent even when not given", async () => {
        sdk.postTemplatesImport.mockResolvedValue(ok);
        const file = new Blob(["www 3600 IN A 192.0.2.1\n"]);

        expect(await importZoneTemplate({ origin: "example.com." }, file)).toEqual(saved);
        expect(sdk.postTemplatesImport).toHaveBeenCalledWith({
            body: { name: "", description: undefined, origin: "example.com.", zone: file },
        });
    });

    it("deletes a template, answered without content", async () => {
        sdk.deleteTemplatesByTemplateId.mockResolvedValue({
            data: undefined,
            response: { status: 204, ok: true },
        });

        await expect(deleteZoneTemplate("t1")).resolves.toBeUndefined();
        expect(sdk.deleteTemplatesByTemplateId).toHaveBeenCalledWith({
            path: { templateId: "t1" },
        });
    });

    it("throws the error of the backend on deletion", async () => {
        sdk.deleteTemplatesByTemplateId.mockResolvedValue({
            error: { errmsg: "template not found" },
            response: { status: 404 },
        });

        await expect(deleteZoneTemplate("t1")).rejects.toThrow("template not found");
    });
});

describe("zone template services", () => {
    it("adds a service of the root under @", async () => {
        sdk.postTemplatesByTemplateIdBySubdomainServices.mockResolvedValue(ok);
        const svc = makeService("svcs.MXs");

        expect(await addZoneTemplateService("t1", svc)).toEqual(saved);
        expect(sdk.postTemplatesByTemplateIdBySubdomainServices).toHaveBeenCalledWith({
            path: { templateId: "t1", subdomain: "@" },
            body: svc,
        });
    });

    it("updates a service of a subdomain by its identifier", async () => {
        sdk.putTemplatesByTemplateIdBySubdomainServicesByServiceId.mockResolvedValue(ok);
        const svc = makeService("svcs.MXs", {}, { _domain: "mail", _id: "s1" });

        await updateZoneTemplateService("t1", svc);
        expect(sdk.putTemplatesByTemplateIdBySubdomainServicesByServiceId).toHaveBeenCalledWith({
            path: { templateId: "t1", subdomain: "mail", serviceId: "s1" },
            body: svc,
        });
    });

    it("deletes a service of the root under @", async () => {
        sdk.deleteTemplatesByTemplateIdBySubdomainServicesByServiceId.mockResolvedValue(ok);

        await deleteZoneTemplateService("t1", makeService("svcs.MXs", {}, { _id: "s1" }));
        expect(sdk.deleteTemplatesByTemplateIdBySubdomainServicesByServiceId).toHaveBeenCalledWith({
            path: { templateId: "t1", subdomain: "@", serviceId: "s1" },
        });
    });

    it("throws the error of the backend", async () => {
        sdk.postTemplatesByTemplateIdBySubdomainServices.mockResolvedValue({
            error: { errmsg: "template not found" },
            response: { status: 404 },
        });

        await expect(addZoneTemplateService("t1", makeService("svcs.MXs"))).rejects.toThrow(
            "template not found",
        );
    });
});

describe("zone template records", () => {
    it("names the records after their subdomain", async () => {
        sdk.postTemplatesByTemplateIdRecords.mockResolvedValue(ok);

        await addZoneTemplateRecord("t1", "www", a("192.0.2.1"));
        expect(sdk.postTemplatesByTemplateIdRecords).toHaveBeenCalledWith({
            path: { templateId: "t1" },
            body: ["www\t300\tIN\tA\t192.0.2.1"],
        });
    });

    it("deletes the records named after their subdomain", async () => {
        sdk.postTemplatesByTemplateIdRecordsDelete.mockResolvedValue(ok);

        await deleteZoneTemplateRecord("t1", "www", a("192.0.2.1"));
        expect(sdk.postTemplatesByTemplateIdRecordsDelete).toHaveBeenCalledWith({
            path: { templateId: "t1" },
            body: ["www\t300\tIN\tA\t192.0.2.1"],
        });
    });

    it("sends the record changed along with the one it replaces", async () => {
        sdk.patchTemplatesByTemplateIdRecords.mockResolvedValue(ok);

        await updateZoneTemplateRecord("t1", "www", a("192.0.2.2"), a("192.0.2.1"));
        expect(sdk.patchTemplatesByTemplateIdRecords).toHaveBeenCalledWith({
            path: { templateId: "t1" },
            body: { oldrr: "www\t300\tIN\tA\t192.0.2.1", newrr: "www\t300\tIN\tA\t192.0.2.2" },
        });
    });
});

describe("applyZoneTemplate", () => {
    it("applies the template to the zone of the domain, and gives the zone changed", async () => {
        const zone = makeZone({ id: "z2" });
        sdk.postDomainsByDomainIdZoneByZoneIdApplyTemplate.mockResolvedValue({
            data: zone,
            response: { status: 200 },
        });

        const got = await applyZoneTemplate(makeDomain({ id: "d1" }), "z1", {
            template_id: "t1",
            mode: "merge",
        });
        expect(got).toEqual(zone);
        expect(sdk.postDomainsByDomainIdZoneByZoneIdApplyTemplate).toHaveBeenCalledWith({
            path: { domainId: "d1", zoneId: "z1" },
            body: { template_id: "t1", mode: "merge" },
        });
    });
});

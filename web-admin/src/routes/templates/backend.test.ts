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
    getTemplates: vi.fn(),
    getTemplatesByTemplateId: vi.fn(),
    putTemplatesByTemplateId: vi.fn(),
    deleteTemplatesByTemplateId: vi.fn(),
    postTemplates: vi.fn(),
    postTemplatesImport: vi.fn(),
    postTemplatesByTemplateIdBySubdomainServices: vi.fn(),
    putTemplatesByTemplateIdBySubdomainServicesByServiceId: vi.fn(),
    deleteTemplatesByTemplateIdBySubdomainServicesByServiceId: vi.fn(),
    postTemplatesByTemplateIdRecords: vi.fn(),
    patchTemplatesByTemplateIdRecords: vi.fn(),
    postTemplatesByTemplateIdRecordsDelete: vi.fn(),
}));
vi.mock("$lib/api-admin", async (importOriginal) => ({
    ...(await importOriginal<typeof import("$lib/api-admin")>()),
    ...sdk,
}));

import type { dnsRR } from "$lib/dns_rr";
import { ServiceCombined } from "$lib/model/service.svelte";
import { adminTemplateBackend, createGlobalTemplate, listGlobalTemplates } from "./backend";

const sent = {
    id: "t1",
    name: "Mail",
    default_ttl: 3600,
    last_modified: "2026-01-02T03:04:05Z",
    services: {
        "": [
            {
                _svctype: "svcs.MXs",
                _domain: "",
                _propagated_at: "2026-01-01T00:00:00Z",
                Service: { mx: [] },
            },
        ],
    },
};

function a(address: string): dnsRR {
    return { Hdr: { Name: "", Rrtype: 1, Class: 1, Ttl: 300 }, A: address } as unknown as dnsRR;
}

afterEach(() => {
    vi.resetAllMocks();
});

describe("adminTemplateBackend", () => {
    it("gives the dates the administrative API sends as strings as dates", async () => {
        sdk.getTemplates.mockResolvedValue({ data: [sent] });

        const [t] = await listGlobalTemplates();
        expect(t.last_modified).toEqual(new Date("2026-01-02T03:04:05Z"));
        expect(t.services[""][0]._propagated_at).toEqual(new Date("2026-01-01T00:00:00Z"));
        expect(t.services[""][0].Service).toEqual({ mx: [] });
    });

    it("sends a new service with its fields, and its dates as strings", async () => {
        sdk.postTemplatesByTemplateIdBySubdomainServices.mockResolvedValue({ data: sent });
        const svc = new ServiceCombined({
            _svctype: "svcs.MXs",
            _domain: "",
            _propagated_at: new Date("2026-01-01T00:00:00Z"),
            Service: { mx: [] },
        });

        await adminTemplateBackend.addService("t1", svc);
        const call = sdk.postTemplatesByTemplateIdBySubdomainServices.mock.calls[0][0];
        expect(call.path).toEqual({ templateId: "t1", subdomain: "@" });
        expect(call.body).toMatchObject({
            _svctype: "svcs.MXs",
            _propagated_at: "2026-01-01T00:00:00.000Z",
            Service: { mx: [] },
        });
    });

    it("leaves the date of a service never propagated unset", async () => {
        const unpropagated = { ...sent.services[""][0], _propagated_at: undefined };
        sdk.getTemplatesByTemplateId.mockResolvedValue({
            data: { ...sent, services: { "": [unpropagated] } },
        });

        const t = await adminTemplateBackend.get("t1");
        expect(t.services[""][0]._propagated_at).toBeUndefined();
        expect(sdk.getTemplatesByTemplateId).toHaveBeenCalledWith({ path: { templateId: "t1" } });
    });

    it("changes the details of a template, and deletes it", async () => {
        sdk.putTemplatesByTemplateId.mockResolvedValue({ data: { ...sent, name: "Mail servers" } });
        sdk.deleteTemplatesByTemplateId.mockResolvedValue({ data: undefined });
        const input = { name: "Mail servers", description: "" };

        expect((await adminTemplateBackend.update("t1", input)).name).toBe("Mail servers");
        expect(sdk.putTemplatesByTemplateId).toHaveBeenCalledWith({
            path: { templateId: "t1" },
            body: input,
        });
        await expect(adminTemplateBackend.delete("t1")).resolves.toBeUndefined();
        expect(sdk.deleteTemplatesByTemplateId).toHaveBeenCalledWith({
            path: { templateId: "t1" },
        });
    });

    it("updates and deletes a service of the root under @, by its identifier", async () => {
        sdk.putTemplatesByTemplateIdBySubdomainServicesByServiceId.mockResolvedValue({
            data: sent,
        });
        sdk.deleteTemplatesByTemplateIdBySubdomainServicesByServiceId.mockResolvedValue({
            data: sent,
        });
        const svc = new ServiceCombined({
            _id: "s1",
            _svctype: "svcs.MXs",
            _domain: "",
            Service: { mx: [] },
        });
        const path = { templateId: "t1", subdomain: "@", serviceId: "s1" };

        await adminTemplateBackend.updateService("t1", svc);
        expect(sdk.putTemplatesByTemplateIdBySubdomainServicesByServiceId).toHaveBeenCalledWith(
            expect.objectContaining({ path }),
        );
        await adminTemplateBackend.deleteService("t1", svc);
        expect(sdk.deleteTemplatesByTemplateIdBySubdomainServicesByServiceId).toHaveBeenCalledWith({
            path,
        });
    });

    it("names the records after their subdomain", async () => {
        sdk.postTemplatesByTemplateIdRecords.mockResolvedValue({ data: sent });
        sdk.patchTemplatesByTemplateIdRecords.mockResolvedValue({ data: sent });
        sdk.postTemplatesByTemplateIdRecordsDelete.mockResolvedValue({ data: sent });
        const path = { templateId: "t1" };

        await adminTemplateBackend.addRecord("t1", "www", a("192.0.2.1"));
        expect(sdk.postTemplatesByTemplateIdRecords).toHaveBeenCalledWith({
            path,
            body: ["www\t300\tIN\tA\t192.0.2.1"],
        });
        await adminTemplateBackend.updateRecord("t1", "www", a("192.0.2.2"), a("192.0.2.1"));
        expect(sdk.patchTemplatesByTemplateIdRecords).toHaveBeenCalledWith({
            path,
            body: { oldrr: "www\t300\tIN\tA\t192.0.2.1", newrr: "www\t300\tIN\tA\t192.0.2.2" },
        });
        await adminTemplateBackend.deleteRecord("t1", "www", a("192.0.2.2"));
        expect(sdk.postTemplatesByTemplateIdRecordsDelete).toHaveBeenCalledWith({
            path,
            body: ["www\t300\tIN\tA\t192.0.2.2"],
        });
    });

    it("throws the error of the backend", async () => {
        sdk.getTemplates.mockResolvedValue({ error: { errmsg: "not an administrator" } });
        await expect(listGlobalTemplates()).rejects.toThrow("not an administrator");

        sdk.getTemplates.mockResolvedValue({ error: "unreachable" });
        await expect(listGlobalTemplates()).rejects.toThrow("unreachable");
    });
});

describe("createGlobalTemplate", () => {
    const input = { name: "Mail", description: "Our mail servers" };

    it("creates an empty template", async () => {
        sdk.postTemplates.mockResolvedValue({ data: sent });

        expect((await createGlobalTemplate(input)).id).toBe("t1");
        expect(sdk.postTemplates).toHaveBeenCalledWith({ body: input });
        expect(sdk.postTemplatesImport).not.toHaveBeenCalled();
    });

    it("imports the zone file given", async () => {
        sdk.postTemplatesImport.mockResolvedValue({ data: sent });
        const file = new Blob(["@ IN TXT hello"]);

        await createGlobalTemplate({ ...input, origin: "example.org." }, file);
        expect(sdk.postTemplatesImport).toHaveBeenCalledWith({
            body: { ...input, origin: "example.org.", zone: file },
        });
        expect(sdk.postTemplates).not.toHaveBeenCalled();
    });
});

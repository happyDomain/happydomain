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

import { cleanup, fireEvent, render, screen } from "@testing-library/svelte";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

vi.mock("$lib/api/service_specs", () => ({
    getServiceSpec: vi.fn(async () => ({ fields: [{ id: "A", type: "dns.A" }] })),
}));

import type { HappydnsCheckerStatus } from "$lib/api-base/types.gen";
import type { CheckSection } from "$lib/domain_checks";
import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import { domainChecks } from "$lib/stores/domainchecks";
import { providers } from "$lib/stores/providers";
import { makeZone } from "$lib/test-utils/fixtures";
import { loadTranslations } from "$lib/translations";
import { domainZoneEditor, zoneEditorContext, type ZoneEditor } from "./editor";
import { controls as ctrlRecord } from "$lib/components/modals/Record.svelte";
import ServiceCard from "./ServiceCard.svelte";
import { controls as ctrlServiceDetails } from "./ServiceDetailsOffcanvas.svelte";

const domain = { id: "d1", domain: "example.com.", id_provider: "p1" } as unknown as Domain;

const service = {
    _id: "s1",
    _svctype: "abstract.Server",
    _domain: "www",
    _ttl: 300,
    Service: { A: { Hdr: { Name: "www", Rrtype: 1, Class: 1, Ttl: 300 }, A: "192.0.2.1" } },
} as unknown as ServiceWithValue;

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    providers.set(undefined);
    domainChecks.set(null);
});

function setup(readonly: boolean, context?: Map<symbol, ZoneEditor>) {
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    return render(ServiceCard, {
        props: { dn: "www", origin: domain, service, zoneId: "z1" },
        context,
    });
}

function withChecks(serviceId: string) {
    const check = (id: string) => ({
        status: { id } as HappydnsCheckerStatus,
        scope: {},
        target: { subdomain: "www", serviceId },
    });
    const sections: CheckSection[] = [{ subdomain: "www", items: [check("c1"), check("c2")] }];
    domainChecks.set({
        domainId: "d1",
        zone: makeZone({ id: "z1" }),
        sections: Promise.resolve(sections),
    });
}

describe("ServiceCard", () => {
    it("leads to the editor of the service", () => {
        setup(false);
        expect(screen.getByRole("link")).toHaveAttribute(
            "href",
            expect.stringMatching(/\/www\/.*s1/),
        );
    });

    it("opens a record in its modal, rather than the editor of the service", async () => {
        const open = vi.spyOn(ctrlRecord, "Open").mockImplementation(() => {});
        setup(false);

        await fireEvent.click(await screen.findByRole("button", { name: /192\.0\.2\.1/ }));
        expect(open).toHaveBeenCalledWith(service.Service.A, "www");
    });

    it("offers the TTL, to edit and to delete the service through a writable provider", async () => {
        setup(false);
        await fireEvent.click(screen.getByRole("button", { name: "Service actions" }));
        expect(screen.getByLabelText("Default TTL")).toBeEnabled();
        expect(screen.getByRole("button", { name: /Edit this service/ })).toBeEnabled();
        expect(screen.getByRole("button", { name: /Delete this service/ })).toBeEnabled();
    });

    it("shows the TTL but changes nothing through a read-only provider", async () => {
        setup(true);
        await fireEvent.click(screen.getByRole("button", { name: "Service actions" }));
        expect(screen.getByLabelText("Default TTL")).toBeDisabled();
        expect(screen.getByRole("button", { name: /Edit this service/ })).toHaveAttribute(
            "aria-disabled",
            "true",
        );
        expect(screen.queryByRole("button", { name: /Delete this service/ })).toBeNull();
    });

    it("shows no check badge to a service without checks", () => {
        withChecks("other");
        setup(false);
        expect(screen.queryByRole("button", { name: /Service checks/i })).toBeNull();
    });

    it("shows a dot by check, opening the details of the service", async () => {
        const open = vi.spyOn(ctrlServiceDetails, "Open").mockImplementation(() => {});
        withChecks("s1");
        const { container } = setup(false);

        const badge = await screen.findByRole("button", { name: /Service checks/i });
        expect(container.querySelectorAll(".check-dot")).toHaveLength(2);
        await fireEvent.click(badge);
        expect(open).toHaveBeenCalledWith(service);
    });

    it("shows no check in a zone the page made read-only", async () => {
        const editor: ZoneEditor = { ...domainZoneEditor(() => domain), readOnly: true };
        withChecks("s1");
        setup(false, zoneEditorContext(editor));
        await Promise.resolve();
        expect(screen.queryByRole("button", { name: /Service checks/i })).toBeNull();
    });
});

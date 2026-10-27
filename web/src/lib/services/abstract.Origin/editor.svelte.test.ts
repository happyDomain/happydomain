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

import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import type { dnsTypeNS, dnsTypeSOA } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import type { AbstractOriginBody } from "$lib/services_bodies";
import { loadTranslations } from "$lib/translations";
import Editor from "./editor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

const origin = { domain: "example.com." } as Domain;

function ns(target: string): dnsTypeNS {
    return {
        Hdr: { Name: "", Rrtype: 2, Class: 1, Ttl: 3600, Rdlength: 0 },
        Ns: target,
    };
}

const soa: dnsTypeSOA = {
    Hdr: { Name: "", Rrtype: 6, Class: 1, Ttl: 3600, Rdlength: 0 },
    Ns: "ns0",
    Mbox: "hostmaster",
    Serial: 2026102701,
    Refresh: 10800,
    Retry: 3600,
    Expire: 604800,
    Minttl: 3600,
};

// The value has to be state: a plain object handed to a component is proxied
// on its way in, and the proxy does not write back to the object kept here.
function mount(nameServers: dnsTypeNS[] | null) {
    const value = $state({ soa: { ...soa }, ns: nameServers } as AbstractOriginBody);

    const { container } = render(Editor, { dn: "", origin, value });

    return {
        value,
        input: (id: string) => container.querySelector<HTMLInputElement>("#" + id)!,
    };
}

describe("Origin editor", () => {
    it("adds a name server to a zone that has none yet", async () => {
        const user = userEvent.setup();
        const { value, input } = mount(null);

        await user.click(screen.getByRole("button", { name: /New row/ }));

        expect(value.ns).toHaveLength(1);
        await user.type(input("ns-0"), "ns1");
        expect(value.ns[0].Ns).toBe("ns1");
    });

    it("adds a name server after the existing ones", async () => {
        const user = userEvent.setup();
        const { value, input } = mount([ns("ns1")]);

        await user.click(screen.getByRole("button", { name: /New row/ }));
        await user.type(input("ns-1"), "ns2.ovh.net.");

        expect(value.ns.map((rr) => rr.Ns)).toEqual(["ns1", "ns2.ovh.net."]);
    });

    it("shows name servers as they are stored, relative or absolute", () => {
        const { input } = mount([ns("ns1"), ns("ns1.example.com."), ns("dns.ovh.net.")]);

        expect(input("ns-0")).toHaveValue("ns1");
        expect(input("ns-1")).toHaveValue("ns1.example.com.");
        expect(input("ns-2")).toHaveValue("dns.ovh.net.");
    });

    it("removes a name server", async () => {
        const user = userEvent.setup();
        const { value } = mount([ns("ns1"), ns("ns2")]);

        const trash = screen.getAllByRole("button").filter((b) => b.querySelector(".bi-trash"));
        await user.click(trash[0]);

        expect(value.ns.map((rr) => rr.Ns)).toEqual(["ns2"]);
    });

    it("edits the primary name server of the SOA as typed", async () => {
        const user = userEvent.setup();
        const { value, input } = mount([]);

        const mname = input("spec-soa_mname-mname");
        expect(mname).toHaveValue("ns0");

        await user.clear(mname);
        await user.type(mname, "ns1");
        expect(value.soa.Ns).toBe("ns1");

        await user.clear(mname);
        await user.type(mname, "ns1.example.com.");
        expect(value.soa.Ns).toBe("ns1.example.com.");

        await user.clear(mname);
        await user.type(mname, "ns.provider.net.");
        expect(value.soa.Ns).toBe("ns.provider.net.");
    });
});

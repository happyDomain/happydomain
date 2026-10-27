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

import { getRrtype, newRR, type dnsRR } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import { loadTranslations } from "$lib/translations";
import Editor from "./Editor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

const origin = { domain: "example.com." } as Domain;

function mount(rrtype: string, fields: Record<string, unknown> = {}, dn = "") {
    const record = $state(Object.assign(newRR("www", getRrtype(rrtype)), fields) as dnsRR);

    const { container } = render(Editor, { dn, origin, record });

    return {
        record,
        input: (id: string) => container.querySelector<HTMLInputElement>("#" + id)!,
    };
}

describe("Record editor", () => {
    describe("owner name", () => {
        it("is shown relative to the zone", () => {
            const { input } = mount("A");

            expect(input("rr-hdr-name")).toHaveValue("www");
            expect(screen.getByText(".example.com.")).toBeVisible();
        });

        it("is relative to the subdomain of the service", () => {
            const { input } = mount("A", {}, "blog");

            expect(input("rr-hdr-name")).toHaveValue("www");
            expect(screen.getByText(".blog.example.com.")).toBeVisible();
            expect(screen.getByText("www.blog.example.com.")).toBeVisible();
        });

        it("cannot leave the zone", async () => {
            const user = userEvent.setup();
            const { record, input } = mount("A");

            await user.clear(input("rr-hdr-name"));
            await user.type(input("rr-hdr-name"), "www.other.org.");

            expect(input("rr-hdr-name")).toHaveClass("is-invalid");
            expect(screen.getByText("The name has to be within example.com.")).toBeVisible();
            expect(record.Hdr.Name).not.toBe("www.other.org.");
        });
    });

    describe("rdata", () => {
        it("edits a mail exchanger as a name of the zone", async () => {
            const user = userEvent.setup();
            const { record, input } = mount("MX");

            await user.type(input("rr-Mx"), "mail");

            expect(record.Mx).toBe("mail");
            expect(screen.getByText("mail.example.com.")).toBeVisible();
        });

        it("keeps a mail exchanger typed in full absolute", async () => {
            const user = userEvent.setup();
            const { record, input } = mount("MX");

            await user.type(input("rr-Mx"), "mail.example.com.");

            expect(record.Mx).toBe("mail.example.com.");
        });

        it("offers to make absolute a target that looks like one", async () => {
            const user = userEvent.setup();
            const { record, input } = mount("CNAME");

            await user.type(input("rr-Target"), "ghs.googlehosted.com");
            await user.click(screen.getByRole("button", { name: "Use it" }));

            expect(record.Target).toBe("ghs.googlehosted.com.");
        });

        it("edits the target of a SRV record as a name, its port as is", async () => {
            const user = userEvent.setup();
            const { record, input } = mount("SRV");

            await user.type(input("rr-Target"), "sip");
            expect(record.Target).toBe("sip");

            expect(input("rr-Port").closest(".input-group")).toBeNull();
        });

        it("edits the mailbox of a SOA as an e-mail address", () => {
            const { input } = mount("SOA", { Ns: "ns0", Mbox: "john\\.doe" });

            expect(input("rr-Mbox")).toHaveValue("john.doe");
            expect(screen.getByText("@example.com")).toBeVisible();
            expect(input("rr-Ns")).toHaveValue("ns0");
        });

        it("leaves the fields holding no name as plain text", async () => {
            const user = userEvent.setup();
            const { record, input } = mount("TXT");

            await user.type(input("rr-Txt"), "v=spf1 -all");

            expect(record.Txt).toBe("v=spf1 -all");
            expect(input("rr-Txt").closest(".input-group")).toBeNull();
        });
    });
});

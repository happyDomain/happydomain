// This file is part of the happyDomain (R) project.
// Copyright (c) 2020-2026 happyDomain
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

import { fireEvent, render, screen } from "@testing-library/svelte";
import { beforeAll, describe, expect, it } from "vitest";

import type { dnsTypeTXT } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import type { AbstractDomainVerificationBody } from "$lib/services_bodies";
import { loadTranslations } from "$lib/translations";
import Editor from "./editor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

const origin = { domain: "example.com" } as Domain;

// The value has to be state: a plain object handed to a component is proxied
// on its way in, and the proxy does not write back to the object kept here.
function mount(record?: dnsTypeTXT) {
    const value = $state<AbstractDomainVerificationBody>(
        (record ? { txt: record } : {}) as AbstractDomainVerificationBody,
    );

    render(Editor, { dn: "", origin, value });

    return value;
}

describe("Domain verification editor", () => {
    it("shows the provider told by the record", () => {
        mount({
            Hdr: { Name: "_keybase", Rrtype: 16, Class: 1, Ttl: 3600, Rdlength: 0 },
            Txt: "keybase-site-verification=abc",
        });

        expect((screen.getByLabelText("Provider") as HTMLSelectElement).value).toBe("Keybase");
    });

    it("detects the provider and its label from a pasted value", async () => {
        const value = mount();

        await fireEvent.input(screen.getByLabelText("Value"), {
            target: { value: "did=did:plc:z72i7hdynmk6r22z27h6tvur" },
        });

        expect(value.txt.Hdr.Name).toBe("_atproto");
        expect(value.txt.Txt).toBe("did=did:plc:z72i7hdynmk6r22z27h6tvur");
        expect((screen.getByLabelText("Provider") as HTMLSelectElement).value).toBe("Bluesky");
    });

    it("prefixes the value with the provider picked", async () => {
        const value = mount();

        await fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "Stripe" } });

        expect(value.txt.Hdr.Name).toBe("");
        expect(value.txt.Txt).toBe("stripe-verification=");
    });

    it("offers each provider once", () => {
        mount();

        const names = Array.from(
            (screen.getByLabelText("Provider") as HTMLSelectElement).options,
        ).map((o) => o.value);
        expect(names.filter((n) => n === "Twilio")).toHaveLength(1);
        expect(names).toContain("Amazon SES");
    });
});

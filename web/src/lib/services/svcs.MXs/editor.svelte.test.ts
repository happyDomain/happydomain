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

import type { dnsTypeMX } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import type { SvcsMXsBody } from "$lib/services_bodies";
import { loadTranslations } from "$lib/translations";
import Editor from "./editor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

const origin = { domain: "example.com." } as Domain;

function mx(target: string, preference = 10): dnsTypeMX {
    return {
        Hdr: { Name: "", Rrtype: 15, Class: 1, Ttl: 3600, Rdlength: 0 },
        Preference: preference,
        Mx: target,
    };
}

function mount(records: dnsTypeMX[]) {
    const value = $state({ mx: records } as unknown as SvcsMXsBody);
    const { container } = render(Editor, { dn: "", origin, value });

    return {
        records: () => value.mx as unknown as dnsTypeMX[],
        input: (id: string) => container.querySelector<HTMLInputElement>("#" + id)!,
    };
}

describe("MX editor", () => {
    it("shows the mail servers relative to the zone, or in full", () => {
        const { input } = mount([mx("mail"), mx("mx.provider.net.", 20)]);

        expect(input("mx-0")).toHaveValue("mail");
        expect(input("mx-1")).toHaveValue("mx.provider.net.");
    });

    it("catches a provider's server typed without its final dot", async () => {
        const user = userEvent.setup();
        const { records, input } = mount([mx("")]);

        await user.type(input("mx-0"), "aspmx.l.google.com");
        expect(records()[0].Mx).toBe("aspmx.l.google.com");
        expect(screen.getByText("aspmx.l.google.com.example.com.")).toBeVisible();

        await user.click(screen.getByRole("button", { name: "Use it" }));
        expect(records()[0].Mx).toBe("aspmx.l.google.com.");
    });

    it("accepts a null MX", () => {
        const { input } = mount([mx(".", 0)]);

        expect(input("mx-0")).toHaveValue(".");
        expect(input("mx-0")).not.toHaveClass("is-invalid");
    });

    it("adds a mail server", async () => {
        const user = userEvent.setup();
        const { records, input } = mount([mx("mail")]);

        await user.click(screen.getByRole("button", { name: /New row/ }));
        await user.type(input("mx-1"), "backup");

        expect(records().map((rr) => rr.Mx)).toEqual(["mail", "backup"]);
    });
});

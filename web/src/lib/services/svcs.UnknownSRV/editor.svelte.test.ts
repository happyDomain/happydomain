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

import type { dnsTypeSRV } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import type { SvcsUnknownSRVBody } from "$lib/services_bodies";
import { loadTranslations } from "$lib/translations";
import Editor from "./editor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

const origin = { domain: "example.com." } as Domain;

function srv(target: string): dnsTypeSRV {
    return {
        Hdr: { Name: "_sip._tcp", Rrtype: 33, Class: 1, Ttl: 3600, Rdlength: 0 },
        Priority: 10,
        Weight: 5,
        Port: 5060,
        Target: target,
    };
}

function mount(records: dnsTypeSRV[], readonly = false) {
    const value = $state({ srv: records } as unknown as SvcsUnknownSRVBody);
    const { container } = render(Editor, { dn: "", origin, readonly, value });

    return {
        records: () => value.srv as unknown as dnsTypeSRV[],
        input: (id: string) => container.querySelector<HTMLInputElement>("#" + id)!,
    };
}

describe("SRV editor", () => {
    it("shows a target of the zone as relative", () => {
        const { input } = mount([srv("sip")]);

        expect(input("target-0")).toHaveValue("sip");
        expect(screen.getByText("sip.example.com.")).toBeVisible();
    });

    it("keeps the target typed in full absolute", async () => {
        const user = userEvent.setup();
        const { records, input } = mount([srv("")]);

        await user.type(input("target-0"), "sip.example.com.");

        expect(records()[0].Target).toBe("sip.example.com.");
        expect(records()[0].Port).toBe(5060);
    });

    it("keeps a target outside the zone absolute", async () => {
        const user = userEvent.setup();
        const { records, input } = mount([srv("")]);

        await user.type(input("target-0"), "sip.provider.net.");

        expect(records()[0].Target).toBe("sip.provider.net.");
        expect(screen.getByText(/outside of your zone/)).toBeVisible();
    });

    it("adds a record whose target can be typed", async () => {
        const user = userEvent.setup();
        const { records, input } = mount([srv("sip")]);

        await user.click(screen.getByRole("button", { name: /New row/ }));
        await user.type(input("target-1"), "sip2");

        expect(records().map((rr) => rr.Target)).toEqual(["sip", "sip2"]);
    });

    it("does not offer to change the target when read-only", () => {
        const { input } = mount([srv("sip.provider.net")], true);

        expect(input("target-0")).toHaveAttribute("readonly");
        expect(screen.queryByRole("button", { name: "Use it" })).toBeNull();
    });
});

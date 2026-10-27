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
import { beforeAll, describe, expect, it, vi } from "vitest";

vi.mock("$lib/api/provider_specs", () => ({ getProviderSpec: vi.fn(() => new Promise(() => {})) }));

import { getRrtype, newRR, type dnsRR } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import type { SvcsAliasBody } from "$lib/services_bodies";
import { loadTranslations } from "$lib/translations";
import Editor from "./editor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

const origin = { domain: "example.com." } as Domain;

function mount(rrtype: string, target: string) {
    const record = newRR("www", getRrtype(rrtype));
    if (record.Data) (record.Data as { Target: string }).Target = target;
    else record.Target = target;

    const value = $state({ record } as SvcsAliasBody);
    const { container } = render(Editor, { dn: "www", origin, value });

    return {
        record: () => value.record as dnsRR,
        input: () => container.querySelector<HTMLInputElement>("#spec-target-target")!,
    };
}

describe("Alias editor", () => {
    it("edits the target of a CNAME as a name of the zone", async () => {
        const user = userEvent.setup();
        const { record, input } = mount("CNAME", "");

        await user.type(input(), "web");

        expect(record().Target).toBe("web");
        expect(screen.getByText("web.example.com.")).toBeVisible();
    });

    it("shows the target of a CNAME outside the zone in full", () => {
        const { input } = mount("CNAME", "example.github.io.");

        expect(input()).toHaveValue("example.github.io.");
        expect(screen.getByText(/outside of your zone/)).toBeVisible();
    });

    it("keeps a plain text input for an Azure resource id", () => {
        const id =
            "/subscriptions/0000/resourceGroups/g/providers/Microsoft.Network/dnszones/example.com/A/www";
        const { input } = mount("AZURE_ALIAS", id);

        expect(input()).toHaveValue(id);
        expect(input().classList.contains("font-monospace")).toBe(false);
        expect(screen.queryByText(/example\.com\.$/)).toBeNull();
    });
});

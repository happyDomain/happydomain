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

import type { Domain } from "$lib/model/domain";
import type { AbstractEmailAutoConfigBody } from "$lib/services_bodies";
import { loadTranslations } from "$lib/translations";
import Editor from "./editor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

const origin = { domain: "example.com." } as Domain;

describe("E-mail autoconfiguration editor", () => {
    it("stores the hostnames in full, whatever the zone", async () => {
        const user = userEvent.setup();
        const value = $state({} as AbstractEmailAutoConfigBody);
        const { container } = render(Editor, { dn: "", origin, value });

        const host = container.querySelector<HTMLInputElement>("#spec-incomingHost-incomingHost")!;
        await user.type(host, "imap.example.com");

        expect(value.incomingSRV?.Target).toBe("imap.example.com.");
        // No zone to append: the name is not shown with a suffix.
        expect(screen.queryByText(".example.com")).toBeNull();
    });

    it("explains why a hostname cannot exist", async () => {
        const user = userEvent.setup();
        const value = $state({} as AbstractEmailAutoConfigBody);
        const { container } = render(Editor, { dn: "", origin, value });

        const host = container.querySelector<HTMLInputElement>("#spec-outgoingHost-outgoingHost")!;
        await user.type(host, "smtp..example.com");

        expect(host).toHaveClass("is-invalid");
        expect(screen.getByText(/two consecutive dots/)).toBeVisible();
    });
});

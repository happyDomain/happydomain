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
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import { providers } from "$lib/stores/providers";
import { loadTranslations } from "$lib/translations";
import RecordLine from "./RecordLine.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    providers.set(undefined);
});

function setup(readonly: boolean) {
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    return {
        dn: "www",
        origin: { id: "d1", domain: "example.com.", id_provider: "p1" } as unknown as Domain,
        rr: {
            Hdr: { Name: "www", Rrtype: 16, Class: 1, Ttl: 300, Rdlength: 0 },
            Txt: "hello",
        } as never,
    };
}

describe("RecordLine", () => {
    it("offers to edit the record through a writable provider", () => {
        render(RecordLine, setup(false));
        expect(screen.getByRole("button")).toBeEnabled();
    });

    it("hides the edit button of the record through a read-only provider", () => {
        render(RecordLine, setup(true));
        expect(screen.queryByRole("button")).toBeNull();
    });
});

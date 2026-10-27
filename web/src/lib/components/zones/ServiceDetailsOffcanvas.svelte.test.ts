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

import { cleanup, render, screen } from "@testing-library/svelte";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

const checkers = vi.hoisted(() => ({ listScopedCheckers: vi.fn() }));
vi.mock("$lib/api/checkers", async (importOriginal) => ({
    ...(await importOriginal<typeof import("$lib/api/checkers")>()),
    ...checkers,
}));

import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import type { ServiceWithValue } from "$lib/model/service.svelte";
import { providers } from "$lib/stores/providers";
import { thisZone } from "$lib/stores/thiszone";
import { makeZone } from "$lib/test-utils/fixtures";
import { loadTranslations } from "$lib/translations";
import { domainZoneEditor, zoneEditorContext, type ZoneEditor } from "./editor";
import ServiceDetailsOffcanvas, { controls } from "./ServiceDetailsOffcanvas.svelte";

const domain = { id: "d1", domain: "example.com.", id_provider: "p1" } as unknown as Domain;

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    cleanup();
    vi.resetAllMocks();
    providers.set(undefined);
    thisZone.set(null);
});

function open(readonly: boolean, context?: Map<symbol, ZoneEditor>) {
    providers.set([
        { _id: "p1", _srctype: "DDNSServer", _readonly: readonly } as unknown as Provider,
    ]);
    thisZone.set(makeZone({ id: "z1" }));
    render(ServiceDetailsOffcanvas, { props: { domain, selectedHistory: "z1" }, context });
    controls.Open({
        _id: "s1",
        _svctype: "svcs.Alias",
        _domain: "www",
        _ttl: 300,
    } as unknown as ServiceWithValue);
}

describe("ServiceDetailsOffcanvas", () => {
    it("offers to edit and to delete the service through a writable provider", async () => {
        open(false);
        expect(await screen.findByRole("button", { name: /Edit/ })).toBeEnabled();
        expect(screen.getByRole("button", { name: /Delete/ })).toBeEnabled();
        expect(screen.getByLabelText("Default TTL")).toBeEnabled();
        expect(checkers.listScopedCheckers).toHaveBeenCalled();
    });

    it("keeps the details but changes nothing through a read-only provider", async () => {
        open(true);
        const edit = await screen.findByRole("button", { name: /Edit/ });
        expect(edit).toBeDisabled();
        expect(edit.parentElement).toHaveAttribute("title", expect.stringMatching(/read-only/));
        expect(screen.queryByRole("button", { name: /Delete/ })).toBeNull();
        expect(screen.getByLabelText("Default TTL")).toBeDisabled();
    });

    it("changes nothing, and offers no check, in a zone the page made read-only", async () => {
        const editor: ZoneEditor = {
            ...domainZoneEditor(() => domain),
            isDomain: false,
            readOnly: true,
        };
        open(false, zoneEditorContext(editor));

        const edit = await screen.findByRole("button", { name: /Edit/ });
        expect(edit).toBeDisabled();
        expect(screen.queryByRole("button", { name: /Delete/ })).toBeNull();
        expect(screen.getByLabelText("Default TTL")).toBeDisabled();
        expect(checkers.listScopedCheckers).not.toHaveBeenCalled();
    });
});

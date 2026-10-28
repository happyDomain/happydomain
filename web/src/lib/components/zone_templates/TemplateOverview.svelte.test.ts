// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
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

import { zoneEditorContext } from "$lib/components/zones/editor";
import type { User } from "$lib/model/user";
import { templateAsZone, type ZoneTemplate } from "$lib/model/zone_template";
import { thisZone } from "$lib/stores/thiszone";
import { userSession } from "$lib/stores/usersession";
import { thisTemplate } from "$lib/stores/zone_templates";
import { makeService } from "$lib/test-utils/fixtures";
import { stubWebAnimations } from "$lib/test-utils/animations";
import { loadTranslations } from "$lib/translations";
import type { ZoneTemplateBackend } from "./backend";
import { templateZoneEditor } from "./editor";
import TemplateOverview from "./TemplateOverview.svelte";

const template: ZoneTemplate = {
    id: "t1",
    userId: "u1",
    name: "Mail",
    description: "Our mail servers",
    default_ttl: 3600,
    services: {},
} as unknown as ZoneTemplate;

beforeAll(async () => {
    await loadTranslations("en", "/");
    vi.stubGlobal("matchMedia", () => ({ matches: false }));
    stubWebAnimations();
});

afterEach(() => {
    cleanup();
    thisTemplate.set(null);
    thisZone.set(null);
    userSession.set({} as User);
});

function setup(tpl: ZoneTemplate, readOnly: boolean) {
    userSession.set({ settings: {} } as unknown as User);
    thisTemplate.set(tpl);
    thisZone.set(templateAsZone(tpl));
    // As the layout of the templates does.
    const context = zoneEditorContext(
        templateZoneEditor(
            () => ({}) as ZoneTemplateBackend,
            () => readOnly,
            () => {},
        ),
    );
    render(TemplateOverview, { props: { template: tpl, readOnly }, context });
}

describe("TemplateOverview", () => {
    it("shows the template, and its apex while empty", () => {
        setup(template, false);

        expect(screen.getByRole("heading", { name: "Mail" })).toBeVisible();
        expect(screen.getByText("Our mail servers")).toBeVisible();
        expect(screen.getByRole("link", { name: /^@/ })).toHaveAttribute(
            "href",
            "/templates/t1/%40",
        );
    });

    it("leads to the page of each subdomain", () => {
        setup(
            {
                ...template,
                services: {
                    "": [makeService("svcs.MXs", {}, { _id: "s1" })],
                    mail: [makeService("abstract.Server", {}, { _id: "s2", _domain: "mail" })],
                },
            } as unknown as ZoneTemplate,
            false,
        );

        expect(screen.getByRole("link", { name: /^mail\.example\.com\./ })).toHaveAttribute(
            "href",
            "/templates/t1/mail",
        );
    });

    it("tells a user a global template is only to apply", () => {
        setup({ ...template, userId: undefined } as ZoneTemplate, true);

        expect(screen.getAllByText(/Set up by the administrators/).length).toBeGreaterThan(0);
        expect(screen.getByText("Offered to everyone")).toBeVisible();
    });

    it("follows the changes saved to the template", () => {
        setup(template, false);
        thisTemplate.set({ ...template, name: "Mail servers" } as ZoneTemplate);

        return vi.waitFor(() =>
            expect(screen.getByRole("heading", { name: "Mail servers" })).toBeVisible(),
        );
    });
});

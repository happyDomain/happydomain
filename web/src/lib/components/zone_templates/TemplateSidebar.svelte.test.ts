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
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { zoneEditorContext } from "$lib/components/zones/editor";
import { controls as ctrlNewSubdomain } from "$lib/components/zones/NewSubdomainPath.svelte";
import { templateAsZone, type ZoneTemplate } from "$lib/model/zone_template";
import { thisZone } from "$lib/stores/thiszone";
import { zoneTemplates } from "$lib/stores/zone_templates";
import { makeService } from "$lib/test-utils/fixtures";
import { loadTranslations } from "$lib/translations";
import type { ZoneTemplateBackend } from "./backend";
import { templateZoneEditor } from "./editor";
import { controls as ctrlTemplateDetails } from "./ModalTemplateDetails.svelte";
import TemplateSidebar from "./TemplateSidebar.svelte";

function tpl(id: string, name: string, userId?: string): ZoneTemplate {
    return {
        id,
        name,
        userId,
        default_ttl: 3600,
        services: {
            "": [makeService("svcs.MXs", {}, { _id: "s1" })],
            mail: [makeService("abstract.Server", {}, { _id: "s2", _domain: "mail" })],
        },
    } as unknown as ZoneTemplate;
}

const mine = tpl("t1", "Mail", "u1");
const global = tpl("t2", "Google Workspace");

beforeAll(async () => {
    await loadTranslations("en", "/");
    // jsdom has none, and the list of subdomains brings the current one into view.
    Element.prototype.scrollIntoView = () => {};
});

afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
    thisZone.set(null);
    zoneTemplates.set(undefined);
});

function setup(template: ZoneTemplate, readOnly: boolean, subdomain?: string, serviceid?: string) {
    zoneTemplates.set([mine, global]);
    thisZone.set(templateAsZone(template));
    const onselect = vi.fn();
    const ondelete = vi.fn();
    // As the layout of the templates does.
    const context = zoneEditorContext(
        templateZoneEditor(
            () => ({}) as ZoneTemplateBackend,
            () => readOnly,
            () => {},
        ),
    );
    render(TemplateSidebar, {
        props: {
            template,
            readOnly,
            subdomain,
            serviceid,
            deleteInProgress: false,
            onselect,
            ondelete,
        },
        context,
    });
    return { onselect, ondelete };
}

describe("TemplateSidebar", () => {
    it("switches to another template", async () => {
        const { onselect } = setup(mine, false);

        const select = screen.getByRole("combobox", { name: "Zone templates" });
        expect(select).toHaveValue("t1");
        expect(screen.getByRole("group", { name: "My templates" })).toBeInTheDocument();
        expect(screen.getByRole("group", { name: "Offered to everyone" })).toBeInTheDocument();

        await userEvent.setup().selectOptions(select, "Google Workspace");
        expect(onselect).toHaveBeenCalledWith("t2");
    });

    it("leads to the overview and to the page of each subdomain", () => {
        setup(mine, false);

        expect(screen.getByTitle("All my templates")).toHaveAttribute("href", "/templates");
        expect(screen.getByRole("link", { name: /details · subdomains/ })).toHaveAttribute(
            "aria-current",
            "page",
        );

        expect(screen.getByRole("link", { name: "example.com." })).toHaveAttribute(
            "href",
            "/templates/t1/%40",
        );
        expect(screen.getByRole("link", { name: "mail.example.com." })).toHaveAttribute(
            "href",
            "/templates/t1/mail",
        );
    });

    it("goes back to the overview from a subdomain", () => {
        setup(mine, false, "mail");

        expect(screen.getByTitle("Template overview")).toHaveAttribute("href", "/templates/t1");
        expect(screen.getByRole("link", { name: /details · subdomains/ })).not.toHaveAttribute(
            "aria-current",
        );
        expect(screen.getByRole("link", { name: "mail.example.com." })).toHaveClass("fw-bold");
    });

    it("offers to add a subdomain and to manage the template", async () => {
        const user = userEvent.setup();
        const openSubdomain = vi.spyOn(ctrlNewSubdomain, "Open").mockImplementation(() => {});
        const openDetails = vi.spyOn(ctrlTemplateDetails, "Open").mockImplementation(() => {});
        const { ondelete } = setup(mine, false);

        await user.click(screen.getByRole("button", { name: /Add a subdomain/ }));
        expect(openSubdomain).toHaveBeenCalled();

        await user.click(screen.getByRole("button", { name: /Actions/ }));
        await user.click(await screen.findByText("Edit the details"));
        expect(openDetails).toHaveBeenCalled();

        await user.click(screen.getByRole("button", { name: /Actions/ }));
        await user.click(await screen.findByText("Delete the template"));
        expect(ondelete).toHaveBeenCalled();
    });

    it("disables the changes to a global template, telling why", async () => {
        setup(global, true);

        const add = screen.getByRole("button", { name: /Add a subdomain/ });
        expect(add).toBeDisabled();
        expect(add.parentElement).toHaveAttribute("title", expect.stringMatching(/administrators/));

        await userEvent.setup().click(screen.getByRole("button", { name: /Actions/ }));
        expect(await screen.findByText("Edit the details")).toHaveClass("disabled");
        expect(screen.getByText("Delete the template")).toHaveClass("disabled");
    });

    it("shows the services of the subdomain on the page of a service", () => {
        setup(mine, false, "mail", "s2");

        expect(screen.queryByRole("button", { name: /Add a subdomain/ })).toBeNull();
        expect(screen.getByTitle("Go to the zone")).toHaveAttribute("href", "/templates/t1/mail");
        expect(screen.getByRole("link", { name: "example.com." })).toHaveAttribute(
            "href",
            "/templates/t1/%40",
        );
        expect(screen.getByRole("link", { name: /Server/ })).toHaveAttribute(
            "href",
            "/templates/t1/mail/s2",
        );
    });
});

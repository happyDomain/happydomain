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

import { render, screen } from "@testing-library/svelte";
import { beforeAll, describe, expect, it } from "vitest";

import { loadTranslations } from "$lib/translations";
import DomainInfoDisplay from "./DomainInfoDisplay.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

// The lines of an address, without the markers Svelte leaves in the markup.
function addressLines(card: HTMLElement): string[] | undefined {
    return card
        .querySelector("address")
        ?.innerHTML.replace(/<!---->/g, "")
        .split("<br>");
}

function contactCards(container: HTMLElement): HTMLElement[] {
    return [...container.querySelectorAll<HTMLElement>(".bi-person-vcard")].map(
        (icon) => icon.closest(".card") as HTMLElement,
    );
}

describe("DomainInfoDisplay", () => {
    it("shows the contacts the registry publishes, registrant first", () => {
        const { container } = render(DomainInfoDisplay, {
            domain: "example.com",
            info: {
                name: "example.com",
                contacts: {
                    tech: { name: "Tech Person", email: "tech@example.com" },
                    registrant: { organization: "Example Inc.", city: "Paris", country: "FR" },
                    admin: {},
                    reseller: { name: "Reseller Ltd" },
                },
            },
        });

        const titles = contactCards(container).map((c) =>
            c.querySelector(".card-title")?.textContent?.trim(),
        );
        expect(titles).toEqual(["Registrant", "Technical contact", "reseller"]);
        expect(screen.queryByText("Administrative contact")).toBeNull();
        expect(screen.getByText("Example Inc.")).toBeVisible();
        expect(screen.getByText("tech@example.com")).toBeVisible();
    });

    it("leaves out a contact telling nothing", () => {
        const { container } = render(DomainInfoDisplay, {
            domain: "example.com",
            info: {
                name: "example.com",
                contacts: {
                    admin: { name: "", organization: "", email: "", country: "" },
                },
            },
        });

        expect(contactCards(container)).toHaveLength(0);
    });

    it("shows no contact when the registry publishes none", () => {
        const { container } = render(DomainInfoDisplay, {
            domain: "example.com",
            info: { name: "example.com" },
        });

        expect(contactCards(container)).toHaveLength(0);
    });

    it("writes the address of a contact on as many lines as it has parts", () => {
        const { container } = render(DomainInfoDisplay, {
            domain: "example.com",
            info: {
                name: "example.com",
                contacts: {
                    registrant: {
                        street: "1 rue de la Paix",
                        postal_code: "75002",
                        city: "Paris",
                        province: "Île-de-France",
                        country: "FR",
                        phone: "+33.123456789",
                    },
                    tech: { name: "Tech Person", country: "DE" },
                },
            },
        });

        const [registrant, tech] = contactCards(container);
        expect(addressLines(registrant)).toEqual([
            "1 rue de la Paix",
            "75002 Paris, Île-de-France",
            "FR",
        ]);
        expect(registrant).toHaveTextContent("+33.123456789");
        expect(addressLines(tech)).toEqual(["DE"]);
    });

    it("tells the name and the statuses of the domain at the top", () => {
        const { container } = render(DomainInfoDisplay, {
            domain: "example.com",
            info: { name: "example.com", status: ["clientTransferProhibited"] },
        });

        expect(container.querySelector("h2")).toHaveTextContent("example.com");
        expect(screen.getByText("clientTransferProhibited")).toBeVisible();
    });

    it("lists the statuses with their description among the cards, without summary", () => {
        const { container } = render(DomainInfoDisplay, {
            domain: "example.com",
            summary: false,
            info: { name: "example.com", status: ["clientTransferProhibited", "someStatus"] },
        });

        expect(container.querySelector("h2")).toBeNull();
        expect(screen.getByText("Transfer to another registrar is locked.")).toBeVisible();
        const unknown = screen.getByText("someStatus").closest(".list-group-item");
        expect(unknown?.querySelector(".text-muted")).toBeNull();
    });
});

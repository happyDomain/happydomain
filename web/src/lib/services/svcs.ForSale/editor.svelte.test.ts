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

import { fireEvent, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import type { dnsTypeTXT } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import { loadTranslations } from "$lib/translations";
import ForSaleEditor from "./editor.svelte";
import { FORSALE_LABEL, FORSALE_VERSION, type ForSaleValue } from "./model.svelte";

// Without this, every label reads back as its translation key.
beforeAll(async () => {
    await loadTranslations("en", "/");
});

// The forms below are ours, so the automatic cleanup knows nothing about them.
afterEach(() => {
    document.querySelectorAll("form").forEach((form) => form.remove());
});

const origin = { domain: "example.com" } as Domain;

function txtRecord(txt: string): dnsTypeTXT {
    return {
        Hdr: { Name: FORSALE_LABEL, Rrtype: 16, Class: 1, Ttl: 3600, Rdlength: 0 },
        Txt: txt,
    };
}

/**
 * Mount the editor inside a form, the way the service page does, and hand back
 * the RRset it edits. The editor mutates that array in place, so asserting on
 * it is asserting on what the page would save. It has to be state: a plain
 * object handed to a component is proxied on its way in, and the proxy does not
 * write back to the object the test kept.
 */
function mount(records?: Array<dnsTypeTXT>, opts: { readonly?: boolean } = {}) {
    const value = $state<ForSaleValue>(records ? { txt: records } : {});

    const form = document.createElement("form");
    // Submitting is as far as the test goes: jsdom has no page to navigate to.
    form.addEventListener("submit", (e) => e.preventDefault());
    document.body.appendChild(form);

    render(
        ForSaleEditor,
        { dn: "example.com", origin, readonly: opts.readonly ?? false, value },
        { baseElement: form },
    );

    return {
        form,
        /** The values of the RRset, in order, as the page would send them. */
        published: () => (Array.isArray(value.txt) ? value.txt : [value.txt!]).map((r) => r!.Txt),
    };
}

describe("For Sale editor", () => {
    it("shows nothing to edit and keeps a bare record when there is nothing to sell", () => {
        const svc = mount();

        expect(screen.queryAllByRole("textbox")).toHaveLength(0);
        expect(svc.published()).toEqual([FORSALE_VERSION]);
    });

    it("adds a price with the EUR default", async () => {
        const user = userEvent.setup();
        const svc = mount();

        await user.click(screen.getByRole("button", { name: "Price" }));

        expect(screen.getByPlaceholderText("USD")).toHaveValue("EUR");
        expect(screen.getByPlaceholderText("750")).toHaveValue("");
        expect(svc.published()).toEqual(["v=FORSALE1;fval=EUR"]);
    });

    it("adds a message and captures what is typed in it", async () => {
        const user = userEvent.setup();
        const svc = mount();

        await user.click(screen.getByRole("button", { name: "Message" }));
        await user.type(screen.getByPlaceholderText("Call for info."), "Cheap!");

        expect(svc.published()).toEqual(["v=FORSALE1;ftxt=Cheap!"]);
    });

    it("replaces the placeholder bare record instead of adding alongside it", async () => {
        const user = userEvent.setup();
        const svc = mount([txtRecord(FORSALE_VERSION)]);

        await user.click(screen.getByRole("button", { name: "Link" }));

        expect(svc.published()).toHaveLength(1);
        expect(svc.published()[0]).toBe("v=FORSALE1;furi=");
    });

    it("flags a message longer than 239 octets", () => {
        mount([txtRecord(FORSALE_VERSION + "ftxt=" + "a".repeat(240))]);

        expect(screen.getByText("240 octets, 239 maximum")).toBeVisible();
        expect(screen.getByDisplayValue("a".repeat(240))).toHaveClass("is-invalid");
    });

    it("flags a broker code longer than 239 octets", () => {
        mount([txtRecord(FORSALE_VERSION + "fcod=" + "a".repeat(240))]);

        expect(screen.getByText("240 octets, 239 maximum")).toBeVisible();
        expect(screen.getByDisplayValue("a".repeat(240))).toHaveClass("is-invalid");
    });

    it("labels each field so a screen reader can tell the rows apart", () => {
        mount([
            txtRecord(FORSALE_VERSION + "fval=USD750"),
            txtRecord(FORSALE_VERSION + "ftxt=Call for info."),
        ]);

        // An input with a datalist counts as a combobox to accessibility tools.
        expect(screen.getByRole("combobox", { name: "Price Currency" })).toHaveValue("USD");
        expect(screen.getByRole("textbox", { name: "Price Amount" })).toHaveValue("750");
        expect(screen.getByRole("textbox", { name: "Message" })).toHaveValue("Call for info.");
    });

    it("adds two different tags in the same session without one replacing the other", async () => {
        const user = userEvent.setup();
        const svc = mount();

        await user.click(screen.getByRole("button", { name: "Price" }));
        await user.click(screen.getByRole("button", { name: "Link" }));

        expect(svc.published()).toEqual(["v=FORSALE1;fval=EUR", "v=FORSALE1;furi="]);
    });

    it("keeps a broken record editable and republishes the fix", async () => {
        const svc = mount([txtRecord(FORSALE_VERSION + "garbage")]);

        const input = screen.getByDisplayValue("garbage");
        expect(input).toHaveClass("is-invalid");

        // Typed one character at a time, clearing the field would empty the
        // content and make the row disappear before the fix is typed in.
        await fireEvent.input(input, { target: { value: "ftxt=fixed" } });

        expect(svc.published()).toEqual(["v=FORSALE1;ftxt=fixed"]);
    });

    it("removes an entry and falls back to a bare record when it was the last one", async () => {
        const user = userEvent.setup();
        const svc = mount([txtRecord(FORSALE_VERSION + "fval=USD750")]);

        await user.click(screen.getByRole("button", { name: "Delete" }));

        expect(svc.published()).toEqual([FORSALE_VERSION]);
    });

    it("removes just the entry that was clicked, keeping the others in order", async () => {
        const user = userEvent.setup();
        const svc = mount([
            txtRecord(FORSALE_VERSION + "fval=USD750"),
            txtRecord(FORSALE_VERSION + "ftxt=Call for info."),
            txtRecord(FORSALE_VERSION + "furi=https://example.com/fs"),
        ]);

        const buttons = screen.getAllByRole("button", { name: "Delete" });
        await user.click(buttons[1]);

        expect(svc.published()).toEqual([
            "v=FORSALE1;fval=USD750",
            "v=FORSALE1;furi=https://example.com/fs",
        ]);
    });

    it("hides the add and delete controls and locks the fields in readonly mode", () => {
        mount([txtRecord(FORSALE_VERSION + "fval=USD750")], { readonly: true });

        expect(screen.queryByRole("button", { name: "Delete" })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Price" })).not.toBeInTheDocument();
        for (const textbox of screen.getAllByRole("textbox")) {
            expect(textbox).toHaveAttribute("readonly");
        }
    });
});

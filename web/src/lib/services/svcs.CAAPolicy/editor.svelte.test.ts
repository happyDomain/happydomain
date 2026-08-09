import { render, screen, within } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it } from "vitest";

import type { dnsTypeCAA } from "$lib/dns_rr";
import type { Domain } from "$lib/model/domain";
import { loadTranslations } from "$lib/translations";
import CAAEditor from "./editor.svelte";
import { newCAARecord } from "./model.svelte";

// Without this, every label reads back as its translation key.
beforeAll(async () => {
    await loadTranslations("en", "/");
});

// The forms below are ours, so the automatic cleanup knows nothing about them.
afterEach(() => {
    document.querySelectorAll("form").forEach((form) => form.remove());
});

const origin = { domain: "example.com" } as Domain;

/**
 * Mount the editor inside a form, the way the service page does, and hand back
 * the RRset it edits. The editor mutates that array in place, so asserting on
 * it is asserting on what the page would save. It has to be state: a plain
 * object handed to a component is proxied on its way in, and the proxy does not
 * write back to the object the test kept.
 */
function mount(records: Array<dnsTypeCAA> = []) {
    const value = $state({ caa: records });

    const form = document.createElement("form");
    // Submitting is as far as the test goes: jsdom has no page to navigate to.
    form.addEventListener("submit", (e) => e.preventDefault());
    document.body.appendChild(form);

    render(CAAEditor, { dn: "example.com", origin, value }, { baseElement: form });

    return {
        form,
        /** The values of the RRset, in order, as the page would send them. */
        published: () => value.caa.map((record) => `${record.Tag} ${record.Value}`),
    };
}

/** The sections repeat the same labels, so queries have to be scoped to one. */
function kind(title: string) {
    return within(screen.getByRole("region", { name: title }));
}

function summary() {
    return within(screen.getByRole("region", { name: "What this policy says today" }));
}

describe("CAA policy editor", () => {
    it("mounts on an empty policy", () => {
        const policy = mount();

        expect(
            kind("TLS certificates").getByRole("radio", { name: /Any authority/ }),
        ).toBeChecked();
        expect(policy.published()).toEqual([]);
    });

    it("tells wildcard certificates follow the TLS rule", () => {
        mount();

        expect(
            kind("Wildcard TLS certificates").getByRole("radio", {
                name: /Same as TLS certificates/,
            }),
        ).toBeChecked();
    });

    it("names in its summary the authorities a restricted policy allows", () => {
        mount([newCAARecord("example.com", "issue", "letsencrypt.org")]);

        expect(kind("TLS certificates").getByRole("radio", { name: /Only chosen/ })).toBeChecked();
        expect(summary().getByText("Let's Encrypt")).toBeVisible();
    });

    it("publishes the deny marker when nobody may issue", async () => {
        const user = userEvent.setup();
        const policy = mount();

        await user.click(kind("S/MIME certificates").getByRole("radio", { name: /Nobody/ }));

        expect(policy.published()).toEqual(["issuemail ;"]);
        expect(summary().getByText("Nobody")).toBeVisible();
    });

    it("drops the records of a kind handed back to any authority", async () => {
        const user = userEvent.setup();
        const policy = mount([newCAARecord("example.com", "issue", ";")]);

        await user.click(kind("TLS certificates").getByRole("radio", { name: /Any authority/ }));

        expect(policy.published()).toEqual([]);
    });

    it("keeps an authority left in the add row on submit", async () => {
        const user = userEvent.setup();
        const policy = mount([newCAARecord("example.com", "issue", "letsencrypt.org")]);

        // The add row comes last: a user who took the save button for the way
        // to confirm their choice leaves it filled in.
        const rows = kind("TLS certificates").getAllByRole("combobox");
        await user.selectOptions(rows[rows.length - 1], "sectigo.com");

        policy.form.requestSubmit();

        expect(policy.published()).toEqual(["issue letsencrypt.org", "issue sectigo.com"]);
    });

    it("leaves an untouched add row out of the records", () => {
        const policy = mount([newCAARecord("example.com", "issue", "letsencrypt.org")]);

        policy.form.requestSubmit();

        expect(policy.published()).toEqual(["issue letsencrypt.org"]);
    });
});

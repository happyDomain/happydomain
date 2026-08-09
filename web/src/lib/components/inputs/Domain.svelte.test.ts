import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import type { Domain } from "$lib/model/domain";
import { loadTranslations } from "$lib/translations";
import DomainInput from "./Domain.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    document.querySelectorAll("form").forEach((form) => form.remove());
});

const origin = { domain: "example.com" } as Domain;

function mount(value = "", onValidityChanged?: (valid: boolean | undefined) => void) {
    const form = document.createElement("form");
    document.body.appendChild(form);

    render(
        DomainInput,
        {
            origin,
            value,
            $$events: {
                ...(onValidityChanged
                    ? {
                          "validity-changed": (event: CustomEvent<boolean | undefined>) =>
                              onValidityChanged(event.detail),
                      }
                    : {}),
            },
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } as any,
        { baseElement: form },
    );

    return { form };
}

describe("Domain input", () => {
    it("suggests the origin as a suffix on an empty value", () => {
        mount();

        expect(screen.getByText("example.com")).toBeVisible();
    });

    it("appends the origin to a bare subdomain", async () => {
        const user = userEvent.setup();
        mount();

        await user.type(screen.getByPlaceholderText("new.subdomain"), "www");

        expect(screen.getByText(".example.com")).toBeVisible();
    });

    it("hides the suffix once the value already ends with the origin", async () => {
        const user = userEvent.setup();
        mount();

        await user.type(screen.getByPlaceholderText("new.subdomain"), "www.example.com");

        expect(screen.queryByText(".example.com")).not.toBeInTheDocument();
    });

    it("marks a valid subdomain as valid", async () => {
        const user = userEvent.setup();
        mount();

        await user.type(screen.getByPlaceholderText("new.subdomain"), "www");

        expect(screen.getByPlaceholderText("new.subdomain")).toHaveClass("is-valid");
    });

    it("marks an illegal subdomain as invalid", async () => {
        const user = userEvent.setup();
        mount();

        await user.type(screen.getByPlaceholderText("new.subdomain"), "in valid");

        expect(screen.getByPlaceholderText("new.subdomain")).toHaveClass("is-invalid");
    });

    it("dispatches validity-changed whenever validity is (re-)computed", async () => {
        const user = userEvent.setup();
        const onValidityChanged = vi.fn();
        mount("", onValidityChanged);

        // Fires once up front for the initial (empty) value. The dispatched
        // CustomEvent's detail normalizes an undefined value to null.
        expect(onValidityChanged).toHaveBeenCalledWith(null);

        onValidityChanged.mockClear();
        await user.type(screen.getByPlaceholderText("new.subdomain"), "www");

        expect(onValidityChanged).toHaveBeenLastCalledWith(true);
    });

    it("validates a value that was typed out fully, already ending with the origin", async () => {
        const user = userEvent.setup();
        mount();

        // No ".example.com" suggestion is appended here since the value
        // already ends with the origin; this exercises the branch that
        // re-validates the value as typed rather than value + origin.
        await user.type(screen.getByPlaceholderText("new.subdomain"), "www.example.com");

        expect(screen.queryByText(".example.com")).not.toBeInTheDocument();
        expect(screen.getByPlaceholderText("new.subdomain")).toHaveClass("is-valid");
    });
});

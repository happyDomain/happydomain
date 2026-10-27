import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { loadTranslations } from "$lib/translations";
import MboxInput from "./Mbox.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

function mount(value: string | undefined) {
    const state = $state({ value });

    render(MboxInput, {
        id: "rname",
        origin: "example.com.",
        get value() {
            return state.value;
        },
        set value(v: string | undefined) {
            state.value = v;
        },
    });

    return { current: () => state.value };
}

describe("Mbox input", () => {
    it("shows a mailbox of the zone as its local part, followed by the domain", () => {
        mount("john\\.doe");

        expect(screen.getByRole("textbox")).toHaveValue("john.doe");
        expect(screen.getByText("@example.com")).toBeVisible();
    });

    it("shows a mailbox of another domain as a full address", () => {
        mount("hostmaster.other.org.");

        expect(screen.getByRole("textbox")).toHaveValue("hostmaster@other.org");
        expect(screen.queryByText("@example.com")).toBeNull();
    });

    it("turns a typed local part into a mailbox of the zone", async () => {
        const user = userEvent.setup();
        const state = mount("");

        await user.type(screen.getByRole("textbox"), "john.doe");

        expect(state.current()).toBe("john\\.doe");
    });

    it("drops the domain suffix once an @ is typed", async () => {
        const user = userEvent.setup();
        const state = mount("");

        await user.type(screen.getByRole("textbox"), "hostmaster@other.org");

        expect(screen.queryByText("@example.com")).toBeNull();
        expect(state.current()).toBe("hostmaster.other.org.");
    });

    it("keeps the zone's domain when it is typed", async () => {
        const user = userEvent.setup();
        const state = mount("");

        const input = screen.getByRole("textbox");
        await user.type(input, "dnsmaster@example.com");
        await user.tab();

        expect(state.current()).toBe("dnsmaster.example.com.");
        expect(input).toHaveValue("dnsmaster@example.com");
    });

    it("flags what is not an address and keeps the last valid mailbox", async () => {
        const user = userEvent.setup();
        const state = mount("dnsmaster");

        const input = screen.getByRole("textbox");
        await user.clear(input);
        await user.type(input, "john doe");

        expect(input).toHaveClass("is-invalid");
        expect(state.current()).toBe("john");
    });
});

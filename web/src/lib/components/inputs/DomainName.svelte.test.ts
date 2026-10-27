import { tick } from "svelte";
import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { loadTranslations } from "$lib/translations";
import DomainNameInput from "./DomainName.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

function mount(
    value: string | undefined,
    origin = "example.com.",
    within = false,
    opts: {
        readonly?: boolean;
        required?: boolean;
        placeholder?: string;
        onfocus?: () => void;
        onblur?: () => void;
    } = {},
) {
    const state = $state({ value, valid: undefined as boolean | undefined });

    render(DomainNameInput, {
        id: "target",
        origin,
        within,
        ...opts,
        get valid() {
            return state.valid;
        },
        set valid(v: boolean | undefined) {
            state.valid = v;
        },
        get value() {
            return state.value;
        },
        set value(v: string | undefined) {
            state.value = v;
        },
    });

    return {
        current: () => state.value,
        valid: () => state.valid,
        // Changes the value from outside, as a reset or another field would.
        set: (v: string | undefined) => (state.value = v),
    };
}

describe("Domain name input", () => {
    it("shows a name of the zone as relative, followed by the zone", () => {
        mount("ns1");

        expect(screen.getByRole("textbox")).toHaveValue("ns1");
        expect(screen.getByText(".example.com.")).toBeVisible();
        expect(screen.getByText("ns1.example.com.")).toBeVisible();
    });

    it("shows a name outside the zone in full, without the suffix", () => {
        mount("ns1.ovh.net.");

        expect(screen.getByRole("textbox")).toHaveValue("ns1.ovh.net.");
        expect(screen.queryByText(".example.com")).toBeNull();
        expect(screen.getByText(/outside of your zone/)).toBeVisible();
    });

    it("stores what is typed relative to the zone", async () => {
        const user = userEvent.setup();
        const state = mount("");

        await user.type(screen.getByRole("textbox"), "mail");

        expect(state.current()).toBe("mail");
    });

    it("keeps the zone typed without its final dot relative, as a zone file", async () => {
        const user = userEvent.setup();
        const state = mount("");

        const input = screen.getByRole("textbox");
        await user.type(input, "mail.example.com");
        expect(state.current()).toBe("mail.example.com");
        expect(screen.getByText("mail.example.com.example.com.")).toBeVisible();
        expect(screen.getByText(/Did you mean mail\.example\.com\./)).toBeVisible();

        await user.tab();
        expect(input).toHaveValue("mail.example.com");
    });

    it("keeps a name of the zone typed in full absolute", async () => {
        const user = userEvent.setup();
        const state = mount("");

        const input = screen.getByRole("textbox");
        await user.type(input, "mail.example.com.");
        await user.tab();

        expect(state.current()).toBe("mail.example.com.");
        expect(input).toHaveValue("mail.example.com.");
        expect(screen.queryByText(/outside of your zone/)).toBeNull();
        expect(document.querySelector(".input-group-text")).toBeNull();
    });

    it("offers to make absolute a name that looks like one", async () => {
        const user = userEvent.setup();
        const state = mount("");

        await user.type(screen.getByRole("textbox"), "ns1.ovh.net");
        expect(state.current()).toBe("ns1.ovh.net");
        expect(screen.getByText(/Did you mean ns1\.ovh\.net\./)).toBeVisible();

        await user.click(screen.getByRole("button", { name: "Use it" }));

        expect(state.current()).toBe("ns1.ovh.net.");
        expect(screen.getByRole("textbox")).toHaveValue("ns1.ovh.net.");
        expect(screen.queryByText(/Did you mean/)).toBeNull();
    });

    it("does not suggest anything for an ordinary subdomain", async () => {
        const user = userEvent.setup();
        mount("");

        await user.type(screen.getByRole("textbox"), "www.blog");

        expect(screen.queryByText(/Did you mean/)).toBeNull();
    });

    it("keeps the last valid name while the typed one cannot exist", async () => {
        const user = userEvent.setup();
        const state = mount("");

        const input = screen.getByRole("textbox");
        await user.type(input, "ns1..");

        expect(input).toHaveClass("is-invalid");
        expect(screen.getByText(/two consecutive dots/)).toBeVisible();
        expect(state.current()).toBe("ns1.");
    });

    it("explains an unusual name but still stores it", async () => {
        const user = userEvent.setup();
        const state = mount("");

        await user.type(screen.getByRole("textbox"), "my_host");

        expect(screen.getByText(/character “_” is unusual/)).toBeVisible();
        expect(state.current()).toBe("my_host");
    });

    it("shows the apex as @", () => {
        mount("@");

        expect(screen.getByRole("textbox")).toHaveValue("@");
        expect(screen.queryByText(".example.com")).toBeNull();
        expect(screen.getByText("example.com.")).toBeVisible();
    });

    it("accepts the root, standing for no name", () => {
        mount(".");

        expect(screen.getByRole("textbox")).not.toHaveClass("is-invalid");
        expect(screen.getByText(/the root of the DNS/)).toBeVisible();
    });

    it("reports whether the name is acceptable", async () => {
        const user = userEvent.setup();
        const state = mount("");
        expect(state.valid()).toBe(false);

        await user.type(screen.getByRole("textbox"), "www");
        expect(state.valid()).toBe(true);

        await user.type(screen.getByRole("textbox"), "..");
        expect(state.valid()).toBe(false);
    });

    it("refuses a name outside the zone when it has to be within", async () => {
        const user = userEvent.setup();
        const state = mount("", "example.com.", true);

        await user.type(screen.getByRole("textbox"), "www.other.org.");

        expect(screen.getByRole("textbox")).toHaveClass("is-invalid");
        expect(screen.getByText("The name has to be within example.com.")).toBeVisible();
        expect(screen.queryByText(/Did you mean/)).toBeNull();
        expect(state.valid()).toBe(false);
    });

    it("accepts the zone typed in full when the name has to be within", async () => {
        const user = userEvent.setup();
        const state = mount("", "example.com.", true);

        await user.type(screen.getByRole("textbox"), "www.example.com.");

        expect(state.valid()).toBe(true);
        expect(state.current()).toBe("www.example.com.");
    });

    it("offers the final dot to a name of the zone when it has to be within", async () => {
        const user = userEvent.setup();
        const state = mount("", "example.com.", true);

        await user.type(screen.getByRole("textbox"), "www.example.com");
        await user.click(screen.getByRole("button", { name: "Use it" }));

        expect(state.current()).toBe("www.example.com.");
    });

    describe("edge cases", () => {
        it("shows a value changed from elsewhere", async () => {
            const state = mount("ns1");

            state.set("dns.ovh.net.");
            await tick();

            expect(screen.getByRole("textbox")).toHaveValue("dns.ovh.net.");
            expect(screen.queryByText(".example.com")).toBeNull();
        });

        it("does not rewrite the text while it is typed", async () => {
            const user = userEvent.setup();
            mount("");

            const input = screen.getByRole("textbox");
            await user.type(input, "MAIL.Example.COM");

            expect(input).toHaveValue("MAIL.Example.COM");
        });

        it("finds an absolute name within the zone whatever its case", async () => {
            const user = userEvent.setup();
            const state = mount("", "example.com.", true);

            await user.type(screen.getByRole("textbox"), "MAIL.Example.COM.");

            expect(state.valid()).toBe(true);
            expect(state.current()).toBe("MAIL.Example.COM.");
        });

        it("works with an origin given without its final dot", async () => {
            const user = userEvent.setup();
            const state = mount("", "example.com");

            await user.type(screen.getByRole("textbox"), "www");

            expect(state.current()).toBe("www");
            expect(screen.getByText(".example.com.")).toBeVisible();
            expect(screen.getByText("www.example.com.")).toBeVisible();
        });

        it("does not take a lookalike of the zone for the zone", async () => {
            const user = userEvent.setup();
            const state = mount("");

            await user.type(screen.getByRole("textbox"), "notexample.com.");

            expect(state.current()).toBe("notexample.com.");
            expect(screen.getByText(/outside of your zone/)).toBeVisible();
        });

        it("stores the apex typed as @", async () => {
            const user = userEvent.setup();
            const state = mount("");

            await user.type(screen.getByRole("textbox"), "@");

            expect(state.current()).toBe("@");
            expect(state.valid()).toBe(true);
        });

        it("accepts a wildcard", async () => {
            const user = userEvent.setup();
            const state = mount("", "example.com.", true);

            await user.type(screen.getByRole("textbox"), "*.dev");

            expect(state.current()).toBe("*.dev");
            expect(state.valid()).toBe(true);
            expect(screen.getByText("*.dev.example.com.")).toBeVisible();
        });

        it("refuses the root when the name has to be within the zone", async () => {
            const user = userEvent.setup();
            const state = mount("", "example.com.", true);

            await user.type(screen.getByRole("textbox"), ".");

            expect(screen.getByRole("textbox")).toHaveClass("is-invalid");
            expect(state.valid()).toBe(false);
        });

        it("keeps an impossible name on blur, for the user to fix it", async () => {
            const user = userEvent.setup();
            mount("");

            const input = screen.getByRole("textbox");
            await user.type(input, "a..b");
            await user.tab();

            expect(input).toHaveValue("a..b");
        });

        it("refuses a label longer than 63 characters", async () => {
            const user = userEvent.setup();
            const state = mount("ok");

            const input = screen.getByRole("textbox");
            await user.clear(input);
            await user.type(input, "x".repeat(64));

            expect(input).toHaveClass("is-invalid");
            expect(state.current()).toBe("x".repeat(63));
        });

        it("refuses a name longer than 253 characters once in the zone", async () => {
            const user = userEvent.setup();
            const state = mount("");

            // 4 labels of 60 characters: fine alone, too long with .example.com.
            const name = Array(4).fill("x".repeat(60)).join(".");
            await user.click(screen.getByRole("textbox"));
            await user.paste(name);

            expect(screen.getByText(/cannot exceed 253 characters/)).toBeVisible();
            expect(state.valid()).toBe(false);
        });

        it("shows no full name nor suffix without an origin", async () => {
            const user = userEvent.setup();
            const state = mount("", "");

            await user.type(screen.getByRole("textbox"), "imap.example.com");

            expect(state.current()).toBe("imap.example.com");
            expect(screen.queryByText(/^= /)).toBeNull();
            expect(screen.queryByText(/Did you mean/)).toBeNull();
            expect(document.querySelector(".input-group-text")).toBeNull();
        });

        it("shows a read-only name without offering to change it", () => {
            mount("ns1.ovh.net", "example.com.", false, { readonly: true });

            const input = screen.getByRole("textbox");
            expect(input).toHaveAttribute("readonly");
            expect(input).toHaveClass("form-control-plaintext");
            expect(screen.queryByRole("button", { name: "Use it" })).toBeNull();
        });

        it("passes required and placeholder to the input", () => {
            mount("", "example.com.", false, { required: true, placeholder: "ns0" });

            const input = screen.getByRole("textbox");
            expect(input).toBeRequired();
            expect(input).toHaveAttribute("placeholder", "ns0");
        });

        it("tells the parent about focus and blur", async () => {
            const user = userEvent.setup();
            const onfocus = vi.fn();
            const onblur = vi.fn();
            mount("", "example.com.", false, { onfocus, onblur });

            await user.click(screen.getByRole("textbox"));
            expect(onfocus).toHaveBeenCalledTimes(1);

            await user.tab();
            expect(onblur).toHaveBeenCalledTimes(1);
        });

        it("does not submit the form when taking the suggestion", async () => {
            const user = userEvent.setup();
            const onsubmit = vi.fn((e: Event) => e.preventDefault());
            const form = document.createElement("form");
            form.addEventListener("submit", onsubmit);
            document.body.appendChild(form);
            const state = $state({ value: "" as string | undefined });

            render(DomainNameInput, {
                target: form,
                props: {
                    origin: "example.com.",
                    get value() {
                        return state.value;
                    },
                    set value(v: string | undefined) {
                        state.value = v;
                    },
                },
            });

            await user.type(screen.getByRole("textbox"), "ns1.ovh.net");
            await user.click(screen.getByRole("button", { name: "Use it" }));

            expect(onsubmit).not.toHaveBeenCalled();
            expect(state.value).toBe("ns1.ovh.net.");
            form.remove();
        });
    });
});

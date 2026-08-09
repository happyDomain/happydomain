import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { Field } from "$lib/model/custom_form.svelte";
import { loadTranslations } from "$lib/translations";
import BasicInput from "./basic.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    document.querySelectorAll("form").forEach((form) => form.remove());
});

function field(props: Partial<Field>): Field {
    const specs = new Field();
    Object.assign(specs, props);
    return specs;
}

function mount(props: {
    alwaysShow?: boolean;
    edit?: boolean;
    readonly?: boolean;
    showDescription?: boolean;
    specs: Field;
    value: unknown;
    onFocus?: () => void;
    onBlur?: () => void;
}) {
    const state = $state({ value: props.value });

    const form = document.createElement("form");
    document.body.appendChild(form);

    render(
        BasicInput,
        {
            alwaysShow: props.alwaysShow ?? false,
            edit: props.edit ?? true,
            index: "0",
            readonly: props.readonly ?? false,
            showDescription: props.showDescription ?? true,
            specs: props.specs,
            get value() {
                return state.value;
            },
            set value(v: unknown) {
                state.value = v;
            },
            $$events: {
                ...(props.onFocus ? { focus: () => props.onFocus!() } : {}),
                ...(props.onBlur ? { blur: () => props.onBlur!() } : {}),
            },
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } as any,
        { baseElement: form },
    );

    return { current: () => state.value };
}

describe("Basic input", () => {
    it("labels the field with its human-readable label", () => {
        const specs = field({ id: "domain", type: "string", label: "Domain name" });
        mount({ specs, value: "example.com" });

        expect(screen.getByText("Domain name")).toBeVisible();
    });

    it("falls back to the field id when there is no label", () => {
        const specs = field({ id: "domain", type: "string" });
        mount({ specs, value: "example.com" });

        expect(screen.getByText("domain")).toBeVisible();
    });

    it("shows the field's description alongside the input", () => {
        const specs = field({
            id: "ttl",
            type: "uint32",
            label: "TTL",
            description: "Time before the record expires",
        });
        mount({ specs, value: 3600 });

        expect(screen.getByText("Time before the record expires")).toBeVisible();
    });

    it("hides an empty, non-required field when not editing and not forced", () => {
        const specs = field({ id: "note", type: "string", label: "Note" });
        mount({ alwaysShow: false, edit: false, specs, value: null });

        expect(screen.queryByText("Note")).not.toBeInTheDocument();
    });

    it("shows an empty field once alwaysShow is set", () => {
        const specs = field({ id: "note", type: "string", label: "Note" });
        mount({ alwaysShow: true, edit: false, specs, value: null });

        expect(screen.getByText("Note")).toBeVisible();
    });

    it("delegates typing to the underlying raw input", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "domain", type: "string", label: "Domain name" });
        const state = mount({ specs, value: "" });

        await user.type(screen.getByRole("textbox"), "example.com");

        expect(state.current()).toBe("example.com");
    });

    it("renders the underlying input as plaintext and read-only when readonly", () => {
        const specs = field({ id: "domain", type: "string", label: "Domain name" });
        mount({ specs, value: "example.com", readonly: true });

        expect(screen.getByDisplayValue("example.com")).toHaveAttribute("readonly");
    });

    it("dispatches focus and blur from the underlying raw input", async () => {
        const user = userEvent.setup();
        const onFocus = vi.fn();
        const onBlur = vi.fn();
        const specs = field({ id: "domain", type: "string", label: "Domain name" });
        mount({ specs, value: "", onFocus, onBlur });

        await user.click(screen.getByRole("textbox"));
        expect(onFocus).toHaveBeenCalled();

        await user.tab();
        expect(onBlur).toHaveBeenCalled();
    });

    it("shows the description alongside choices even when showDescription is off", () => {
        const specs = field({
            id: "mode",
            type: "string",
            label: "Mode",
            description: "Pick one",
            choices: ["a", "b"],
        });
        mount({ showDescription: false, specs, value: "a" });

        expect(screen.getByText("Pick one")).toBeVisible();
    });

    it("hides the description when showDescription is off and there are no choices", () => {
        const specs = field({
            id: "domain",
            type: "string",
            label: "Domain name",
            description: "The domain",
        });
        mount({ showDescription: false, specs, value: "example.com" });

        expect(screen.queryByText("The domain")).not.toBeInTheDocument();
    });
});

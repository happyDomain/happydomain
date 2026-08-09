import { tick } from "svelte";
import { fireEvent, render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { Field, REDACTED_SECRET, REDACTED_SECRET_B64 } from "$lib/model/custom_form.svelte";
import { loadTranslations } from "$lib/translations";
import RawInput from "./raw.svelte";

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
    edit?: boolean;
    readonly?: boolean;
    specs: Field;
    value: unknown;
    onFocus?: () => void;
    onBlur?: () => void;
}) {
    const state = $state({ value: props.value });

    const form = document.createElement("form");
    document.body.appendChild(form);

    render(
        RawInput,
        {
            edit: props.edit ?? true,
            index: "0",
            readonly: props.readonly ?? false,
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

    return {
        current: () => state.value,
        set: (v: unknown) => {
            state.value = v;
        },
    };
}

describe("Raw input", () => {
    it("lets a plain text field be typed into", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "name", type: "string" });
        const state = mount({ specs, value: "" });

        await user.type(screen.getByRole("textbox"), "hello");

        expect(state.current()).toBe("hello");
    });

    it("flags a number above its type's max", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "priority", type: "uint8" });
        mount({ specs, value: 0 });

        const input = screen.getByRole("spinbutton");
        await user.clear(input);
        await user.type(input, "300");

        expect(screen.getByText("Number too high, max: 255")).toBeVisible();
        expect(input).toHaveClass("is-invalid");
    });

    it("renders a boolean field as a checkbox", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "active", type: "bool" });
        const state = mount({ specs, value: false });

        await user.click(screen.getByRole("checkbox"));

        expect(state.current()).toBe(true);
    });

    it("offers the field's choices as a select when editable", () => {
        const specs = field({ id: "mode", type: "string", choices: ["a", "b"] });
        mount({ specs, value: "a" });

        const select = screen.getByRole("combobox");
        expect(select).toHaveValue("a");
        expect(screen.getByRole("option", { name: "b" })).toBeInTheDocument();
    });

    it("keeps a redacted secret read-only until the user chooses to replace it", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "password", type: "string", secret: true });
        const state = mount({ specs, value: REDACTED_SECRET });

        const revealButton = screen.getByRole("button", { name: "Hidden by the server" });
        expect(revealButton).toBeDisabled();

        await user.click(screen.getByRole("button", { name: "Replace this secret" }));

        expect(state.current()).toBe("");
        expect(screen.getByRole("button", { name: "Keep the current secret" })).toBeInTheDocument();

        await user.click(screen.getByRole("button", { name: "Keep the current secret" }));

        expect(state.current()).toBe(REDACTED_SECRET);
        expect(screen.getByRole("button", { name: "Hidden by the server" })).toBeDisabled();
    });

    it("flags an invalid base64 value for a []byte field", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "raw", type: "[]byte" });
        mount({ specs, value: "" });

        await user.type(screen.getByRole("textbox"), "!!!not-base64");

        expect(screen.getByText(/Invalid base64 string\./)).toBeVisible();
    });

    it("flags a number below its type's min", async () => {
        const specs = field({ id: "priority", type: "int8" });
        mount({ specs, value: 0 });

        const input = screen.getByRole("spinbutton");
        await fireEvent.input(input, { target: { value: "-300" } });

        expect(screen.getByText(/Number too low, min: -256/)).toBeVisible();
    });

    it("suggests padding a []byte value that is one character short", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "raw", type: "[]byte" });
        mount({ specs, value: "" });

        await user.type(screen.getByRole("textbox"), "YQ=");

        expect(screen.getByText(/Did you mean: YQ==/)).toBeVisible();
    });

    it("flags an unfinished []byte value", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "raw", type: "[]byte" });
        mount({ specs, value: "" });

        await user.type(screen.getByRole("textbox"), "Y");

        expect(screen.getByText(/Unfinished string\./)).toBeVisible();
    });

    it("keeps a redacted []byte secret read-only using the base64 sentinel", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "cert", type: "[]byte", secret: true });
        const state = mount({ specs, value: REDACTED_SECRET_B64 });

        expect(screen.getByRole("button", { name: "Hidden by the server" })).toBeDisabled();

        await user.click(screen.getByRole("button", { name: "Replace this secret" }));
        expect(state.current()).toBe("");

        await user.click(screen.getByRole("button", { name: "Keep the current secret" }));
        expect(state.current()).toBe(REDACTED_SECRET_B64);
    });

    it("allows non-numeric duration shorthand like 1m", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "timeout", type: "time.Duration" });
        const state = mount({ specs, value: "" });

        await user.type(screen.getByRole("textbox"), "1m");

        expect(state.current()).toBe("1m");
    });

    it("shows the unit suffix for a duration field", () => {
        const specs = field({ id: "timeout", type: "time.Duration" });
        mount({ specs, value: 30 });

        expect(screen.getByText("s")).toBeVisible();
    });

    it("reveals a secret's value when the eye toggle is clicked", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "password", type: "string", secret: true });
        mount({ specs, value: "hunter2" });

        const input = screen.getByDisplayValue("hunter2") as HTMLInputElement;
        expect(input).toHaveAttribute("type", "password");

        await user.click(screen.getByRole("button", { name: "Show" }));

        expect(input).toHaveAttribute("type", "text");
        expect(screen.getByRole("button", { name: "Hide" })).toBeInTheDocument();
    });

    it("masks a secret's inherited placeholder while hidden", () => {
        const specs = field({ id: "password", type: "string", secret: true, placeholder: "abc" });
        mount({ specs, value: "" });

        expect(screen.getByPlaceholderText("•••")).toBeInTheDocument();
    });

    it("renders as plaintext and read-only when not editable", () => {
        const specs = field({ id: "name", type: "string" });
        mount({ specs, value: "hello", edit: false });

        const input = screen.getByDisplayValue("hello");
        expect(input).toHaveAttribute("readonly");
    });

    it("renders as plaintext and read-only when the readonly prop is set", () => {
        const specs = field({ id: "name", type: "string" });
        mount({ specs, value: "hello", edit: true, readonly: true });

        const input = screen.getByDisplayValue("hello");
        expect(input).toHaveAttribute("readonly");
    });

    it("does not offer a select for choices when not editable", () => {
        const specs = field({ id: "mode", type: "string", choices: ["a", "b"] });
        mount({ specs, value: "a", edit: false });

        expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    });

    it("dispatches focus and blur events", async () => {
        const user = userEvent.setup();
        const onFocus = vi.fn();
        const onBlur = vi.fn();
        const specs = field({ id: "name", type: "string" });
        mount({ specs, value: "", onFocus, onBlur });

        await user.click(screen.getByRole("textbox"));
        expect(onFocus).toHaveBeenCalled();

        await user.tab();
        expect(onBlur).toHaveBeenCalled();
    });

    it("renders a textarea when specs.textarea is set", () => {
        const specs = field({ id: "notes", type: "string", textarea: true });
        mount({ specs, value: "hello" });

        expect(screen.getByRole("textbox").tagName).toBe("TEXTAREA");
    });

    it.each([
        ["int16", 65536, -65537],
        ["uint16", 65536, 0],
        ["int32", 2147483647, -2147483648],
        ["uint32", 2147483647, 0],
        ["int", 2147483647, -2147483648],
        ["uint", 2147483647, 0],
        ["int64", 9007199254740991, -9007199254740991 - 1],
        ["uint64", 9007199254740991, 0],
    ])("computes min/max bounds for %s", (type, max, min) => {
        const specs = field({ id: "n", type });
        mount({ specs, value: 0 });

        const input = screen.getByRole("spinbutton");
        expect(input).toHaveAttribute("max", String(max));
        expect(input).toHaveAttribute("min", String(min));
    });

    it("hides a revealed secret again once it becomes redacted", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "password", type: "string", secret: true });
        const state = mount({ specs, value: "hunter2" });

        await user.click(screen.getByRole("button", { name: "Show" }));
        expect(screen.getByDisplayValue("hunter2")).toHaveAttribute("type", "text");

        // Simulate the server response replacing the value with the redacted sentinel.
        state.set(REDACTED_SECRET);
        await tick();

        expect(screen.getByRole("button", { name: "Hidden by the server" })).toBeDisabled();
        expect(screen.queryByRole("button", { name: "Hide" })).not.toBeInTheDocument();
    });

    it("leaves the value unparsed when the digits don't round-trip through parseInt", async () => {
        const specs = field({ id: "n", type: "int" });
        const state = mount({ specs, value: 0 });

        await fireEvent.input(screen.getByRole("spinbutton"), { target: { value: "007" } });

        expect(state.current()).toBe(0);
    });
});

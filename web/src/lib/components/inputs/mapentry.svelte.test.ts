import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { Field } from "$lib/model/custom_form.svelte";
import { loadTranslations } from "$lib/translations";
import MapEntry from "./mapentry.svelte";

// A map entry's value is always a struct in this app (map[string]*SomeType),
// so its ResourceInput is routed to the object editor; stub the spec lookup
// it makes rather than hitting the real API.
vi.mock("$lib/api/service_specs", () => ({
    getServiceSpec: vi.fn(async () => ({
        fields: [Object.assign(new Field(), { id: "contentType", type: "string", label: "Content-Type" })],
    })),
}));

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
    isNew?: boolean;
    key: string;
    onDeleteKey?: () => void;
    onRenameKey?: (key: string) => void;
    specs?: Field;
    value: Record<string, unknown>;
}) {
    const state = $state({ value: props.value });

    const form = document.createElement("form");
    document.body.appendChild(form);

    // MapEntry still dispatches with the pre-Svelte-5 createEventDispatcher,
    // whose listeners are wired through the `$$events` prop bag rather than
    // a real DOM CustomEvent, so `component.$on` (removed in Svelte 5) can't
    // be used here.
    const { container } = render(
        MapEntry,
        {
            edit: props.edit ?? true,
            index: "0",
            isNew: props.isNew ?? false,
            key: props.key,
            specs: props.specs ?? field({ id: "headers", label: "Headers" }),
            valuetype: "Header",
            get value() {
                return state.value;
            },
            set value(v: Record<string, unknown>) {
                state.value = v;
            },
            $$events: {
                ...(props.onDeleteKey ? { "delete-key": () => props.onDeleteKey!() } : {}),
                ...(props.onRenameKey
                    ? {
                          "rename-key": (e: CustomEvent<string>) => props.onRenameKey!(e.detail),
                      }
                    : {}),
            },
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } as any,
        { baseElement: form },
    );

    return {
        current: () => state.value,
        pencilButton: () => container.querySelector(".bi-pencil")!.closest("button")!,
        trashButton: () => container.querySelector(".bi-trash")!.closest("button")!,
        renameKeyInput: () => container.querySelector("form input") as HTMLInputElement,
    };
}

describe("Map entry", () => {
    it("shows the key next to its value input", async () => {
        mount({ key: "content-type", value: { contentType: "text/plain" } });

        expect(screen.getByText("content-type")).toBeVisible();
        expect(await screen.findByDisplayValue("text/plain")).toBeVisible();
    });

    it("starts in rename mode for a brand-new entry", () => {
        const entry = mount({ isNew: true, key: "", value: {} });

        expect(entry.renameKeyInput()).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Create new headers key" })).toBeInTheDocument();
    });

    it("dispatches delete-key when the trash button is clicked", async () => {
        const user = userEvent.setup();
        const onDeleteKey = vi.fn();
        const entry = mount({
            key: "content-type",
            onDeleteKey,
            value: { contentType: "text/plain" },
        });

        await user.click(entry.trashButton());

        expect(onDeleteKey).toHaveBeenCalled();
    });

    it("switches to rename mode and dispatches rename-key once confirmed", async () => {
        const user = userEvent.setup();
        const onRenameKey = vi.fn();
        const entry = mount({
            key: "content-type",
            onRenameKey,
            value: { contentType: "text/plain" },
        });

        await user.click(entry.pencilButton());

        const input = entry.renameKeyInput();
        await user.clear(input);
        await user.type(input, "x-content-type");
        await user.click(screen.getByRole("button", { name: "Rename" }));

        expect(onRenameKey).toHaveBeenCalledWith("x-content-type");
    });

    it("edits the entry's value through the delegated resource input", async () => {
        const user = userEvent.setup();
        const state = mount({ key: "content-type", value: { contentType: "text/plain" } });

        const input = await screen.findByDisplayValue("text/plain");
        await user.clear(input);
        await user.type(input, "text/html");

        expect(state.current().contentType).toBe("text/html");
    });

    it("stays in rename mode without dispatching when the key is cleared to empty", async () => {
        const user = userEvent.setup();
        const onRenameKey = vi.fn();
        const entry = mount({
            key: "content-type",
            onRenameKey,
            value: { contentType: "text/plain" },
        });

        await user.click(entry.pencilButton());
        const input = entry.renameKeyInput();
        await user.clear(input);
        await user.click(screen.getByRole("button", { name: "Rename" }));

        expect(onRenameKey).not.toHaveBeenCalled();
        expect(entry.renameKeyInput()).toBeInTheDocument();
    });

    it("exits rename mode without dispatching when the key is unchanged", async () => {
        const user = userEvent.setup();
        const onRenameKey = vi.fn();
        const entry = mount({
            key: "content-type",
            onRenameKey,
            value: { contentType: "text/plain" },
        });

        await user.click(entry.pencilButton());
        await user.click(screen.getByRole("button", { name: "Rename" }));

        expect(onRenameKey).not.toHaveBeenCalled();
        expect(screen.getByText("content-type")).toBeVisible();
    });

    it("hides the rename and delete controls when not editable", () => {
        mount({ edit: false, key: "content-type", value: { contentType: "text/plain" } });

        expect(document.querySelector(".bi-pencil")).not.toBeInTheDocument();
        expect(document.querySelector(".bi-trash")).not.toBeInTheDocument();
    });

    it("shows a spinner and disables the button while a delete is in flight", async () => {
        const user = userEvent.setup();
        const entry = mount({ key: "content-type", value: { contentType: "text/plain" } });

        const trash = entry.trashButton();
        await user.click(trash);

        expect(trash).toBeDisabled();
        expect(trash.querySelector(".spinner-border")).toBeInTheDocument();
        expect(trash.querySelector(".bi-trash")).not.toBeInTheDocument();
    });

    it("shows a spinner and disables the button while a rename is in flight", async () => {
        const user = userEvent.setup();
        const entry = mount({ key: "content-type", value: { contentType: "text/plain" } });

        await user.click(entry.pencilButton());
        const input = entry.renameKeyInput();
        await user.clear(input);
        await user.type(input, "x-content-type");

        const confirmButton = screen.getByRole("button", { name: "Rename" });
        await user.click(confirmButton);

        expect(confirmButton).toBeDisabled();
        expect(confirmButton.querySelector(".spinner-border")).toBeInTheDocument();
    });
});

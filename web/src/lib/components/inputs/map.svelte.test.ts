import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { Field } from "$lib/model/custom_form.svelte";
import { loadTranslations } from "$lib/translations";
import MapInput from "./map.svelte";

// Map entries are objects (map[string]*SomeType); stub the spec lookup the
// nested object editor makes rather than hitting the real API.
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
    specs?: Field;
    type?: string;
    value: Record<string, unknown> | undefined;
}) {
    const state = $state({ value: props.value });

    const form = document.createElement("form");
    document.body.appendChild(form);

    render(
        MapInput,
        {
            edit: props.edit ?? true,
            index: "0",
            specs: props.specs ?? field({ id: "headers", label: "Headers" }),
            type: props.type ?? "map[string]Header",
            get value() {
                return state.value;
            },
            set value(v: Record<string, unknown> | undefined) {
                state.value = v;
            },
        },
        { baseElement: form },
    );

    return { current: () => state.value };
}

describe("Map input", () => {
    it("says there is nothing yet when the map is empty", () => {
        mount({ value: {} });

        expect(screen.getByText("No Headers")).toBeVisible();
    });

    it("lists an entry for each existing key", () => {
        mount({ value: { "content-type": { contentType: "text/plain" } } });

        expect(screen.getByText("content-type")).toBeVisible();
    });

    it("adds a new, empty entry when the add button is clicked", async () => {
        const user = userEvent.setup();
        const state = mount({ value: {} });

        await user.click(screen.getByRole("button", { name: "Add new Headers" }));

        expect(state.current()).toHaveProperty("");
    });

    it("hides the add button once an unnamed entry is already pending", async () => {
        const user = userEvent.setup();
        mount({ value: {} });

        await user.click(screen.getByRole("button", { name: "Add new Headers" }));

        expect(screen.queryByRole("button", { name: "Add new Headers" })).not.toBeInTheDocument();
    });

    it("removes an entry when its delete button is clicked", async () => {
        const user = userEvent.setup();
        const state = mount({
            value: { "content-type": { contentType: "text/plain" } },
        });

        const trash = document.querySelector(".bi-trash")!.closest("button")!;
        await user.click(trash);

        expect(state.current()).toEqual({});
    });

    it("hides the add button in read mode even when the map is empty", () => {
        mount({ edit: false, value: {} });

        expect(screen.queryByRole("button", { name: "Add new Headers" })).not.toBeInTheDocument();
    });

    it("hides the add button in read mode even when entries exist", () => {
        mount({
            edit: false,
            value: { "content-type": { contentType: "text/plain" } },
        });

        expect(screen.queryByRole("button", { name: "Add new Headers" })).not.toBeInTheDocument();
    });

    it("renames a key when the entry is renamed", async () => {
        const user = userEvent.setup();
        const state = mount({
            value: { "content-type": { contentType: "text/plain" } },
        });

        await user.click(document.querySelector(".bi-pencil")!.closest("button")!);
        const input = document.querySelector("h3 input") as HTMLInputElement;
        await user.clear(input);
        await user.type(input, "accept");
        await user.click(screen.getByRole("button", { name: "Rename" }));

        expect(state.current()).toEqual({ accept: { contentType: "text/plain" } });
    });

    it("initializes an unset value to an empty map once the type is recognized", async () => {
        const user = userEvent.setup();
        const state = mount({ value: undefined });

        expect(screen.getByText("No Headers")).toBeVisible();

        await user.click(screen.getByRole("button", { name: "Add new Headers" }));

        expect(state.current()).toHaveProperty("");
    });

    it("renders nothing when the type doesn't match the map[key]value pattern", () => {
        mount({ type: "notamap", value: { "content-type": { contentType: "text/plain" } } });

        expect(screen.queryByText("content-type")).not.toBeInTheDocument();
        expect(screen.queryByText("No Headers")).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Add new Headers" })).not.toBeInTheDocument();
    });
});

import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { getServiceSpec } from "$lib/api/service_specs";
import { Field } from "$lib/model/custom_form.svelte";
import { loadTranslations } from "$lib/translations";
import TableInput from "./table.svelte";

// Only "MyRow" is a known multi-field row type in these tests; every other
// type string (e.g. "string") must keep failing the lookup like the real,
// server-less test environment does, so the scalar-row tests still exercise
// the single-column fallback.
vi.mock("$lib/api/service_specs", () => ({
    getServiceSpec: vi.fn(async (type: string) => {
        if (type === "MyRow") {
            return {
                fields: [
                    Object.assign(new Field(), { id: "host", type: "string", label: "Host" }),
                    Object.assign(new Field(), { id: "port", type: "uint16", label: "Port" }),
                ],
            };
        }
        throw new Error(`no spec for ${type}`);
    }),
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
    noDecorate?: boolean;
    readonly?: boolean;
    specs?: Field;
    type?: string;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    value: Array<any>;
}) {
    const state = $state({ value: props.value });

    const form = document.createElement("form");
    document.body.appendChild(form);

    render(
        TableInput,
        {
            edit: props.edit ?? true,
            index: "0",
            noDecorate: props.noDecorate ?? false,
            readonly: props.readonly ?? false,
            specs: props.specs ?? field({ id: "aliases", type: "[]string", label: "Aliases" }),
            type: props.type ?? "string",
            get value() {
                return state.value;
            },
            // TableInput's own value prop is typed Array<Record<string, any>>,
            // even though a scalar row (as used here) is a plain string.
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            set value(v: Array<any>) {
                state.value = v;
            },
        },
        { baseElement: form },
    );

    return { current: () => state.value };
}

describe("Table input", () => {
    it("shows the field's label as a header", async () => {
        mount({ value: [] });

        expect(await screen.findByRole("columnheader", { name: "Aliases" })).toBeVisible();
    });

    it("says there is no content on an empty table", async () => {
        mount({ value: [] });

        expect(await screen.findByText("No content")).toBeVisible();
    });

    it("adds a new row when the user clicks the add-row button", async () => {
        const user = userEvent.setup();
        const state = mount({ value: [] });

        await user.click(await screen.findByRole("button", { name: /New row/ }));

        expect(state.current()).toEqual([""]);
    });

    it("edits an existing row's value", async () => {
        const user = userEvent.setup();
        const state = mount({ value: ["one"] });

        const input = await screen.findByRole("textbox");
        await user.clear(input);
        await user.type(input, "two");

        expect(state.current()).toEqual(["two"]);
    });

    it("removes a row when its delete button is clicked", async () => {
        const user = userEvent.setup();
        const state = mount({ value: ["one", "two"] });

        const deleteButtons = await screen.findAllByRole("button", { name: "" });
        await user.click(deleteButtons[0]);

        expect(state.current()).toEqual(["two"]);
    });

    it("hides the add-row and delete buttons when not editable", async () => {
        mount({ edit: false, value: ["one"] });

        await screen.findByDisplayValue("one");
        expect(screen.queryByRole("button", { name: /New row/ })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "" })).not.toBeInTheDocument();
    });

    it("spans the empty-state cell across the single fallback column plus the delete column", async () => {
        mount({ value: [] });

        const cell = (await screen.findByText("No content")).closest("td")!;
        expect(cell).toHaveAttribute("colspan", "2");
    });

    it("spans the empty-state cell across every known field column plus the delete column", async () => {
        mount({ type: "MyRow", value: [] });

        const cell = (await screen.findByText("No content")).closest("td")!;
        expect(cell).toHaveAttribute("colspan", "3");
    });

    describe("multi-column rows", () => {
        it("shows one header per field instead of the spec label", async () => {
            mount({ type: "MyRow", value: [] });

            expect(await screen.findByRole("columnheader", { name: "Host" })).toBeVisible();
            expect(screen.getByRole("columnheader", { name: "Port" })).toBeVisible();
            expect(screen.queryByRole("columnheader", { name: "Aliases" })).not.toBeInTheDocument();
        });

        it("renders one cell per field for each row, bound to that field", async () => {
            const user = userEvent.setup();
            const state = mount({ type: "MyRow", value: [{ host: "localhost", port: 8080 }] });

            const hostInput = await screen.findByDisplayValue("localhost");
            await user.clear(hostInput);
            await user.type(hostInput, "example.com");

            expect(state.current()).toEqual([{ host: "example.com", port: 8080 }]);
        });

        it("adds an empty object row when the add-row button is clicked", async () => {
            const user = userEvent.setup();
            const state = mount({ type: "MyRow", value: [] });

            await user.click(await screen.findByRole("button", { name: /New row/ }));

            expect(state.current()).toEqual([{}]);
        });
    });

    it("shows a spinner while the row spec is still loading", () => {
        let resolveSpec!: (v: { fields: Field[] }) => void;
        vi.mocked(getServiceSpec).mockReturnValueOnce(
            new Promise((resolve) => {
                resolveSpec = resolve;
            }),
        );

        mount({ value: [] });

        expect(document.querySelector(".spinner-border")).toBeInTheDocument();
        expect(screen.queryByRole("table")).not.toBeInTheDocument();

        // Avoid leaking the pending promise into a later, unrelated test.
        resolveSpec({ fields: [] });
    });

    it("shows the field's label and description as a heading above the table", async () => {
        mount({
            specs: field({
                id: "aliases",
                type: "[]string",
                label: "Aliases",
                description: "Other names for this record",
            }),
            value: [],
        });

        expect(await screen.findByText("Aliases", { selector: "h4" })).toBeVisible();
        expect(screen.getByText("Other names for this record")).toBeVisible();
    });

    it("hides the heading when noDecorate is set", async () => {
        mount({
            noDecorate: true,
            specs: field({ id: "aliases", type: "[]string", label: "Aliases" }),
            value: [],
        });

        await screen.findByText("No content");
        expect(screen.queryByText("Aliases", { selector: "h4" })).not.toBeInTheDocument();
    });
});

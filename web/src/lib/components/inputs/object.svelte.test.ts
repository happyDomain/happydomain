import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { getServiceSpec } from "$lib/api/service_specs";
import { Field } from "$lib/model/custom_form.svelte";
import { ServiceInfos } from "$lib/model/service_specs.svelte";
import { loadTranslations } from "$lib/translations";
import ObjectInput from "./object.svelte";

function field(props: Partial<Field>): Field {
    const specs = new Field();
    Object.assign(specs, props);
    return specs;
}

// The fields of the object type under test, returned in place of the real API.
const innerFields = [
    field({ id: "host", type: "string", label: "Host", required: true, default: "localhost" }),
    field({ id: "port", type: "uint16", label: "Port", required: true, default: "8080" }),
];

vi.mock("$lib/api/service_specs", () => ({
    getServiceSpec: vi.fn(async () => ({ fields: innerFields })),
}));

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    document.querySelectorAll("form").forEach((form) => form.remove());
});

function mount(props: {
    edit?: boolean;
    editToolbar?: boolean;
    readonly?: boolean;
    specs?: ServiceInfos;
    type?: string;
    value: Record<string, unknown>;
    onUpdate?: () => void;
    onDelete?: () => void;
}) {
    const state = $state({ value: props.value });

    const form = document.createElement("form");
    document.body.appendChild(form);

    render(
        ObjectInput,
        {
            edit: props.edit ?? true,
            editToolbar: props.editToolbar ?? false,
            index: "0",
            readonly: props.readonly ?? false,
            specs: props.specs ?? new ServiceInfos(),
            type: props.type ?? "MyObject",
            get value() {
                return state.value;
            },
            set value(v: Record<string, unknown>) {
                state.value = v;
            },
            // `$$events` wires up createEventDispatcher listeners; it isn't part of
            // the component's typed props, so the object needs a loose cast.
            $$events: {
                ...(props.onUpdate ? { "update-this-service": () => props.onUpdate!() } : {}),
                ...(props.onDelete ? { "delete-this-service": () => props.onDelete!() } : {}),
            },
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } as any,
        { baseElement: form },
    );

    return { current: () => state.value };
}

describe("Object input", () => {
    it("fills required fields in with their default once the spec loads", async () => {
        const state = mount({ value: {} });

        expect(await screen.findByText("Host")).toBeVisible();
        expect(state.current()).toEqual({ host: "localhost", port: "8080" });
    });

    it("fills in an optional field's default even outside tabbed mode", async () => {
        vi.mocked(getServiceSpec).mockResolvedValueOnce({
            fields: [
                ...innerFields,
                field({ id: "tags", type: "[]string", label: "Tags" }),
            ],
        } as never);
        const state = mount({ value: {} });

        expect(await screen.findByText("Host")).toBeVisible();
        expect(state.current()).toEqual({ host: "localhost", port: "8080", tags: [] });
    });

    it("lets an existing value be edited through its fields", async () => {
        const user = userEvent.setup();
        const state = mount({ value: { host: "localhost", port: 8080 } });

        const hostInput = await screen.findByDisplayValue("localhost");
        await user.clear(hostInput);
        await user.type(hostInput, "example.com");

        expect(state.current().host).toBe("example.com");
    });

    it("shows edit and delete controls when the toolbar is enabled", async () => {
        mount({ editToolbar: true, value: {} });

        expect(await screen.findByRole("button", { name: /Edit/ })).toBeVisible();
        expect(screen.getByRole("button", { name: /Delete/ })).toBeVisible();
    });

    it("switches to a save button once edit is toggled on, and dispatches on save", async () => {
        const user = userEvent.setup();
        const onUpdate = vi.fn();
        mount({ editToolbar: true, onUpdate, value: {} });

        await user.click(await screen.findByRole("button", { name: /Edit/ }));
        await user.click(screen.getByRole("button", { name: /Save those modifications/ }));

        expect(onUpdate).toHaveBeenCalled();
    });

    it("dispatches delete-this-service when the delete button is clicked", async () => {
        const user = userEvent.setup();
        const onDelete = vi.fn();
        mount({ editToolbar: true, onDelete, value: {} });

        await user.click(await screen.findByRole("button", { name: /Delete/ }));

        expect(onDelete).toHaveBeenCalled();
    });

    it("hides the delete button for the abstract origin object", async () => {
        mount({ editToolbar: true, type: "abstract.Origin", value: {} });

        await screen.findByRole("button", { name: /Edit/ });
        expect(screen.queryByRole("button", { name: /Delete/ })).not.toBeInTheDocument();
    });

    it("hides the toolbar when readonly is set, even with the toolbar enabled", async () => {
        mount({ editToolbar: true, readonly: true, value: {} });

        await screen.findByText("Host");
        expect(screen.queryByRole("button", { name: /Edit/ })).not.toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /Delete/ })).not.toBeInTheDocument();
    });

    describe("tabbed mode", () => {
        const tabFields = [
            field({ id: "host", type: "string", label: "Host", required: true, default: "localhost" }),
            field({ id: "extra", type: "svcs.Extra", label: "Extra" }),
        ];

        function mountTabs(value: Record<string, unknown>) {
            vi.mocked(getServiceSpec).mockResolvedValueOnce({ fields: tabFields } as never);
            return mount({ specs: new ServiceInfos({ tabs: true }), value });
        }

        it("offers to add an optional tab whose value is undefined", async () => {
            mountTabs({});

            expect(
                await screen.findByRole("button", { name: /Add a Extra/ }),
            ).toBeVisible();
        });

        it("fills the optional tab in when its add button is clicked", async () => {
            const user = userEvent.setup();
            const state = mountTabs({});

            await user.click(await screen.findByRole("button", { name: /Add a Extra/ }));

            expect(state.current()).toHaveProperty("extra");
        });

        it("shows a remove icon on an optional tab that already has a value", async () => {
            mountTabs({ extra: { foo: "bar" } });

            await screen.findByText("Host");
            expect(document.querySelector(".bi-trash")).toBeInTheDocument();
        });

        it("clears an optional tab's value when its remove icon is clicked", async () => {
            const user = userEvent.setup();
            const state = mountTabs({ extra: { foo: "bar" } });

            await screen.findByText("Host");
            await user.click(document.querySelector(".bi-trash")!.closest("button")!);

            expect(state.current().extra).toBeUndefined();
        });

        it("does not offer to remove a required tab", async () => {
            mountTabs({});

            await screen.findByText("Host");
            expect(document.querySelector(".bi-trash")).not.toBeInTheDocument();
        });

        it("deletes the tab's whole value and forwards an update when its nested object is deleted", async () => {
            const user = userEvent.setup();
            const onUpdate = vi.fn();
            vi.mocked(getServiceSpec).mockResolvedValueOnce({ fields: tabFields } as never);
            const state = mount({
                specs: new ServiceInfos({ tabs: true }),
                editToolbar: true,
                onUpdate,
                value: { extra: { host: "localhost", port: 8080 } },
            });

            await user.click(await screen.findByRole("link", { name: /Extra/ }));
            await user.click(await screen.findByRole("button", { name: /Delete/ }));

            expect(state.current().extra).toBeUndefined();
            expect(onUpdate).toHaveBeenCalled();
        });

        it("forwards an update-this-service dispatched by a nested tab's object", async () => {
            const user = userEvent.setup();
            const onUpdate = vi.fn();
            vi.mocked(getServiceSpec).mockResolvedValueOnce({ fields: tabFields } as never);
            mount({
                specs: new ServiceInfos({ tabs: true }),
                editToolbar: true,
                onUpdate,
                value: { extra: { host: "localhost", port: 8080 } },
            });

            await user.click(await screen.findByRole("link", { name: /Extra/ }));
            await user.click(await screen.findByRole("button", { name: /Edit/ }));
            await user.click(await screen.findByRole("button", { name: /Save those modifications/ }));

            expect(onUpdate).toHaveBeenCalled();
        });
    });
});

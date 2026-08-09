import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { getServiceSpec } from "$lib/api/service_specs";
import { Field } from "$lib/model/custom_form.svelte";
import { ServiceInfos } from "$lib/model/service_specs.svelte";
import { loadTranslations } from "$lib/translations";
import ResourceInput from "./Resource.svelte";

// Only "MyStruct" is a known object type in these tests; every other type
// string (e.g. "[]string", "string") must keep failing the lookup like the
// real, server-less test environment does, or the object/table routing
// tests below would bleed into each other.
vi.mock("$lib/api/service_specs", () => ({
    getServiceSpec: vi.fn(async (type: string) => {
        if (type === "MyStruct") {
            return {
                fields: [Object.assign(new Field(), { id: "host", type: "string", label: "Host" })],
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
    editToolbar?: boolean;
    noDecorate?: boolean;
    specs?: Field | ServiceInfos;
    type: string;
    value: unknown;
    onUpdate?: () => void;
    onDelete?: () => void;
}) {
    const state = $state({ value: props.value });

    const form = document.createElement("form");
    document.body.appendChild(form);

    render(
        ResourceInput,
        {
            edit: props.edit ?? true,
            editToolbar: props.editToolbar ?? false,
            index: "0",
            noDecorate: props.noDecorate ?? false,
            specs: props.specs,
            type: props.type,
            get value() {
                return state.value;
            },
            set value(v: unknown) {
                state.value = v;
            },
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

describe("Resource input", () => {
    it("renders nothing for a hidden field", () => {
        const specs = field({ id: "secret", type: "string", hide: true });
        mount({ specs, type: "string", value: "x" });

        expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    });

    it("decorates a plain scalar with its label by default", () => {
        const specs = field({ id: "name", type: "string", label: "Name" });
        mount({ specs, type: "string", value: "" });

        expect(screen.getByText("Name")).toBeVisible();
        expect(screen.getByRole("textbox")).toBeInTheDocument();
    });

    it("skips the label decoration when noDecorate is set", () => {
        const specs = field({ id: "name", type: "string", label: "Name" });
        mount({ noDecorate: true, specs, type: "string", value: "" });

        expect(screen.queryByText("Name")).not.toBeInTheDocument();
        expect(screen.getByRole("textbox")).toBeInTheDocument();
    });

    it("routes an array type to the table editor", async () => {
        mount({ specs: field({ id: "aliases", type: "[]string", label: "Aliases" }), type: "[]string", value: [] });

        expect(await screen.findByRole("columnheader", { name: "Aliases" })).toBeVisible();
    });

    it("routes a map type to the map editor", () => {
        const specs = field({ id: "headers", type: "map[string]string", label: "Headers" });
        mount({ specs, type: "map[string]string", value: {} });

        expect(screen.getByText("No Headers")).toBeVisible();
    });

    it("keeps a []byte field as a scalar instead of routing it to the table editor", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "cert", type: "[]byte", label: "Cert" });
        const state = mount({ specs, type: "[]byte", value: "" });

        expect(screen.queryByRole("columnheader")).not.toBeInTheDocument();
        const input = screen.getByRole("textbox");

        await user.type(input, "!!!not-base64");
        expect(screen.getByText(/Invalid base64 string\./)).toBeVisible();

        expect(state.current()).toBe("!!!not-base64");
    });

    it("keeps a []uint8 field as a scalar instead of routing it to the table editor", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "cert", type: "[]uint8", label: "Cert" });
        const state = mount({ specs, type: "[]uint8", value: "" });

        expect(screen.queryByRole("columnheader")).not.toBeInTheDocument();
        const input = screen.getByRole("textbox");

        await user.type(input, "!!!not-base64");
        expect(screen.getByText(/Invalid base64 string\./)).toBeVisible();

        expect(state.current()).toBe("!!!not-base64");
    });

    it("lets a scalar value be typed into", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "name", type: "string", label: "Name" });
        const state = mount({ specs, type: "string", value: "" });

        await user.type(screen.getByRole("textbox"), "example.com");

        expect(state.current()).toBe("example.com");
    });

    it("strips a leading pointer marker before routing a scalar type", async () => {
        const user = userEvent.setup();
        const specs = field({ id: "name", type: "*string", label: "Name" });
        const state = mount({ specs, type: "*string", value: "" });

        await user.type(screen.getByRole("textbox"), "example.com");

        expect(state.current()).toBe("example.com");
    });

    it("routes an object-typed value to the object editor", async () => {
        const specs = new ServiceInfos();
        mount({ specs, type: "MyStruct", value: { host: "localhost" } });

        expect(await screen.findByText("Host")).toBeVisible();
    });

    it("routes to the object editor when specs is an array, even though value isn't an object", () => {
        // A scalar value would normally route to Basic/RawInput, neither of
        // which shows a loading spinner; ObjectInput is the only route that
        // renders one synchronously, before its own getServiceSpec resolves.
        // fields: [] keeps ObjectInput's post-load default-filling effect
        // from indexing into the non-object value once it resolves.
        vi.mocked(getServiceSpec).mockResolvedValueOnce({ fields: [] });
        const specs = [field({ id: "host", type: "string", label: "Host" })];
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        mount({ specs: specs as any, type: "EmptyStruct", value: "" });

        expect(document.querySelector(".spinner-border")).toBeInTheDocument();
        expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    });

    it("forces edit on for a scalar when editToolbar is set even if edit is off", () => {
        const specs = field({ id: "name", type: "string", label: "Name" });
        mount({ edit: false, editToolbar: true, specs, type: "string", value: "hello" });

        const input = screen.getByDisplayValue("hello");
        expect(input).not.toHaveAttribute("readonly");
    });

    it("forwards update-this-service and delete-this-service from a nested object editor", async () => {
        const user = userEvent.setup();
        const onUpdate = vi.fn();
        const onDelete = vi.fn();
        const specs = new ServiceInfos();
        mount({ editToolbar: true, onDelete, onUpdate, specs, type: "MyStruct", value: {} });

        await user.click(await screen.findByRole("button", { name: /Edit/ }));
        await user.click(screen.getByRole("button", { name: /Save those modifications/ }));
        expect(onUpdate).toHaveBeenCalled();

        await user.click(screen.getByRole("button", { name: /Delete/ }));
        expect(onDelete).toHaveBeenCalled();
    });
});

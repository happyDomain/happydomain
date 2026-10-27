import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { loadTranslations } from "$lib/translations";
import DurationInput from "./Duration.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

function mount(props: { value: unknown; zeroHint?: string; onchange?: () => void }) {
    const state = $state({ value: props.value });
    render(DurationInput, {
        id: "ttl",
        zeroHint: props.zeroHint,
        onchange: props.onchange,
        get value() {
            return state.value as number;
        },
        set value(v: number) {
            state.value = v;
        },
    });
    return { current: () => state.value };
}

describe("DurationInput", () => {
    it("names what zero stands for", () => {
        mount({ value: 0, zeroHint: "default TTL" });

        expect(screen.getByText("= default TTL")).toBeVisible();
    });

    it("reports a committed value once the field is left", async () => {
        const user = userEvent.setup();
        const onchange = vi.fn();
        const state = mount({ value: 300, onchange });

        const input = screen.getByRole("textbox");
        await user.clear(input);
        await user.type(input, "1h");
        expect(onchange).not.toHaveBeenCalled();

        await user.tab();

        expect(onchange).toHaveBeenCalledOnce();
        expect(state.current()).toBe(3600);
    });

    it("does not report a value it cannot read", async () => {
        const user = userEvent.setup();
        const onchange = vi.fn();
        mount({ value: 300, onchange });

        const input = screen.getByRole("textbox");
        await user.clear(input);
        await user.type(input, "soon");
        await user.tab();

        expect(onchange).not.toHaveBeenCalled();
    });
});

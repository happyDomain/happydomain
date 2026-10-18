import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { REDACTED_SECRET } from "$lib/model/custom_form.svelte";
import { loadTranslations } from "$lib/translations";

const updateChannel = vi.fn(async () => ({}));
const createChannel = vi.fn(async () => ({}));

vi.mock("$lib/api/notifications", () => ({
    updateChannel: (...args: unknown[]) => updateChannel(...(args as [])),
    createChannel: (...args: unknown[]) => createChannel(...(args as [])),
}));

import ChannelEditor from "./ChannelEditor.svelte";

beforeAll(async () => {
    await loadTranslations("en", "/");
});

beforeEach(() => {
    updateChannel.mockClear();
    createChannel.mockClear();
});

function mount(channel: unknown) {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const props: any = {
        open: true,
        channelTypes: ["webhook", "unifiedpush"],
        channel,
        onSave: () => {},
        onClose: () => {},
    };
    render(ChannelEditor, props);
}

describe("ChannelEditor secret fields", () => {
    it("keeps a withheld secret read-only and sends it back verbatim", async () => {
        const user = userEvent.setup();
        mount({
            id: "c1",
            type: "webhook",
            name: "hook",
            enabled: true,
            config: { url: "https://example.com/hook", headers: {}, secret: REDACTED_SECRET },
        });

        const input = document.getElementById("spec-channel-secret") as HTMLInputElement;
        expect(input).not.toBeNull();
        expect(input.readOnly).toBe(true);
        expect(input.value).toBe(REDACTED_SECRET);

        await user.click(screen.getByRole("button", { name: /save/i }));

        expect(updateChannel).toHaveBeenCalledTimes(1);
        const [, payload] = updateChannel.mock.calls[0] as unknown as [
            string,
            { config: Record<string, unknown> },
        ];
        expect(payload.config.secret).toBe(REDACTED_SECRET);
    });

    it("lets the user replace a withheld secret", async () => {
        const user = userEvent.setup();
        mount({
            id: "c1",
            type: "webhook",
            name: "hook",
            enabled: true,
            config: { url: "https://example.com/hook", headers: {}, secret: REDACTED_SECRET },
        });

        await user.click(screen.getByTitle("Replace this secret"));
        const input = document.getElementById("spec-channel-secret") as HTMLInputElement;
        await user.type(input, "n3w");
        await user.click(screen.getByRole("button", { name: /save/i }));

        const [, payload] = updateChannel.mock.calls[0] as unknown as [
            string,
            { config: Record<string, unknown> },
        ];
        expect(payload.config.secret).toBe("n3w");
    });

    it("treats the UnifiedPush endpoint as a secret too", () => {
        mount({
            id: "c2",
            type: "unifiedpush",
            name: "up",
            enabled: true,
            config: { endpoint: REDACTED_SECRET },
        });

        const input = document.getElementById("spec-channel-endpoint") as HTMLInputElement;
        expect(input.readOnly).toBe(true);
        expect(input.value).toBe(REDACTED_SECRET);
    });
});

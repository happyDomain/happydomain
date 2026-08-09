import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import type { Domain } from "$lib/model/domain";
import type { Provider } from "$lib/model/provider";
import { loadTranslations } from "$lib/translations";
import NewDomain from "./NewDomain.svelte";

const addDomain = vi.fn(async (domain: string, _provider: Provider | undefined) => ({ domain }) as Domain);
const refreshDomains = vi.fn(async () => []);

vi.mock("$lib/api/domains", () => ({
    addDomain: (...args: [string, Provider | undefined]) => addDomain(...args),
}));
vi.mock("$lib/stores/domains", () => ({
    refreshDomains: () => refreshDomains(),
}));

// Swaps in a plain-button stand-in for the real, multi-step picker modal so
// onProviderSelected can be exercised without driving a Bootstrap modal.
vi.mock("$lib/components/modals/PickProvider.svelte", async () => {
    const stub = await import("./__fixtures__/PickProviderStub.svelte");
    return { default: stub.default, controls: { Open: vi.fn() } };
});

const { controls: pickProviderControls } = (await import(
    "$lib/components/modals/PickProvider.svelte"
)) as unknown as { controls: { Open: ReturnType<typeof vi.fn> } };

beforeAll(async () => {
    await loadTranslations("en", "/");
});

afterEach(() => {
    vi.clearAllMocks();
    document.querySelectorAll("form").forEach((form) => form.remove());
});

const provider = { _id: "p1" } as Provider;

function mount(
    props: {
        provider?: Provider;
        noButton?: boolean;
        onNewDomainAdded?: (domain: Domain) => void;
    } = {},
) {
    const form = document.createElement("form");
    document.body.appendChild(form);

    render(
        NewDomain,
        {
            provider: props.provider,
            noButton: props.noButton,
            $$events: props.onNewDomainAdded
                ? { newDomainAdded: (e: CustomEvent<Domain>) => props.onNewDomainAdded!(e.detail) }
                : undefined,
        },
        { baseElement: form },
    );
}

describe("New domain input", () => {
    it("does not offer an add button while the field is empty", () => {
        mount({ provider });

        expect(screen.queryByRole("button", { name: /Add new domain/ })).not.toBeInTheDocument();
    });

    it("submits the typed domain to the given provider", async () => {
        const user = userEvent.setup();
        const onNewDomainAdded = vi.fn();
        mount({ provider, onNewDomainAdded });

        await user.type(screen.getByPlaceholderText("my.new.domain."), "example.com");
        await user.click(screen.getByRole("button", { name: /Add new domain/ }));

        expect(addDomain).toHaveBeenCalledWith("example.com", provider);
        await vi.waitFor(() => expect(refreshDomains).toHaveBeenCalled());
        await vi.waitFor(() => expect(onNewDomainAdded).toHaveBeenCalledWith({ domain: "example.com" }));
    });

    it("clears the field once the domain has been added", async () => {
        const user = userEvent.setup();
        mount({ provider });

        const input = screen.getByPlaceholderText("my.new.domain.");
        await user.type(input, "example.com");
        await user.click(screen.getByRole("button", { name: /Add new domain/ }));

        await vi.waitFor(() => expect(input).toHaveValue(""));
    });

    it("marks a valid typed domain as valid", async () => {
        const user = userEvent.setup();
        mount({ provider });

        await user.type(screen.getByPlaceholderText("my.new.domain."), "example.com");

        expect(screen.getByPlaceholderText("my.new.domain.")).toHaveClass("is-valid");
    });

    it("marks an illegal typed domain as invalid", async () => {
        const user = userEvent.setup();
        mount({ provider });

        await user.type(screen.getByPlaceholderText("my.new.domain."), "in valid");

        expect(screen.getByPlaceholderText("my.new.domain.")).toHaveClass("is-invalid");
    });

    it("leaves the field unmarked while it is empty", () => {
        mount({ provider });

        const input = screen.getByPlaceholderText("my.new.domain.");
        expect(input).not.toHaveClass("is-valid");
        expect(input).not.toHaveClass("is-invalid");
    });

    it("opens the provider picker instead of adding when no provider is set", async () => {
        const user = userEvent.setup();
        mount();

        await user.type(screen.getByPlaceholderText("my.new.domain."), "example.com");
        await user.click(screen.getByRole("button", { name: /Add new domain/ }));

        expect(pickProviderControls.Open).toHaveBeenCalled();
        expect(addDomain).not.toHaveBeenCalled();
    });

    it("skips adding the domain when preAddFunc rejects it", async () => {
        const user = userEvent.setup();
        const preAddFunc = vi.fn(async () => false);
        const form = document.createElement("form");
        document.body.appendChild(form);

        render(NewDomain, { preAddFunc, provider }, { baseElement: form });

        await user.type(screen.getByPlaceholderText("my.new.domain."), "example.com");
        await user.click(screen.getByRole("button", { name: /Add new domain/ }));

        expect(preAddFunc).toHaveBeenCalledWith("example.com");
        expect(addDomain).not.toHaveBeenCalled();
    });

    it("still adds the domain when preAddFunc approves it", async () => {
        const user = userEvent.setup();
        const preAddFunc = vi.fn(async () => true);
        const form = document.createElement("form");
        document.body.appendChild(form);

        render(NewDomain, { preAddFunc, provider }, { baseElement: form });

        await user.type(screen.getByPlaceholderText("my.new.domain."), "example.com");
        await user.click(screen.getByRole("button", { name: /Add new domain/ }));

        expect(preAddFunc).toHaveBeenCalledWith("example.com");
        expect(addDomain).toHaveBeenCalledWith("example.com", provider);
    });

    it("re-throws when the API rejects adding the domain, resetting the submitting state", async () => {
        const user = userEvent.setup();
        addDomain.mockRejectedValueOnce(new Error("boom"));
        const onUnhandledRejection = vi.fn();
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (globalThis as any).process.once("unhandledRejection", onUnhandledRejection);
        mount({ provider });

        await user.type(screen.getByPlaceholderText("my.new.domain."), "example.com");
        const button = screen.getByRole("button", { name: /Add new domain/ });
        await user.click(button);

        // The button is re-enabled (not left in the submitting state) even
        // though the add itself failed and its error propagated.
        await vi.waitFor(() => expect(button).not.toBeDisabled());
        await vi.waitFor(() => expect(onUnhandledRejection).toHaveBeenCalled());
    });

    it("adds the domain through the provider picker when none was pre-selected", async () => {
        const user = userEvent.setup();
        const onNewDomainAdded = vi.fn();
        mount({ onNewDomainAdded });

        await user.type(screen.getByPlaceholderText("my.new.domain."), "example.com");
        await user.click(screen.getByRole("button", { name: /Add new domain/ }));

        expect(pickProviderControls.Open).toHaveBeenCalled();

        await user.click(screen.getByRole("button", { name: "pick provider" }));

        expect(addDomain).toHaveBeenCalledWith("example.com", { _id: "picked-provider" });
        await vi.waitFor(() => expect(refreshDomains).toHaveBeenCalled());
        await vi.waitFor(() => expect(onNewDomainAdded).toHaveBeenCalledWith({ domain: "example.com" }));
    });

    it("hides the submit button when noButton is set, even with a value typed in", async () => {
        const user = userEvent.setup();
        mount({ provider, noButton: true });

        await user.type(screen.getByPlaceholderText("my.new.domain."), "example.com");

        expect(screen.queryByRole("button", { name: /Add new domain/ })).not.toBeInTheDocument();
    });

    it("still submits on Enter when noButton hides the submit button", async () => {
        const user = userEvent.setup();
        mount({ provider, noButton: true });

        await user.type(screen.getByPlaceholderText("my.new.domain."), "example.com{Enter}");

        expect(addDomain).toHaveBeenCalledWith("example.com", provider);
    });
});

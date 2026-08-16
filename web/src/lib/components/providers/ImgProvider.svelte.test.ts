import { fireEvent, render, screen } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";

import type { Provider, ProviderInfos } from "$lib/model/provider";
import { providers, providersSpecs } from "$lib/stores/providers";
import ImgProvider from "./ImgProvider.svelte";

afterEach(() => {
    providers.set(undefined);
    providersSpecs.set(undefined);
});

function provider(props: Partial<Provider>): Provider {
    return {
        _id: "p1",
        _srctype: "ovh.api",
        Provider: {},
        ...props,
    } as Provider;
}

describe("ImgProvider", () => {
    it("points the icon lookup at the given provider type", () => {
        render(ImgProvider, { ptype: "ovh.api" });

        expect(screen.getByRole("img")).toHaveAttribute("src", "/api/providers/_specs/ovh.api/icon");
    });

    it("resolves the provider type from the configured provider when only id_provider is given", () => {
        providers.set([provider({ _id: "p1", _srctype: "ovh.api" })]);

        render(ImgProvider, { id_provider: "p1" });

        expect(screen.getByRole("img")).toHaveAttribute("src", "/api/providers/_specs/ovh.api/icon");
    });

    it("prefers an explicit ptype over the one derived from id_provider", () => {
        providers.set([provider({ _id: "p1", _srctype: "ovh.api" })]);

        render(ImgProvider, { id_provider: "p1", ptype: "gandi.api" });

        expect(screen.getByRole("img")).toHaveAttribute("src", "/api/providers/_specs/gandi.api/icon");
    });

    it("renders no image and an empty placeholder when the type cannot be resolved", () => {
        const { container } = render(ImgProvider, { id_provider: "unknown" });

        expect(container.querySelector("img")).not.toBeInTheDocument();
        expect(container.querySelector("svg")).not.toBeInTheDocument();
    });

    it("labels the fallback monogram with the human-readable provider name once specs are loaded", async () => {
        providersSpecs.set({
            "ovh.api": { name: "OVH" } as ProviderInfos,
        });
        render(ImgProvider, { ptype: "ovh.api" });

        await fireEvent.error(screen.getByRole("img"));

        expect(screen.getByRole("img", { name: "OVH" })).toBeVisible();
    });

    it("labels the fallback monogram with the raw type when specs are not loaded yet", async () => {
        render(ImgProvider, { ptype: "ovh.api" });

        await fireEvent.error(screen.getByRole("img"));

        expect(screen.getByRole("img", { name: "ovh.api" })).toBeVisible();
    });

    it("uses the raw type as alt and title, not the human-readable name", () => {
        providersSpecs.set({
            "ovh.api": { name: "OVH" } as ProviderInfos,
        });
        render(ImgProvider, { ptype: "ovh.api" });

        const img = screen.getByRole("img");
        expect(img).toHaveAttribute("alt", "ovh.api");
        expect(img).toHaveAttribute("title", "ovh.api");
    });

    it("does not let an error on one provider type hide the icon of a different type rendered afterwards", async () => {
        const { rerender } = render(ImgProvider, { ptype: "ovh.api" });
        await fireEvent.error(screen.getByRole("img"));
        expect(document.querySelector("img")).not.toBeInTheDocument();

        await rerender({ ptype: "gandi.api" });

        expect(screen.getByRole("img")).toHaveAttribute("src", "/api/providers/_specs/gandi.api/icon");
    });

    it("forwards extra attributes to the img element", () => {
        render(ImgProvider, { ptype: "ovh.api", "data-testid": "provider-icon" } as never);

        expect(screen.getByTestId("provider-icon")).toBeVisible();
    });
});

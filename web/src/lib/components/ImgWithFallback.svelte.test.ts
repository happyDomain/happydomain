import { fireEvent, render, screen } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";

import ImgWithFallback from "./ImgWithFallback.svelte";

describe("ImgWithFallback", () => {
    it("renders an img pointing at the given src", () => {
        render(ImgWithFallback, { src: "/icon.png", label: "Example" });

        const img = screen.getByRole("img");
        expect(img).toHaveAttribute("src", "/icon.png");
    });

    it("uses the label as alt and title by default", () => {
        render(ImgWithFallback, { src: "/icon.png", label: "Example" });

        const img = screen.getByRole("img");
        expect(img).toHaveAttribute("alt", "Example");
        expect(img).toHaveAttribute("title", "Example");
    });

    it("lets alt and title be overridden independently of the label", () => {
        render(ImgWithFallback, { src: "/icon.png", label: "Example", alt: "Custom alt", title: "Custom title" });

        const img = screen.getByRole("img");
        expect(img).toHaveAttribute("alt", "Custom alt");
        expect(img).toHaveAttribute("title", "Custom title");
    });

    it("falls back to the monogram when the image fails to load", async () => {
        render(ImgWithFallback, { src: "/icon.png", label: "Example" });

        await fireEvent.error(screen.getByRole("img"));

        expect(screen.queryByRole("img", { name: "Example" })).toBeInTheDocument();
        // The monogram is an SVG, so the failed <img> tag must be gone.
        expect(document.querySelector("img")).not.toBeInTheDocument();
        expect(document.querySelector("svg")).toBeInTheDocument();
    });

    it("renders an empty placeholder when there is no src and no label", () => {
        const { container } = render(ImgWithFallback, {});

        expect(container.querySelector("img")).not.toBeInTheDocument();
        expect(container.querySelector("svg")).not.toBeInTheDocument();
        expect(container.querySelector("span")).toBeInTheDocument();
    });

    it("skips straight to the monogram fallback when there is no src", () => {
        render(ImgWithFallback, { label: "Example" });

        expect(document.querySelector("img")).not.toBeInTheDocument();
        expect(screen.getByRole("img", { name: "Example" })).toBeVisible();
    });

    it("does not let an error on one key hide an icon rendered under another key", async () => {
        const { rerender } = render(ImgWithFallback, { src: "/a.png", errorKey: "a", label: "A" });
        await fireEvent.error(screen.getByRole("img"));
        expect(document.querySelector("img")).not.toBeInTheDocument();

        await rerender({ src: "/b.png", errorKey: "b", label: "B" });

        expect(screen.getByRole("img")).toHaveAttribute("src", "/b.png");
    });

    it("keeps showing the fallback when re-rendered with the same errored key", async () => {
        const { rerender } = render(ImgWithFallback, { src: "/a.png", errorKey: "a", label: "A" });
        await fireEvent.error(screen.getByRole("img"));

        await rerender({ src: "/a.png", errorKey: "a", label: "A" });

        expect(document.querySelector("img")).not.toBeInTheDocument();
        expect(screen.getByRole("img", { name: "A" })).toBeVisible();
    });

    it("applies the given style to the img element", () => {
        render(ImgWithFallback, { src: "/icon.png", label: "Example", style: "width: 10px" });

        expect(screen.getByRole("img")).toHaveAttribute("style", "width: 10px;");
    });

    it("passes the loading attribute through to the img element", () => {
        render(ImgWithFallback, { src: "/icon.png", label: "Example", loading: "lazy" });

        expect(screen.getByRole("img")).toHaveAttribute("loading", "lazy");
    });

    it("forwards extra attributes to the img element", () => {
        render(ImgWithFallback, { src: "/icon.png", label: "Example", "data-testid": "icon" } as never);

        expect(screen.getByTestId("icon")).toBeVisible();
    });

    it("forwards extra attributes to the empty placeholder", () => {
        render(ImgWithFallback, { "data-testid": "placeholder" } as never);

        expect(screen.getByTestId("placeholder")).toBeVisible();
    });
});

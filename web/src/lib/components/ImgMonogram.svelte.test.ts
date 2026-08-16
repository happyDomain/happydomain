import { render, screen } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";

import ImgMonogram from "./ImgMonogram.svelte";

describe("ImgMonogram", () => {
    it("draws the first letter of the label, uppercased", () => {
        render(ImgMonogram, { label: "example.com" });

        expect(screen.getByText("E")).toBeVisible();
    });

    it("draws the first digit when the label starts with a number", () => {
        render(ImgMonogram, { label: "42things.com" });

        expect(screen.getByText("4")).toBeVisible();
    });

    it("skips leading punctuation to find the initial", () => {
        render(ImgMonogram, { label: "-example" });

        expect(screen.getByText("E")).toBeVisible();
    });

    it("renders nothing as the initial when the label has no letter or digit", () => {
        render(ImgMonogram, { label: "---" });

        expect(screen.queryByText(/[a-zA-Z0-9]/)).not.toBeInTheDocument();
    });

    it("exposes the label as the accessible name and title", () => {
        render(ImgMonogram, { label: "example.com" });

        expect(screen.getByRole("img", { name: "example.com" })).toBeVisible();
    });

    it("produces the same colour for the same label", () => {
        const { container: c1 } = render(ImgMonogram, { label: "example.com" });
        const fill1 = c1.querySelector("rect")?.getAttribute("fill");

        const { container: c2 } = render(ImgMonogram, { label: "example.com" });
        const fill2 = c2.querySelector("rect")?.getAttribute("fill");

        expect(fill1).toBe(fill2);
    });

    it("produces different colours for different labels", () => {
        const { container: c1 } = render(ImgMonogram, { label: "example.com" });
        const fill1 = c1.querySelector("rect")?.getAttribute("fill");

        const { container: c2 } = render(ImgMonogram, { label: "other.net" });
        const fill2 = c2.querySelector("rect")?.getAttribute("fill");

        expect(fill1).not.toBe(fill2);
    });

    it("uses the given size for width and height", () => {
        const { container } = render(ImgMonogram, { label: "example.com", size: 64 });

        const svg = container.querySelector("svg");
        expect(svg).toHaveAttribute("width", "64");
        expect(svg).toHaveAttribute("height", "64");
    });

    it("defaults to a 32px size", () => {
        const { container } = render(ImgMonogram, { label: "example.com" });

        const svg = container.querySelector("svg");
        expect(svg).toHaveAttribute("width", "32");
        expect(svg).toHaveAttribute("height", "32");
    });

    it("applies the given style", () => {
        const { container } = render(ImgMonogram, { label: "example.com", style: "opacity: 0.5" });

        expect(container.querySelector("svg")).toHaveAttribute("style", "opacity: 0.5;");
    });

    it("forwards extra attributes to the svg element", () => {
        render(ImgMonogram, { label: "example.com", "data-testid": "monogram" } as never);

        expect(screen.getByTestId("monogram")).toBeVisible();
    });
});

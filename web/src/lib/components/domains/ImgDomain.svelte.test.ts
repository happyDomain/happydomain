import { fireEvent, render, screen } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";

import ImgDomain from "./ImgDomain.svelte";

describe("ImgDomain", () => {
    it("points the favicon lookup at the domain", () => {
        render(ImgDomain, { domain: "example.com" });

        expect(screen.getByRole("img")).toHaveAttribute(
            "src",
            "/api/favicon/" + encodeURIComponent("example.com"),
        );
    });

    it("strips the trailing dot from a FQDN for the favicon lookup", () => {
        render(ImgDomain, { domain: "example.com." });

        expect(screen.getByRole("img")).toHaveAttribute(
            "src",
            "/api/favicon/" + encodeURIComponent("example.com"),
        );
    });

    it("encodes special characters in the domain", () => {
        render(ImgDomain, { domain: "xn--exmple-cva.com" });

        expect(screen.getByRole("img")).toHaveAttribute(
            "src",
            "/api/favicon/" + encodeURIComponent("xn--exmple-cva.com"),
        );
    });

    it("uses the domain, without its www. prefix, as the accessible label", () => {
        render(ImgDomain, { domain: "www.example.com" });

        expect(screen.getByRole("img")).toHaveAttribute("alt", "example.com");
    });

    it("keeps the www. prefix in the favicon lookup even though it is dropped from the label", () => {
        render(ImgDomain, { domain: "www.example.com" });

        expect(screen.getByRole("img")).toHaveAttribute(
            "src",
            "/api/favicon/" + encodeURIComponent("www.example.com"),
        );
    });

    it("lazy-loads the image", () => {
        render(ImgDomain, { domain: "example.com" });

        expect(screen.getByRole("img")).toHaveAttribute("loading", "lazy");
    });

    it("falls back to a monogram of the domain when the favicon fails to load", async () => {
        render(ImgDomain, { domain: "example.com" });

        await fireEvent.error(screen.getByRole("img"));

        expect(document.querySelector("img")).not.toBeInTheDocument();
        expect(screen.getByRole("img", { name: "example.com" })).toBeVisible();
    });

    it("skips the favicon fetch and shows a monogram of the raw domain when it reduces to nothing", () => {
        render(ImgDomain, { domain: "." });

        expect(document.querySelector("img")).not.toBeInTheDocument();
        expect(screen.getByRole("img", { name: "." })).toBeVisible();
    });

    it("forwards extra attributes to the img element", () => {
        render(ImgDomain, { domain: "example.com", "data-testid": "domain-icon" } as never);

        expect(screen.getByTestId("domain-icon")).toBeVisible();
    });
});

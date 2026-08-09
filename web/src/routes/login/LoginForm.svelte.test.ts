import { render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { goto } from "$app/navigation";
import { authUser, cleanUserSession } from "$lib/api/user";
import { CaptchaRequiredError, RateLimitedError } from "$lib/hey-api";
import { appConfig } from "$lib/stores/config";
import { toasts } from "$lib/stores/toasts";
import { loadTranslations } from "$lib/translations";
import LoginFormComponent from "./LoginForm.svelte";

// Leaving the page is what a successful login does, and jsdom has nowhere to
// go: the calls are recorded instead.
vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

// The URL the form reads its `next` parameter from. Each test rewrites it
// before mounting.
const pageState = { url: new URL("http://localhost/login") };
vi.mock("$app/state", () => ({
    get page() {
        return pageState;
    },
}));

// The network lives here: the tests decide what the backend answers.
vi.mock("$lib/api/user", () => ({
    authUser: vi.fn(),
    cleanUserSession: vi.fn(),
}));
vi.mock("$lib/api/auth", () => ({
    getOidcProvider: vi.fn(async () => ({ provider: "gitlab.com" })),
}));
vi.mock("$lib/stores/usersession", () => ({
    refreshUserSession: vi.fn(async () => ({})),
}));

// Without this, every label reads back as its translation key.
beforeAll(async () => {
    await loadTranslations("en", "/");
});

beforeEach(() => {
    vi.mocked(authUser).mockResolvedValue({} as never);
    pageState.url = new URL("http://localhost/login");
    appConfig.set({});
});

afterEach(() => {
    vi.clearAllMocks();
    for (const toast of get(toasts)) toasts.dismiss(toast.id);
});

/** Fill in the two fields the way someone logging in would. */
async function fillIn(
    user: ReturnType<typeof userEvent.setup>,
    email = "me@example.com",
    password = "s3cr3t",
) {
    await user.type(screen.getByLabelText("Email address"), email);
    await user.type(screen.getByLabelText("Password"), password);
}

function submitButton() {
    return screen.getByRole("button", { name: "Go!" });
}

describe("login form", () => {
    it("asks for an email address and a password, and nothing else", () => {
        render(LoginFormComponent);

        expect(screen.getByLabelText("Email address")).toHaveAttribute("type", "email");
        expect(screen.getByLabelText("Password")).toHaveAttribute("type", "password");
        expect(submitButton()).toBeEnabled();
    });

    it("lets the browser fill in a saved account", () => {
        render(LoginFormComponent);

        // The values the password managers look for to offer the right entry.
        expect(screen.getByLabelText("Email address")).toHaveAttribute("autocomplete", "username");
        expect(screen.getByLabelText("Password")).toHaveAttribute(
            "autocomplete",
            "current-password",
        );
    });

    it("sends the credentials that were typed in", async () => {
        const user = userEvent.setup();
        render(LoginFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        expect(authUser).toHaveBeenCalledWith({
            email: "me@example.com",
            password: "s3cr3t",
        });
    });

    it("keeps an incomplete form to itself", async () => {
        const user = userEvent.setup();
        render(LoginFormComponent);

        await user.type(screen.getByLabelText("Email address"), "me@example.com");
        await user.click(submitButton());

        expect(authUser).not.toHaveBeenCalled();
    });

    it("marks an address without an @ as wrong as soon as the field is left", async () => {
        const user = userEvent.setup();
        render(LoginFormComponent);

        const email = screen.getByLabelText("Email address");
        await user.type(email, "not-an-address");
        await user.tab();

        expect(email).toHaveClass("is-invalid");
    });

    it("starts a new session and lands on the home page", async () => {
        const user = userEvent.setup();
        render(LoginFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        // Whatever the previous user left behind must not survive the login.
        await waitFor(() => expect(cleanUserSession).toHaveBeenCalled());
        expect(goto).toHaveBeenCalledWith("/");
    });

    it("goes back to the page the user was sent away from", async () => {
        const user = userEvent.setup();
        pageState.url = new URL("http://localhost/login?next=%2Fdomains%2Fexample.com");
        render(LoginFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        await waitFor(() => expect(goto).toHaveBeenCalledWith("/domains/example.com"));
    });

    it("refuses to hand the user over to another site", async () => {
        const user = userEvent.setup();
        pageState.url = new URL("http://localhost/login?next=%2F%2Fevil.example.net");
        render(LoginFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        await waitFor(() => expect(goto).toHaveBeenCalledWith("/"));
        expect(goto).not.toHaveBeenCalledWith("//evil.example.net");
    });

    it("reports a refused login without losing what was typed", async () => {
        const user = userEvent.setup();
        vi.mocked(authUser).mockRejectedValue(new Error("Invalid username or password"));
        render(LoginFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        await waitFor(() => expect(get(toasts)).toHaveLength(1));
        expect(get(toasts)[0].message).toBe("Invalid username or password");
        expect(screen.getByLabelText("Email address")).toHaveValue("me@example.com");
        // The button comes back, otherwise there is no way to try again.
        expect(submitButton()).toBeEnabled();
    });

    it("closes the form once too many attempts have been made", async () => {
        const user = userEvent.setup();
        vi.mocked(authUser).mockRejectedValue(new RateLimitedError("too many"));
        render(LoginFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        await waitFor(() =>
            expect(
                screen.getByText(
                    "Too many failed login attempts. Please wait a moment before trying again.",
                ),
            ).toBeVisible(),
        );
        expect(submitButton()).toBeDisabled();
    });

    it("asks the user to prove they are human when the server demands it", async () => {
        const user = userEvent.setup();
        appConfig.set({ captcha_provider: "altcha" });
        vi.mocked(authUser).mockRejectedValue(new CaptchaRequiredError("captcha required"));
        render(LoginFormComponent);

        // The check only shows up once asked for: it is not there at first.
        expect(screen.queryByText(/Are you a human/)).not.toBeInTheDocument();

        await fillIn(user);
        await user.click(submitButton());

        await waitFor(() => expect(screen.getByText(/Are you a human/)).toBeVisible());
    });

    it("tells why the password cannot be recovered on a server without mail", async () => {
        const user = userEvent.setup();
        appConfig.set({ no_mail: true });
        render(LoginFormComponent);

        await user.click(screen.getByRole("button", { name: "Forgotten password?" }));

        expect(goto).not.toHaveBeenCalled();
        expect(get(toasts)[0].title).toBe("Password Recovery Unavailable");
    });

    it("offers to recover the password when the server sends mail", async () => {
        const user = userEvent.setup();
        render(LoginFormComponent);

        await user.click(screen.getByRole("button", { name: "Forgotten password?" }));

        expect(goto).toHaveBeenCalledWith("/forgotten-password");
    });

    it("offers the configured single sign-on provider, keeping the page to come back to", () => {
        appConfig.set({ oidc_configured: true });
        pageState.url = new URL("http://localhost/login?next=%2Fdomains");
        render(LoginFormComponent);

        return waitFor(() =>
            expect(screen.getByRole("link", { name: /Sign in with gitlab.com/ })).toHaveAttribute(
                "href",
                "/auth/oidc?next=%2Fdomains",
            ),
        );
    });

    it("keeps the single sign-on button out of the way when nothing is configured", () => {
        render(LoginFormComponent);

        expect(screen.queryByRole("link")).not.toBeInTheDocument();
    });
});

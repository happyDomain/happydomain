// This file is part of the happyDomain (R) project.
// Copyright (c) 2022-2026 happyDomain
// Authors: Pierre-Olivier Mercier, et al.
//
// This program is offered under a commercial and under the AGPL license.
// For commercial licensing, contact us at <contact@happydomain.org>.
//
// For AGPL licensing:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import { render, screen, waitFor } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { goto } from "$app/navigation";
import { registerUser } from "$lib/api/user";
import { appConfig } from "$lib/stores/config";
import { toasts } from "$lib/stores/toasts";
import { loadTranslations } from "$lib/translations";
import SignUpFormComponent from "./SignUpForm.svelte";

// Leaving the page is what a successful registration does, and jsdom has
// nowhere to go: the calls are recorded instead.
vi.mock("$app/navigation", () => ({ goto: vi.fn() }));

// The network lives here: the tests decide what the backend answers.
vi.mock("$lib/api/user", () => ({ registerUser: vi.fn() }));

// Without this, every label reads back as its translation key.
beforeAll(async () => {
    await loadTranslations("en", "/");
});

beforeEach(() => {
    vi.mocked(registerUser).mockResolvedValue(true);
    appConfig.set({});
});

afterEach(() => {
    vi.clearAllMocks();
    for (const toast of get(toasts)) toasts.dismiss(toast.id);
});

/** A password the strength check is happy with. */
const STRONG = "Str0ng!password";

/** Fill in the whole form the way someone signing up would. */
async function fillIn(
    user: ReturnType<typeof userEvent.setup>,
    {
        email = "me@example.com",
        password = STRONG,
        confirmation = password,
    }: { email?: string; password?: string; confirmation?: string } = {},
) {
    await user.type(screen.getByLabelText("Email address"), email);
    await user.type(screen.getByLabelText("Password"), password);
    await user.type(screen.getByLabelText("Password confirmation"), confirmation);
    // The fields only make their mind up once left, and the last one has to be
    // left before the form is sent.
    await user.tab();
}

function submitButton() {
    return screen.getByRole("button", { name: "Sign up!" });
}

describe("sign up form", () => {
    it("asks for an address, a password and its confirmation", () => {
        render(SignUpFormComponent);

        expect(screen.getByLabelText("Email address")).toHaveAttribute("type", "email");
        expect(screen.getByLabelText("Password")).toHaveAttribute("type", "password");
        expect(screen.getByLabelText("Password confirmation")).toHaveAttribute("type", "password");
    });

    it("lets the browser offer to store the new password", () => {
        render(SignUpFormComponent);

        expect(screen.getByLabelText("Email address")).toHaveAttribute("autocomplete", "username");
        expect(screen.getByLabelText("Password")).toHaveAttribute("autocomplete", "new-password");
    });

    it("says what the address will be used for", () => {
        render(SignUpFormComponent);

        expect(
            screen.getByText(/We'll use your address to identify you on this platform/),
        ).toBeVisible();
    });

    it("registers the account that was described, in the language being read", async () => {
        const user = userEvent.setup();
        render(SignUpFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        expect(registerUser).toHaveBeenCalledWith({
            email: "me@example.com",
            password: STRONG,
            wantReceiveUpdate: false,
            lang: "en",
        });
    });

    it("carries the wish to hear about the news", async () => {
        const user = userEvent.setup();
        render(SignUpFormComponent);

        await fillIn(user);
        await user.click(
            screen.getByRole("checkbox", {
                name: "Keep me informed of future big improvements",
            }),
        );
        await user.click(submitButton());

        expect(registerUser).toHaveBeenCalledWith(
            expect.objectContaining({ wantReceiveUpdate: true }),
        );
    });

    it("sends the user to the login page once registered", async () => {
        const user = userEvent.setup();
        render(SignUpFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        await waitFor(() => expect(goto).toHaveBeenCalledWith("/login"));
        expect(get(toasts)[0].message).toBe(
            "Please check your inbox in order to validate your e-mail address.",
        );
    });

    it("tells the user to log in right away when the server sends no mail", async () => {
        const user = userEvent.setup();
        appConfig.set({ no_mail: true });
        render(SignUpFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        await waitFor(() => expect(get(toasts)).toHaveLength(1));
        expect(get(toasts)[0].message).toBe("You can now login with your credentials.");
    });

    it("refuses a password too weak to protect the domains behind it", async () => {
        const user = userEvent.setup();
        render(SignUpFormComponent);

        await fillIn(user, { password: "short" });

        expect(
            screen.getByText(
                "Password needs to be stronger: at least 8 characters with numbers, lower case and upper case characters.",
            ),
        ).toBeVisible();

        await user.click(submitButton());

        expect(registerUser).not.toHaveBeenCalled();
    });

    it("refuses a confirmation that does not match", async () => {
        const user = userEvent.setup();
        render(SignUpFormComponent);

        await fillIn(user, { confirmation: STRONG + "typo" });

        expect(screen.getByText("Password and its confirmation doesn't match.")).toBeVisible();

        await user.click(submitButton());

        expect(registerUser).not.toHaveBeenCalled();
    });

    it("refuses an address without an @", async () => {
        const user = userEvent.setup();
        render(SignUpFormComponent);

        await fillIn(user, { email: "not-an-address" });
        await user.click(submitButton());

        expect(screen.getByText("A valid email address is required")).toBeVisible();
        expect(registerUser).not.toHaveBeenCalled();
    });

    it("keeps an empty form to itself", async () => {
        const user = userEvent.setup();
        render(SignUpFormComponent);

        await user.click(submitButton());

        expect(registerUser).not.toHaveBeenCalled();
    });

    it("reports a refused registration and lets the user try again", async () => {
        const user = userEvent.setup();
        vi.mocked(registerUser).mockRejectedValue(new Error("Email address already used"));
        render(SignUpFormComponent);

        await fillIn(user);
        await user.click(submitButton());

        await waitFor(() => expect(get(toasts)).toHaveLength(1));
        expect(get(toasts)[0].title).toBe("Registration problem");
        expect(goto).not.toHaveBeenCalled();
        expect(submitButton()).toBeEnabled();
        expect(screen.getByLabelText("Email address")).toHaveValue("me@example.com");
    });

    it("asks the user to prove they are human when the server is set up for it", () => {
        appConfig.set({ captcha_provider: "altcha" });
        render(SignUpFormComponent);

        expect(screen.getByText(/Are you a human/)).toBeVisible();
    });

    it("leaves the human check out when the server does not ask for one", () => {
        render(SignUpFormComponent);

        expect(screen.queryByText(/Are you a human/)).not.toBeInTheDocument();
    });
});

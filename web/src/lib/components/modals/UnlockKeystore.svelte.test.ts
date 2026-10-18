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

import { render, screen } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { get } from "svelte/store";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

// The network lives here: the tests decide what the backend answers.
const api = vi.hoisted(() => ({ unlockKeystore: vi.fn() }));
vi.mock("$lib/api/keystores", () => api);
vi.mock("$lib/stores/keystores", () => ({ refreshKeystores: vi.fn(async () => []) }));

import { requestUnlock, settleUnlock, unlockRequest } from "$lib/stores/unlock";
import { loadTranslations } from "$lib/translations";
import UnlockKeystore from "./UnlockKeystore.svelte";

const form = [{ id: "passphrase", type: "string", label: "Passphrase" }];

beforeAll(async () => {
    await loadTranslations("en", "/");
});

beforeEach(() => {
    api.unlockKeystore.mockReset();
    while (get(unlockRequest)) settleUnlock(false);
});

describe("UnlockKeystore", () => {
    it("sends the material, then lets the waiting request go on", async () => {
        const user = userEvent.setup();
        api.unlockKeystore.mockResolvedValue(true);
        render(UnlockKeystore);
        const done = requestUnlock({ ref: "k", name: "Mine", form });

        await user.type(await screen.findByLabelText("Passphrase"), "s3cret");
        await user.click(screen.getByRole("button", { name: /unlock/i }));

        await expect(done).resolves.toBe(true);
        expect(api.unlockKeystore).toHaveBeenCalledWith("k", { passphrase: "s3cret" });
    });

    it("stays open with the error when the material is refused", async () => {
        const user = userEvent.setup();
        api.unlockKeystore.mockRejectedValue(new Error("wrong passphrase"));
        render(UnlockKeystore);
        requestUnlock({ ref: "k", name: "Mine", form });

        await user.click(await screen.findByRole("button", { name: /unlock/i }));

        expect(await screen.findByText("wrong passphrase")).toBeTruthy();
        expect(get(unlockRequest)?.locked.ref).toBe("k");
    });
});
